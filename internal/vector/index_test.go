package vector

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		a       []float64
		b       []float64
		want    float64
		wantErr bool
	}{
		{
			name: "identical vectors",
			a:    []float64{1, 0, 0},
			b:    []float64{1, 0, 0},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float64{1, 0, 0},
			b:    []float64{0, 1, 0},
			want: 0.0,
		},
		{
			name: "opposite vectors",
			a:    []float64{1, 0, 0},
			b:    []float64{-1, 0, 0},
			want: -1.0,
		},
		{
			name: "similar vectors",
			a:    []float64{1, 2, 3},
			b:    []float64{2, 4, 6},
			want: 1.0,
		},
		{
			name:    "different dimensions",
			a:       []float64{1, 2},
			b:       []float64{1, 2, 3},
			wantErr: true,
		},
		{
			name:    "zero vector",
			a:       []float64{0, 0, 0},
			b:       []float64{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CosineSimilarity(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("CosineSimilarity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("CosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndex_AddAndCount(t *testing.T) {
	idx := NewIndex()
	
	if idx.Count() != 0 {
		t.Errorf("new index should have 0 documents, got %d", idx.Count())
	}
	
	doc1 := Document{
		ID:        "doc1",
		Text:      "hello world",
		Embedding: []float64{1, 0, 0},
	}
	idx.Add(doc1)
	
	if idx.Count() != 1 {
		t.Errorf("after adding 1 doc, count should be 1, got %d", idx.Count())
	}
	
	doc2 := Document{
		ID:        "doc2",
		Text:      "foo bar",
		Embedding: []float64{0, 1, 0},
	}
	idx.Add(doc2)
	
	if idx.Count() != 2 {
		t.Errorf("after adding 2 docs, count should be 2, got %d", idx.Count())
	}
}

func TestIndex_Search(t *testing.T) {
	idx := NewIndex()
	
	// Add some test documents
	docs := []Document{
		{ID: "doc1", Text: "hello world", Embedding: []float64{1, 0, 0}},
		{ID: "doc2", Text: "foo bar", Embedding: []float64{0, 1, 0}},
		{ID: "doc3", Text: "hello foo", Embedding: []float64{0.7, 0.7, 0}},
	}
	
	for _, doc := range docs {
		idx.Add(doc)
	}
	
	t.Run("search returns top-k results", func(t *testing.T) {
		query := []float64{1, 0, 0}
		results, err := idx.Search(query, 2)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		
		// First result should be doc1 (perfect match)
		if results[0].Document.ID != "doc1" {
			t.Errorf("expected first result to be doc1, got %s", results[0].Document.ID)
		}
		if math.Abs(results[0].Score-1.0) > 1e-9 {
			t.Errorf("expected score 1.0 for perfect match, got %v", results[0].Score)
		}
		
		// Results should be sorted by score (descending)
		if len(results) > 1 && results[0].Score < results[1].Score {
			t.Error("results not sorted by score descending")
		}
	})
	
	t.Run("search with topK larger than index size", func(t *testing.T) {
		query := []float64{1, 0, 0}
		results, err := idx.Search(query, 100)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		
		if len(results) != 3 {
			t.Errorf("expected all 3 results, got %d", len(results))
		}
	})
	
	t.Run("search on empty index", func(t *testing.T) {
		emptyIdx := NewIndex()
		query := []float64{1, 0, 0}
		results, err := emptyIdx.Search(query, 5)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		
		if len(results) != 0 {
			t.Errorf("expected 0 results from empty index, got %d", len(results))
		}
	})
}

func TestIndex_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test_index.json")
	
	// Create and populate index
	idx := NewIndex()
	docs := []Document{
		{
			ID:        "doc1",
			Text:      "hello world",
			Embedding: []float64{1, 0, 0},
			Metadata:  Metadata{"source": "test.txt"},
		},
		{
			ID:        "doc2",
			Text:      "foo bar",
			Embedding: []float64{0, 1, 0},
		},
	}
	
	for _, doc := range docs {
		idx.Add(doc)
	}
	
	// Save index
	if err := idx.Save(indexPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	
	// Check file exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("index file not created")
	}
	
	// Load index
	loadedIdx, err := Load(indexPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	// Verify loaded index
	if loadedIdx.Count() != idx.Count() {
		t.Errorf("loaded index count = %d, want %d", loadedIdx.Count(), idx.Count())
	}
	
	if len(loadedIdx.Documents) != len(idx.Documents) {
		t.Errorf("loaded documents count = %d, want %d", len(loadedIdx.Documents), len(idx.Documents))
	}
	
	// Verify specific document
	if loadedIdx.Documents[0].ID != "doc1" {
		t.Errorf("first doc ID = %s, want doc1", loadedIdx.Documents[0].ID)
	}
	if loadedIdx.Documents[0].Text != "hello world" {
		t.Errorf("first doc text = %s, want 'hello world'", loadedIdx.Documents[0].Text)
	}
	
	// Verify metadata
	if meta, ok := loadedIdx.Documents[0].Metadata["source"]; !ok || meta != "test.txt" {
		t.Errorf("metadata not preserved correctly")
	}
}

func TestIndex_SearchWithMetadata(t *testing.T) {
	idx := NewIndex()
	
	doc := Document{
		ID:        "doc1",
		Text:      "test document",
		Embedding: []float64{1, 0, 0},
		Metadata: Metadata{
			"source": "test.txt",
			"chunk":  1,
		},
	}
	idx.Add(doc)
	
	results, err := idx.Search([]float64{1, 0, 0}, 1)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	
	result := results[0]
	if result.Document.Metadata["source"] != "test.txt" {
		t.Error("metadata not preserved in search results")
	}
}
