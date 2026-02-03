package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/winzerprince/oc-nlp/internal/embedding"
)

// RAG implements Retrieval-Augmented Generation
type RAG struct {
	Index  *embedding.Index
	Client *embedding.OllamaClient
	TopK   int
}

// ChatResult contains all information about a chat interaction
type ChatResult struct {
	Query            string                    `json:"query"`
	RetrievedChunks  []embedding.SearchResult  `json:"retrievedChunks"`
	AssembledPrompt  string                    `json:"assembledPrompt"`
	Answer           string                    `json:"answer"`
}

// NewRAG creates a new RAG instance
func NewRAG(index *embedding.Index, client *embedding.OllamaClient, topK int) *RAG {
	if topK <= 0 {
		topK = 5
	}
	return &RAG{
		Index:  index,
		Client: client,
		TopK:   topK,
	}
}

// Chat performs RAG: retrieve relevant chunks, assemble prompt, generate answer
func (r *RAG) Chat(ctx context.Context, query, model string) (*ChatResult, error) {
	result := &ChatResult{Query: query}

	// 1. Embed the query
	queryVec, err := r.Client.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 2. Retrieve top-K chunks
	result.RetrievedChunks = r.Index.Search(queryVec, r.TopK)

	// 3. Assemble prompt with context
	result.AssembledPrompt = r.assemblePrompt(query, result.RetrievedChunks)

	// 4. Generate answer
	answer, err := r.Client.Generate(ctx, model, result.AssembledPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}
	result.Answer = strings.TrimSpace(answer)

	return result, nil
}

// assemblePrompt creates a prompt with retrieved context
func (r *RAG) assemblePrompt(query string, chunks []embedding.SearchResult) string {
	var b strings.Builder
	
	b.WriteString("You are a helpful assistant. Answer the question based on the following context.\n\n")
	b.WriteString("Context:\n")
	
	for i, chunk := range chunks {
		b.WriteString(fmt.Sprintf("--- Passage %d (score: %.3f) ---\n", i+1, chunk.Score))
		b.WriteString(chunk.Chunk.Text)
		b.WriteString("\n\n")
	}
	
	b.WriteString("Question: ")
	b.WriteString(query)
	b.WriteString("\n\nAnswer:")
	
	return b.String()
}
