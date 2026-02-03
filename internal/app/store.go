package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/winzerprince/oc-nlp/internal/embeddings"
	"github.com/winzerprince/oc-nlp/internal/ingest"
	"github.com/winzerprince/oc-nlp/internal/vector"
)

type Store struct {
	DataDir string
}

func NewStore(dataDir string) *Store {
	return &Store{DataDir: dataDir}
}

var reName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type ModelStats struct {
	Chunks     int `json:"chunks"`
	Embeddings int `json:"embeddings"`
}

type ModelMeta struct {
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Stats     ModelStats `json:"stats"`
}

func (s *Store) modelsDir() string {
	return filepath.Join(s.DataDir, "models")
}

func (s *Store) modelDir(name string) string {
	return filepath.Join(s.modelsDir(), name)
}

func (s *Store) metaPath(name string) string {
	return filepath.Join(s.modelDir(name), "model.json")
}

func (s *Store) CreateModel(name string) (*ModelMeta, error) {
	if !reName.MatchString(name) {
		return nil, errors.New("invalid model name (use letters/numbers/_/-)")
	}
	if err := os.MkdirAll(s.modelDir(name), 0o755); err != nil {
		return nil, err
	}
	mp := s.metaPath(name)
	if _, err := os.Stat(mp); err == nil {
		return nil, errors.New("model already exists")
	}
	now := time.Now().UTC()
	m := &ModelMeta{Name: name, CreatedAt: now, UpdatedAt: now, Stats: ModelStats{Chunks: 0}}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(mp, b, 0o644); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) ListModels() ([]ModelMeta, error) {
	if err := os.MkdirAll(s.modelsDir(), 0o755); err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(s.modelsDir())
	if err != nil {
		return nil, err
	}
	out := make([]ModelMeta, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		b, err := os.ReadFile(s.metaPath(name))
		if err != nil {
			continue
		}
		var m ModelMeta
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (s *Store) GetModel(name string) (*ModelMeta, error) {
	b, err := os.ReadFile(s.metaPath(name))
	if err != nil {
		return nil, err
	}
	var m ModelMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

type SourcesManifest struct {
	Model   string          `json:"model"`
	Sources []ingest.Source `json:"sources"`
}

func (s *Store) sourcesDir(model string) string {
	return filepath.Join(s.modelDir(model), "sources")
}

func (s *Store) manifestPath(model string) string {
	return filepath.Join(s.modelDir(model), "sources.json")
}

func (s *Store) IngestSources(model, path string) error {
	paths, err := ingest.WalkPaths(path)
	if err != nil {
		return err
	}
	out := SourcesManifest{Model: model}
	for _, p := range paths {
		kind, text, sum, err := ingest.ExtractText(p)
		if err != nil {
			continue
		}
		dest := filepath.Join(s.sourcesDir(model), sum+".txt")
		if err := os.MkdirAll(s.sourcesDir(model), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, []byte(text), 0o644); err != nil {
			return err
		}
		out.Sources = append(out.Sources, ingest.Source{Path: p, Kind: kind, SHA256: sum, TextPath: dest})
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	if err := os.WriteFile(s.manifestPath(model), b, 0o644); err != nil {
		return err
	}
	// update stats
	meta, err := s.GetModel(model)
	if err == nil {
		meta.Stats.Chunks = meta.Stats.Chunks // unchanged; chunking comes later
		meta.UpdatedAt = time.Now().UTC()
		mb, _ := json.MarshalIndent(meta, "", "  ")
		_ = os.WriteFile(s.metaPath(model), mb, 0o644)
	}
	return nil
}

func (s *Store) indexPath(model string) string {
	return filepath.Join(s.modelDir(model), "index.json")
}

// simpleChunk performs simple chunking of text into fixed-size overlapping chunks
func simpleChunk(text string, chunkSize, overlap int) []string {
	if chunkSize <= 0 {
		return []string{}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	chunks := make([]string, 0)
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}

	for i := 0; i < len(words); i += step {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
		if end >= len(words) {
			break
		}
	}

	return chunks
}

// BuildIndex builds the vector index for a model using Ollama embeddings
func (s *Store) BuildIndex(ctx context.Context, model string, cfg embeddings.Config) error {
	// Get sources manifest
	manifest, err := s.loadSourcesManifest(model)
	if err != nil {
		return fmt.Errorf("load sources: %w", err)
	}

	if len(manifest.Sources) == 0 {
		return errors.New("no sources to index")
	}

	// Create embeddings client
	embClient, err := embeddings.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("create embeddings client: %w", err)
	}

	// Create new index
	idx := vector.NewIndex()

	// Process each source
	docID := 0
	for _, src := range manifest.Sources {
		// Read text
		text, err := os.ReadFile(src.TextPath)
		if err != nil {
			continue
		}

		// Simple chunking (100 words with 20 word overlap)
		chunks := simpleChunk(string(text), 100, 20)

		// Generate embeddings for each chunk
		for chunkIdx, chunk := range chunks {
			if strings.TrimSpace(chunk) == "" {
				continue
			}

			embedding, err := embClient.Embed(ctx, chunk)
			if err != nil {
				return fmt.Errorf("embed chunk %d: %w", chunkIdx, err)
			}

			docID++
			doc := vector.Document{
				ID:        fmt.Sprintf("doc_%d", docID),
				Text:      chunk,
				Embedding: embedding,
				Metadata: vector.Metadata{
					"source":      src.Path,
					"chunkIdx":    chunkIdx,
					"totalChunks": len(chunks),
				},
			}
			idx.Add(doc)
		}
	}

	// Save index
	if err := idx.Save(s.indexPath(model)); err != nil {
		return fmt.Errorf("save index: %w", err)
	}

	// Update model stats
	meta, err := s.GetModel(model)
	if err != nil {
		return fmt.Errorf("get model for stats update: %w", err)
	}
	meta.Stats.Embeddings = idx.Count()
	meta.UpdatedAt = time.Now().UTC()
	mb, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal model metadata: %w", err)
	}
	if err := os.WriteFile(s.metaPath(model), mb, 0o644); err != nil {
		return fmt.Errorf("write model metadata: %w", err)
	}

	return nil
}

// SearchIndex performs a semantic search on the model's index
func (s *Store) SearchIndex(ctx context.Context, model string, query string, topK int, cfg embeddings.Config) ([]vector.SearchResult, error) {
	// Load index
	idx, err := vector.Load(s.indexPath(model))
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}

	// Create embeddings client
	embClient, err := embeddings.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create embeddings client: %w", err)
	}

	// Generate query embedding
	queryEmb, err := embClient.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// Search
	results, err := idx.Search(queryEmb, topK)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return results, nil
}

func (s *Store) loadSourcesManifest(model string) (*SourcesManifest, error) {
	b, err := os.ReadFile(s.manifestPath(model))
	if err != nil {
		return nil, err
	}
	var m SourcesManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// IngestTextSources is kept for compatibility; use IngestSources.
func (s *Store) IngestTextSources(model, path string) error {
	return s.IngestSources(model, path)
}
