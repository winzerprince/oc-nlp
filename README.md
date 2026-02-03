# oc-nlp

**oc-nlp** is a Go-first “train-on-your-docs” NLP playground.

It does **not** attempt to train a full LLM from scratch (that needs serious GPUs/time). Instead, it builds a *document model* you can create and keep training over time:

- ingest PDFs / text / folders
- clean + chunk text
- embed chunks (default: **Ollama**) and build a searchable index
- chat with a selected model using retrieval (RAG)

The UI is designed to be **educational**: it shows the pipeline stages, retrieved passages, similarity scores, and the prompt that gets assembled.

## Requirements

- Go 1.22+
- (Recommended) **Ollama** running locally: <https://ollama.com>

## Quickstart

### 1) Run the server

```bash
go run ./cmd/ocnlp server
```

Open: http://127.0.0.1:8090

### 2) Create and train a “model”

A “model” here is a versioned index + metadata under `.ocnlp/models/<name>`.

From the UI:
- Create model
- Add sources (PDF/text/folders)
- Build index
- Chat

## CLI

```bash
# list models
ocnlp models

# create
ocnlp model create mybooks

# ingest a folder
ocnlp ingest mybooks --path ~/Books

# build index (embeddings)
ocnlp build mybooks

# chat
ocnlp chat mybooks
```

## Architecture (high-level)

1. **Ingest**: PDF/text → normalized text
2. **Chunk**: split into overlapping chunks
3. **Embed**: embed each chunk into a vector
4. **Index**: store vectors + metadata, cosine search
5. **Chat**: retrieve top-K chunks → assemble prompt → generate answer

## Project status

Scaffolded; core index + educational UI in progress.
