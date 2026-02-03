package vector

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// Document represents a document chunk with its embedding
type Document struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Embedding []float64 `json:"embedding"`
	Metadata  Metadata  `json:"metadata,omitempty"`
}

// Metadata holds optional metadata for a document
type Metadata map[string]interface{}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// Index is an in-memory vector index with disk persistence
type Index struct {
	Documents []Document `json:"documents"`
}

// NewIndex creates a new empty vector index
func NewIndex() *Index {
	return &Index{
		Documents: make([]Document, 0),
	}
}

// Add adds a document to the index
func (idx *Index) Add(doc Document) {
	idx.Documents = append(idx.Documents, doc)
}

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions don't match: %d vs %d", len(a), len(b))
	}
	
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 {
		return 0, fmt.Errorf("cannot compute similarity: first vector is zero")
	}
	if normB == 0 {
		return 0, fmt.Errorf("cannot compute similarity: second vector is zero")
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}

// Search performs a cosine similarity search and returns the top-k results
func (idx *Index) Search(queryEmbedding []float64, topK int) ([]SearchResult, error) {
	if len(idx.Documents) == 0 {
		return []SearchResult{}, nil
	}
	
	results := make([]SearchResult, 0, len(idx.Documents))
	
	for _, doc := range idx.Documents {
		score, err := CosineSimilarity(queryEmbedding, doc.Embedding)
		if err != nil {
			// Skip documents with incompatible embeddings
			continue
		}
		results = append(results, SearchResult{
			Document: doc,
			Score:    score,
		})
	}
	
	// Sort by score in descending order (highest similarity first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Return top-k results
	if topK > len(results) {
		topK = len(results)
	}
	
	return results[:topK], nil
}

// Save persists the index to disk
func (idx *Index) Save(path string) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	
	return nil
}

// Load loads an index from disk
func Load(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("unmarshal index: %w", err)
	}
	
	return &idx, nil
}

// Count returns the number of documents in the index
func (idx *Index) Count() int {
	return len(idx.Documents)
}
