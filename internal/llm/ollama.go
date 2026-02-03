package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
)

type Config struct {
	Host  string // http://localhost:11434
	Model string // llama3.2:3b etc
}

func DefaultConfig() Config {
	return Config{Host: "http://localhost:11434", Model: "llama3.2:3b"}
}

type Client struct {
	cfg    Config
	client *api.Client
}

func NewClient(cfg Config) (*Client, error) {
	var c *api.Client
	if cfg.Host != "" {
		u, err := url.Parse(cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("parse ollama host: %w", err)
		}
		c = api.NewClient(u, http.DefaultClient)
	} else {
		var err error
		c, err = api.ClientFromEnvironment()
		if err != nil {
			return nil, fmt.Errorf("create ollama client: %w", err)
		}
	}
	return &Client{cfg: cfg, client: c}, nil
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	req := &api.GenerateRequest{Model: c.cfg.Model, Prompt: prompt, Stream: new(bool)}
	*req.Stream = false
	var out string
	err := c.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		out = resp.Response
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("ollama generate: %w", err)
	}
	return out, nil
}
