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
	"time"

	"github.com/winzerprince/oc-nlp/internal/embedding"
	"github.com/winzerprince/oc-nlp/internal/ingest"
)

type Store struct {
	DataDir string
}

func NewStore(dataDir string) *Store {
	return &Store{DataDir: dataDir}
}

var reName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type ModelStats struct {
	Chunks int `json:"chunks"`
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

func (s *Store) indexPath(model string) string {
	return filepath.Join(s.modelDir(model), "index.json")
}

func (s *Store) IngestTextSources(model, path string) error {
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

// BuildIndex chunks and embeds all sources for a model
func (s *Store) BuildIndex(ctx context.Context, model string) error {
	// Read sources manifest
	manifest, err := s.getSourcesManifest(model)
	if err != nil {
		return fmt.Errorf("failed to read sources: %w", err)
	}

	if len(manifest.Sources) == 0 {
		return errors.New("no sources to index")
	}

	// Create Ollama client
	client, err := embedding.NewOllamaClient("", "")
	if err != nil {
		return fmt.Errorf("failed to create Ollama client: %w", err)
	}

	// Create index
	idx := embedding.NewIndex()

	// Process each source
	for _, src := range manifest.Sources {
		text, err := os.ReadFile(src.TextPath)
		if err != nil {
			continue
		}

		// Chunk the text
		chunks := embedding.ChunkText(string(text), 500, 100)

		// Embed each chunk
		for i, chunkText := range chunks {
			vec, err := client.Embed(ctx, chunkText)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to embed chunk %d from %s: %v\n", i, src.Path, err)
				continue
			}

			chunk := embedding.Chunk{
				Text:     chunkText,
				SourceID: src.SHA256,
				Index:    i,
				Vector:   vec,
			}
			idx.Add(chunk)
		}
	}

	// Save index
	if err := idx.Save(s.indexPath(model)); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	// Update model stats
	meta, err := s.GetModel(model)
	if err == nil {
		meta.Stats.Chunks = len(idx.Chunks)
		meta.UpdatedAt = time.Now().UTC()
		mb, _ := json.MarshalIndent(meta, "", "  ")
		_ = os.WriteFile(s.metaPath(model), mb, 0o644)
	}

	return nil
}

// LoadIndex loads the index for a model
func (s *Store) LoadIndex(model string) (*embedding.Index, error) {
	return embedding.LoadIndex(s.indexPath(model))
}

func (s *Store) getSourcesManifest(model string) (*SourcesManifest, error) {
	b, err := os.ReadFile(s.manifestPath(model))
	if err != nil {
		return nil, err
	}
	var manifest SourcesManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
