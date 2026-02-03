package server

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/winzerprince/oc-nlp/internal/chat"
	"github.com/winzerprince/oc-nlp/internal/embeddings"
	"github.com/winzerprince/oc-nlp/internal/llm"

	"github.com/winzerprince/oc-nlp/internal/app"
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
	mux.HandleFunc("/chat", app.handleChat)
	mux.HandleFunc("/ingest/path", app.handleIngestPath)
	mux.HandleFunc("/ingest/upload", app.handleIngestUpload)

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

func (a *App) handleChat(w http.ResponseWriter, r *http.Request) {
	model := r.URL.Query().Get("model")
	if model == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	query := ""
	if r.Method == http.MethodPost {
		query = r.FormValue("q")
	}
	var res any
	var errMsg string
	if query != "" {
		ctx := r.Context()
		embCfg := embeddings.DefaultConfig()
		llmCfg := llm.DefaultConfig()
		r, err := chat.Ask(ctx, a.Store, model, query, 4, embCfg, llmCfg)
		if err != nil {
			errMsg = err.Error()
		} else {
			res = r
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.T.ExecuteTemplate(w, "chat.html", map[string]any{
		"Title":  "oc-nlp chat",
		"Model":  model,
		"Query":  query,
		"Result": res,
		"Error":  errMsg,
	})
}

func (a *App) handleIngestPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	model := r.FormValue("model")
	path := r.FormValue("path")
	if model == "" || path == "" {
		http.Error(w, "model and path are required", http.StatusBadRequest)
		return
	}
	if _, err := a.Store.GetModel(model); err != nil {
		http.Error(w, "unknown model: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.Store.IngestSources(model, path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleIngestUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	model := r.FormValue("model")
	if model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}
	if _, err := a.Store.GetModel(model); err != nil {
		http.Error(w, "unknown model: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	uploadDir := filepath.Join(a.DataDir, "models", model, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dest := filepath.Join(uploadDir, filepath.Base(hdr.Filename))
	out, err := os.Create(dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		_ = out.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = out.Close()

	if err := a.Store.IngestSources(model, dest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
