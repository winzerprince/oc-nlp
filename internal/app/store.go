package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
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
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
