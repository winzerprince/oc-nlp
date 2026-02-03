package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/winzerprince/oc-nlp/internal/app"
	"github.com/winzerprince/oc-nlp/internal/embeddings"
	"github.com/winzerprince/oc-nlp/internal/llm"
	"github.com/winzerprince/oc-nlp/internal/vector"
)

type Retrieved struct {
	Text  string
	Score float64
}

type Result struct {
	Answer          string
	Retrieved       []Retrieved
	AssembledPrompt string
}

func Ask(ctx context.Context, store *app.Store, modelName string, query string, topK int, embCfg embeddings.Config, llmCfg llm.Config) (*Result, error) {
	results, err := store.SearchIndex(ctx, modelName, query, topK, embCfg)
	if err != nil {
		return nil, err
	}

	retrieved := make([]Retrieved, 0, len(results))
	for _, r := range results {
		retrieved = append(retrieved, Retrieved{Text: r.Document.Text, Score: r.Score})
	}

	prompt := assemblePrompt(query, results)
	client, err := llm.NewClient(llmCfg)
	if err != nil {
		return nil, err
	}
	ans, err := client.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &Result{Answer: strings.TrimSpace(ans), Retrieved: retrieved, AssembledPrompt: prompt}, nil
}

func assemblePrompt(query string, results []vector.SearchResult) string {
	var b strings.Builder
	b.WriteString("You are a helpful assistant. Use the provided CONTEXT to answer the QUESTION. If the answer is not in the context, say you don't know.\n\n")
	b.WriteString("CONTEXT:\n")
	for i, r := range results {
		b.WriteString(fmt.Sprintf("[%d] (score=%.4f) %s\n\n", i+1, r.Score, r.Document.Text))
	}
	b.WriteString("QUESTION: " + query + "\n")
	b.WriteString("ANSWER:\n")
	return b.String()
}
