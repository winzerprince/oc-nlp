package server

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/*.html
var templatesFS embed.FS

type App struct {
	DataDir string
	T       *template.Template
}

func Run(addr, dataDir string) error {
	t, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return err
	}

	app := &App{DataDir: dataDir, T: t}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleHome)

	srv := &http.Server{Addr: addr, Handler: mux}
	fmt.Println("oc-nlp server:", "http://"+addr)
	return srv.ListenAndServe()
}

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.T.ExecuteTemplate(w, "home.html", map[string]any{
		"Title": "oc-nlp",
	})
}
