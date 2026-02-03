package embedding

import (
	"strings"
)

// Chunk represents a text chunk with metadata
type Chunk struct {
	Text     string  `json:"text"`
	SourceID string  `json:"sourceId"` // SHA256 of source
	Index    int     `json:"index"`    // chunk index in source
	Vector   []float32 `json:"vector,omitempty"`
}

// ChunkText splits text into overlapping chunks
func ChunkText(text string, chunkSize, overlap int) []string {
	if chunkSize <= 0 {
		chunkSize = 500
	}
	if overlap < 0 || overlap >= chunkSize {
		overlap = chunkSize / 4
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	stride := chunkSize - overlap
	if stride <= 0 {
		stride = 1
	}

	for i := 0; i < len(words); i += stride {
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
