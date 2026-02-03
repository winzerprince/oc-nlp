.PHONY: test build run

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/ocnlp ./cmd/ocnlp

run:
	go run ./cmd/ocnlp server
