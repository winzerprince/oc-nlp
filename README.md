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
ocnlp ingest --path ~/Books mybooks

# build index (embeddings)
# This generates embeddings using Ollama and builds the vector index
ocnlp build mybooks

# search the index
ocnlp search --query "what is machine learning?" --k 5 mybooks

# configure Ollama (optional)
ocnlp build mybooks --host http://localhost:11434 --model nomic-embed-text

# chat (coming soon)
ocnlp chat mybooks
```

## Architecture (high-level)

1. **Ingest**: PDF/text → normalized text
2. **Chunk**: split into overlapping chunks (100 words with 20 word overlap)
3. **Embed**: embed each chunk into a vector using Ollama
4. **Index**: store vectors + metadata on disk with cosine similarity search
5. **Chat**: retrieve top-K chunks → assemble prompt → generate answer (coming soon)

### Vector Index

The local vector index provides:
- **Ollama embeddings**: Uses Ollama's embedding API (default: `nomic-embed-text` model)
- **Disk persistence**: Vectors stored as JSON in `.ocnlp/models/<name>/index.json`
- **Cosine similarity search**: Fast in-memory similarity computation
- **Top-K retrieval**: Returns top results with similarity scores

## Project status

Scaffolded; core index + educational UI in progress.

**Completed:**
- ✅ Ollama embeddings integration
- ✅ Local vector index with cosine similarity
- ✅ CLI commands for build and search
- ✅ Disk persistence for vectors

**In Progress:**
- 🚧 Educational UI
- 🚧 Chat/RAG functionality
