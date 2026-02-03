package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/winzerprince/oc-nlp/internal/app"
	"github.com/winzerprince/oc-nlp/internal/chat"
	"github.com/winzerprince/oc-nlp/internal/embedding"
)

//go:embed templates/*.html
var templatesFS embed.FS

type App struct {
	DataDir string
	T       *template.Template
	Store   *app.Store
}

func Run(addr, dataDir string) error {
	t, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return err
	}

	app := &App{DataDir: dataDir, T: t, Store: app.NewStore(dataDir)}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/models/create", app.handleCreateModel)
	mux.HandleFunc("/models/", app.handleModelDetail)
	mux.HandleFunc("/api/chat", app.handleChat)

	srv := &http.Server{Addr: addr, Handler: mux}
	fmt.Println("oc-nlp server:", "http://"+addr)
	return srv.ListenAndServe()
}

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	models, _ := a.Store.ListModels()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.T.ExecuteTemplate(w, "home.html", map[string]any{
		"Title":  "oc-nlp",
		"Models": models,
	})
}

func (a *App) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	name := r.FormValue("name")
	_, err := a.Store.CreateModel(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleModelDetail(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/models/"):]
	if name == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	model, err := a.Store.GetModel(name)
	if err != nil {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.T.ExecuteTemplate(w, "chat.html", map[string]any{
		"Title": "Chat - " + name,
		"Model": model,
	})
}

func (a *App) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
		Query string `json:"query"`
		TopK  int    `json:"topK"`
		LLM   string `json:"llm"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.TopK <= 0 {
		req.TopK = 3
	}
	if req.LLM == "" {
		req.LLM = "llama3.2:1b"
	}

	// Load index
	idx, err := a.Store.LoadIndex(req.Model)
	if err != nil {
		http.Error(w, "index not found (run build first)", http.StatusBadRequest)
		return
	}

	// Create client and RAG
	client, err := embedding.NewOllamaClient(embedding.DefaultOllamaURL, embedding.DefaultEmbeddingModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rag := chat.NewRAG(idx, client, req.TopK)

	// Perform chat
	ctx := context.Background()
	result, err := rag.Chat(ctx, req.Query, req.LLM)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
