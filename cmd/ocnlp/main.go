package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/winzerprince/oc-nlp/internal/app"
	"github.com/winzerprince/oc-nlp/internal/chat"
	"github.com/winzerprince/oc-nlp/internal/embedding"
	"github.com/winzerprince/oc-nlp/internal/server"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ocnlp <server|models|model|ingest|build|chat>")
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
			fmt.Printf("%s\tchunks=%d\tupdated=%s\n", m.Name, m.Stats.Chunks, m.UpdatedAt)
		}

	case "ingest":
		fs := flag.NewFlagSet("ingest", flag.ExitOnError)
		data := fs.String("data", ".ocnlp", "data directory")
		path := fs.String("path", "", "file or directory to ingest")
		_ = fs.Parse(os.Args[2:])
		args := fs.Args()
		if len(args) < 1 {
			log.Fatal("missing model name")
		}
		if *path == "" {
			log.Fatal("missing --path")
		}
		model := args[0]
		store := app.NewStore(*data)
		if _, err := store.GetModel(model); err != nil {
			log.Fatal(err)
		}
		if err := store.IngestTextSources(model, *path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("ingested into model:", model)

	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		data := fs.String("data", ".ocnlp", "data directory")
		_ = fs.Parse(os.Args[2:])
		args := fs.Args()
		if len(args) < 1 {
			log.Fatal("missing model name")
		}
		model := args[0]
		store := app.NewStore(*data)
		if _, err := store.GetModel(model); err != nil {
			log.Fatal(err)
		}
		fmt.Println("building index for model:", model)
		ctx := context.Background()
		if err := store.BuildIndex(ctx, model); err != nil {
			log.Fatal(err)
		}
		fmt.Println("index built successfully")

	case "chat":
		fs := flag.NewFlagSet("chat", flag.ExitOnError)
		data := fs.String("data", ".ocnlp", "data directory")
		topK := fs.Int("topK", 3, "number of chunks to retrieve")
		model := fs.String("model", "llama3.2:1b", "Ollama model for generation")
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

		// Load index
		idx, err := store.LoadIndex(modelName)
		if err != nil {
			log.Fatal("failed to load index (did you run 'build'?):", err)
		}

		// Create Ollama client
		client, err := embedding.NewOllamaClient(embedding.DefaultOllamaURL, embedding.DefaultEmbeddingModel)
		if err != nil {
			log.Fatal(err)
		}

		// Create RAG
		rag := chat.NewRAG(idx, client, *topK)

		fmt.Printf("Chat with model: %s (type 'exit' to quit)\n", modelName)
		fmt.Printf("Using Ollama model: %s, topK=%d\n\n", *model, *topK)

		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			query := strings.TrimSpace(scanner.Text())
			if query == "" {
				continue
			}
			if query == "exit" || query == "quit" {
				break
			}

			ctx := context.Background()
			result, err := rag.Chat(ctx, query, *model)
			if err != nil {
				log.Printf("error: %v\n", err)
				continue
			}

			// Display results
			fmt.Println("\n=== Retrieved Passages ===")
			for i, chunk := range result.RetrievedChunks {
				fmt.Printf("\n[%d] Score: %.4f\n", i+1, chunk.Score)
				fmt.Printf("%s\n", truncate(chunk.Chunk.Text, 200))
			}

			fmt.Println("\n=== Assembled Prompt ===")
			fmt.Println(truncate(result.AssembledPrompt, 500))

			fmt.Println("\n=== Answer ===")
			fmt.Println(result.Answer)
			fmt.Println()
		}

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

	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(2)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
