package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/winzerprince/oc-nlp/internal/app"
	"github.com/winzerprince/oc-nlp/internal/server"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ocnlp <server|models|model>")
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
