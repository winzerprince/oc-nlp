package embedding

import (
	"context"
	"fmt"
	"strings"

	"github.com/ollama/ollama/api"
)

const DefaultEmbeddingModel = "nomic-embed-text"

// OllamaClient wraps the Ollama API client for embeddings
type OllamaClient struct {
	client *api.Client
	model  string
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(baseURL, model string) (*OllamaClient, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = DefaultEmbeddingModel
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	return &OllamaClient{
		client: client,
		model:  model,
	}, nil
}

// Embed generates an embedding vector for the given text
func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	req := &api.EmbedRequest{
		Model:  c.model,
		Input: text,
	}

	resp, err := c.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert []float64 to []float32
	embedding := make([]float32, len(resp.Embeddings[0]))
	for i, v := range resp.Embeddings[0] {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// Generate generates text using a chat model
func (c *OllamaClient) Generate(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		model = "llama3.2:1b" // default small model
	}

	req := &api.GenerateRequest{
		Model:  model,
		Prompt: prompt,
	}

	var result strings.Builder
	err := c.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		result.WriteString(resp.Response)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	return result.String(), nil
}
