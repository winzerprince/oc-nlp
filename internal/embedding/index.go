package embedding

import (
	"encoding/json"
	"math"
	"os"
	"sort"
)

// Index stores chunks with their embeddings and provides similarity search
type Index struct {
	Chunks []Chunk `json:"chunks"`
}

// SearchResult represents a chunk with its similarity score
type SearchResult struct {
	Chunk Chunk   `json:"chunk"`
	Score float32 `json:"score"`
}

// NewIndex creates a new empty index
func NewIndex() *Index {
	return &Index{Chunks: []Chunk{}}
}

// Add adds a chunk to the index
func (idx *Index) Add(chunk Chunk) {
	idx.Chunks = append(idx.Chunks, chunk)
}

// Save saves the index to a file
func (idx *Index) Save(path string) error {
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Load loads an index from a file
func LoadIndex(path string) (*Index, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// Search finds the top-K most similar chunks to the query vector
func (idx *Index) Search(queryVec []float32, topK int) []SearchResult {
	if topK <= 0 {
		topK = 5
	}

	results := make([]SearchResult, 0, len(idx.Chunks))
	for _, chunk := range idx.Chunks {
		if len(chunk.Vector) == 0 {
			continue
		}
		score := cosineSimilarity(queryVec, chunk.Vector)
		results = append(results, SearchResult{Chunk: chunk, Score: score})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// cosineSimilarity computes cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// Stats returns statistics about the index
func (idx *Index) Stats() map[string]interface{} {
	return map[string]interface{}{
		"chunks": len(idx.Chunks),
	}
}
