package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/winzerprince/oc-nlp/internal/server"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ocnlp <server>")
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
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(2)
	}
}
