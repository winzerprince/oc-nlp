package embeddings

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
)

// Config holds configuration for Ollama embeddings
type Config struct {
	Host  string // e.g., "http://localhost:11434"
	Model string // e.g., "nomic-embed-text"
}

// DefaultConfig returns a default Ollama configuration
func DefaultConfig() Config {
	return Config{
		Host:  "http://localhost:11434",
		Model: "nomic-embed-text",
	}
}

// Client wraps the Ollama API client for generating embeddings
type Client struct {
	cfg    Config
	client *api.Client
}

// NewClient creates a new Ollama embeddings client
func NewClient(cfg Config) (*Client, error) {
	var client *api.Client
	
	if cfg.Host != "" {
		u, err := url.Parse(cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("parse ollama host: %w", err)
		}
		client = api.NewClient(u, http.DefaultClient)
	} else {
		var err error
		client, err = api.ClientFromEnvironment()
		if err != nil {
			return nil, fmt.Errorf("create ollama client from environment (ensure OLLAMA_HOST is set): %w", err)
		}
	}
	
	return &Client{
		cfg:    cfg,
		client: client,
	}, nil
}

// Embed generates an embedding vector for the given text
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	req := &api.EmbedRequest{
		Model: c.cfg.Model,
		Input: text,
	}
	
	resp, err := c.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	
	// Convert []float32 to []float64
	embedding32 := resp.Embeddings[0]
	embedding64 := make([]float64, len(embedding32))
	for i, v := range embedding32 {
		embedding64[i] = float64(v)
	}
	
	return embedding64, nil
}

// EmbedBatch generates embeddings for multiple texts
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	
	for i, text := range texts {
		emb, err := c.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}
	
	return embeddings, nil
}
