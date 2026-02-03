package embedding

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexAddAndSearch(t *testing.T) {
	idx := NewIndex()
	
	// Add some chunks with vectors
	chunk1 := Chunk{
		Text:     "machine learning is great",
		SourceID: "src1",
		Index:    0,
		Vector:   []float32{1.0, 0.0, 0.0},
	}
	chunk2 := Chunk{
		Text:     "deep learning is powerful",
		SourceID: "src1",
		Index:    1,
		Vector:   []float32{0.9, 0.1, 0.0},
	}
	chunk3 := Chunk{
		Text:     "cats are cute",
		SourceID: "src2",
		Index:    0,
		Vector:   []float32{0.0, 0.0, 1.0},
	}
	
	idx.Add(chunk1)
	idx.Add(chunk2)
	idx.Add(chunk3)
	
	if len(idx.Chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(idx.Chunks))
	}
	
	// Search for something similar to chunk1
	queryVec := []float32{1.0, 0.0, 0.0}
	results := idx.Search(queryVec, 2)
	
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	
	// First result should be chunk1 (exact match)
	if results[0].Chunk.Text != "machine learning is great" {
		t.Errorf("expected first result to be chunk1, got %q", results[0].Chunk.Text)
	}
	
	// Score should be high
	if results[0].Score < 0.9 {
		t.Errorf("expected high similarity score, got %f", results[0].Score)
	}
}

func TestCosineSimilarity(t *testing.T) {
	// Test identical vectors
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{1.0, 0.0, 0.0}
	sim := cosineSimilarity(a, b)
	if math.Abs(float64(sim)-1.0) > 0.001 {
		t.Errorf("expected similarity of 1.0 for identical vectors, got %f", sim)
	}
	
	// Test orthogonal vectors
	c := []float32{1.0, 0.0, 0.0}
	d := []float32{0.0, 1.0, 0.0}
	sim2 := cosineSimilarity(c, d)
	if math.Abs(float64(sim2)) > 0.001 {
		t.Errorf("expected similarity of 0.0 for orthogonal vectors, got %f", sim2)
	}
}

func TestIndexSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")
	
	// Create and save index
	idx := NewIndex()
	chunk := Chunk{
		Text:     "test chunk",
		SourceID: "src1",
		Index:    0,
		Vector:   []float32{1.0, 2.0, 3.0},
	}
	idx.Add(chunk)
	
	if err := idx.Save(path); err != nil {
		t.Fatal(err)
	}
	
	// Check file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatal("index file not created")
	}
	
	// Load index
	loaded, err := LoadIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(loaded.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(loaded.Chunks))
	}
	
	if loaded.Chunks[0].Text != "test chunk" {
		t.Errorf("chunk text mismatch: got %q", loaded.Chunks[0].Text)
	}
	
	if len(loaded.Chunks[0].Vector) != 3 {
		t.Errorf("expected vector length 3, got %d", len(loaded.Chunks[0].Vector))
	}
}
