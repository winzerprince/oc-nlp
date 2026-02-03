package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/winzerprince/oc-nlp/internal/app"
	"github.com/winzerprince/oc-nlp/internal/embeddings"
	"github.com/winzerprince/oc-nlp/internal/server"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ocnlp <server|models|model|ingest|build|search>")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "server":
		fs := flag.NewFlagSet("server", flag.ExitOnError)
		addr := fs.String("addr", "127.0.0.1:8090", "listen address")
		data := fs.String("data", ".ocnlp", "data directory")
		_ = fs.Parse(os.Args[2:])

		err := server.Run(*addr, *data)
		if err != nil {
			log.Fatal(err)
		}

	case "models":
		fs := flag.NewFlagSet("models", flag.ExitOnError)
		data := fs.String("data", ".ocnlp", "data directory")
		_ = fs.Parse(os.Args[2:])

		store := app.NewStore(*data)
		models, err := store.ListModels()
		if err != nil {
			log.Fatal(err)
		}
		if len(models) == 0 {
			fmt.Println("(no models)")
			return
		}
		for _, m := range models {
			fmt.Printf("%s\tchunks=%d\tembeddings=%d\tupdated=%s\n", m.Name, m.Stats.Chunks, m.Stats.Embeddings, m.UpdatedAt)
		}

	case "ingest":
		// We allow flags to appear after positional args (e.g. `ocnlp ingest mybooks --path ./docs`).
		// The stdlib flag package does not support interspersed flags, so we manually extract known flags.
		data := ".ocnlp"
		path := ""
		args := make([]string, 0, len(os.Args[2:]))
		for i := 2; i < len(os.Args); i++ {
			a := os.Args[i]
			switch a {
			case "--data":
				if i+1 >= len(os.Args) {
					log.Fatal("--data requires a value")
				}
				data = os.Args[i+1]
				i++
			case "--path":
				if i+1 >= len(os.Args) {
					log.Fatal("--path requires a value")
				}
				path = os.Args[i+1]
				i++
			default:
				args = append(args, a)
			}
		}
		if len(args) < 1 {
			log.Fatal("usage: ocnlp ingest <model> --path <file|dir> [--data .ocnlp]")
		}
		model := args[0]
		// ignore any extra positional args (often introduced by shell completion)

		if path == "" {
			log.Fatal("missing --path")
		}
		store := app.NewStore(data)
		if _, err := store.GetModel(model); err != nil {
			log.Fatal(err)
		}
		if err := store.IngestSources(model, path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("ingested into model:", model)

	case "model":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: ocnlp model <create> <name> [--data .ocnlp]")
			os.Exit(2)
		}
		sub := os.Args[2]
		switch sub {
		case "create":
			fs := flag.NewFlagSet("model create", flag.ExitOnError)
			data := fs.String("data", ".ocnlp", "data directory")
			_ = fs.Parse(os.Args[3:])
			args := fs.Args()
			if len(args) < 1 {
				log.Fatal("missing model name")
			}
			name := args[0]
			store := app.NewStore(*data)
			m, err := store.CreateModel(name)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("created model:", m.Name)
		default:
			fmt.Fprintln(os.Stderr, "unknown model subcommand:", sub)
			os.Exit(2)
		}

	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		data := fs.String("data", ".ocnlp", "data directory")
		host := fs.String("host", "http://localhost:11434", "Ollama host")
		model := fs.String("model", "nomic-embed-text", "embedding model")
		_ = fs.Parse(os.Args[2:])
		args := fs.Args()
		if len(args) < 1 {
			log.Fatal("missing model name")
		}
		modelName := args[0]

		store := app.NewStore(*data)
		if _, err := store.GetModel(modelName); err != nil {
			log.Fatal(err)
		}

		cfg := embeddings.Config{
			Host:  *host,
			Model: *model,
		}

		fmt.Printf("Building index for model '%s' using %s on %s...\n", modelName, cfg.Model, cfg.Host)
		ctx := context.Background()
		if err := store.BuildIndex(ctx, modelName, cfg); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Index built successfully")

	case "search":
		fs := flag.NewFlagSet("search", flag.ExitOnError)
		data := fs.String("data", ".ocnlp", "data directory")
		host := fs.String("host", "http://localhost:11434", "Ollama host")
		model := fs.String("model", "nomic-embed-text", "embedding model")
		topK := fs.Int("k", 5, "number of results to return")
		query := fs.String("query", "", "search query")
		_ = fs.Parse(os.Args[2:])
		args := fs.Args()
		if len(args) < 1 {
			log.Fatal("missing model name")
		}
		modelName := args[0]

		if *query == "" {
			log.Fatal("missing --query")
		}

		store := app.NewStore(*data)
		if _, err := store.GetModel(modelName); err != nil {
			log.Fatal(err)
		}

		cfg := embeddings.Config{
			Host:  *host,
			Model: *model,
		}

		ctx := context.Background()
		results, err := store.SearchIndex(ctx, modelName, *query, *topK, cfg)
		if err != nil {
			log.Fatal(err)
		}

		if len(results) == 0 {
			fmt.Println("No results found")
			return
		}

		fmt.Printf("Found %d results:\n\n", len(results))
		for i, r := range results {
			fmt.Printf("=== Result %d (score: %.4f) ===\n", i+1, r.Score)
			fmt.Printf("Text: %s\n", r.Document.Text)
			if source, ok := r.Document.Metadata["source"]; ok {
				fmt.Printf("Source: %v\n", source)
			}
			fmt.Println()
		}

	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(2)
	}
}
