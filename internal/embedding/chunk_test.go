package embedding

import (
	"testing"
)

func TestChunkText(t *testing.T) {
	text := "one two three four five six seven eight nine ten"
	chunks := ChunkText(text, 3, 1)
	
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	
	// First chunk should be "one two three"
	if chunks[0] != "one two three" {
		t.Fatalf("expected 'one two three', got %q", chunks[0])
	}
	
	// With stride of 2, second chunk should be "three four five"
	if len(chunks) > 1 && chunks[1] != "three four five" {
		t.Fatalf("expected 'three four five', got %q", chunks[1])
	}
}

func TestChunkTextEmpty(t *testing.T) {
	chunks := ChunkText("", 500, 100)
	if len(chunks) != 0 {
		t.Fatal("expected no chunks for empty text")
	}
}

func TestChunkTextDefaults(t *testing.T) {
	text := make([]string, 1000)
	for i := range text {
		text[i] = "word"
	}
	input := ""
	for _, w := range text {
		input += w + " "
	}
	
	chunks := ChunkText(input, 0, 0) // use defaults
	if len(chunks) == 0 {
		t.Fatal("expected chunks with default params")
	}
}
