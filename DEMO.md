# End-to-End Demo: Chat with RAG

This document demonstrates the complete workflow from txt files to chat interface.

## Prerequisites

- Go 1.22+
- Ollama running locally (download from https://ollama.com)
- Pull required models:
  ```bash
  ollama pull nomic-embed-text   # for embeddings
  ollama pull llama3.2:1b        # for chat (or any other model)
  ```

## CLI Workflow

### 1. Create a model
```bash
ocnlp model create aibooks
```

### 2. Ingest text files
```bash
ocnlp ingest --path /path/to/docs aibooks
```

### 3. Build the index (chunk and embed)
```bash
ocnlp build aibooks
```

This will:
- Chunk each source text into overlapping segments
- Generate embeddings for each chunk using Ollama
- Build a searchable vector index

### 4. Chat with the model
```bash
ocnlp chat aibooks
```

Example interaction:
```
> What is machine learning?

=== Retrieved Passages ===

[1] Score: 0.8943
Machine learning is a subset of artificial intelligence that focuses on the 
development of algorithms and statistical models that enable computer systems...

[2] Score: 0.7821
There are three main types of machine learning: Supervised Learning, 
Unsupervised Learning, and Reinforcement Learning...

[3] Score: 0.7234
Popular machine learning algorithms include linear regression, decision trees...

=== Assembled Prompt ===
You are a helpful assistant. Answer the question based on the following context.

Context:
--- Passage 1 (score: 0.894) ---
Machine learning is a subset of artificial intelligence...

--- Passage 2 (score: 0.782) ---
There are three main types of machine learning...

Question: What is machine learning?

Answer:

=== Answer ===
Machine learning is a subset of artificial intelligence that enables computer 
systems to improve their performance on tasks through experience. It involves 
developing algorithms and statistical models that learn from data. There are 
three main types: supervised learning (learning from labeled data), unsupervised 
learning (finding patterns in unlabeled data), and reinforcement learning 
(learning through trial and error with rewards).
```

## Web UI Workflow

### 1. Start the server
```bash
ocnlp server
```

Open http://127.0.0.1:8090 in your browser.

### 2. Create a model
- Enter a model name (e.g., "aibooks")
- Click "Create"

### 3. Ingest documents via CLI
```bash
ocnlp ingest --path /path/to/docs aibooks
```

### 4. Build the index via CLI
```bash
ocnlp build aibooks
```

### 5. Chat in the UI
- Click "Chat" next to your model
- Type your question in the text box
- Click "Ask" or press Enter

The educational UI panels will show:
- **Retrieved Passages**: The top-K chunks that matched your query, with similarity scores
- **Assembled Prompt**: The exact prompt sent to the LLM, showing how context is provided
- **Answer**: The generated response from the Ollama model

## Educational Features

The UI is designed to be transparent about how RAG works:

1. **Retrieval Visualization**: See exactly which text chunks were retrieved and their similarity scores
2. **Prompt Assembly**: Understand how the context is formatted for the LLM
3. **Top-K Control**: Adjust how many chunks to retrieve (1-10)
4. **Model Selection**: Choose which Ollama model to use for generation

This helps developers and learners understand:
- How semantic search works
- How context affects LLM responses
- The relationship between retrieval quality and answer quality
- Trade-offs in chunk count and context window usage

## Sample Data

The repository includes sample documents in `/tmp/sample-docs/` for testing:
- `ml-basics.txt`: Introduction to machine learning
- `neural-networks.txt`: Overview of neural networks
- `nlp.txt`: Natural language processing concepts

Use these to quickly test the end-to-end workflow.
