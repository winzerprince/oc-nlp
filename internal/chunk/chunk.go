package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

type Chunk struct {
	ID     string `json:"id"`
	Index  int    `json:"index"`
	Text   string `json:"text"`
	Start  int    `json:"start"`  // rune offset
	End    int    `json:"end"`    // rune offset
	Source string `json:"source"` // source sha or path
}

type Options struct {
	TargetRunes int
	OverlapRunes int
	MinRunes    int
	Dedupe      bool
}

func DefaultOptions() Options {
	return Options{TargetRunes: 900, OverlapRunes: 180, MinRunes: 120, Dedupe: true}
}

func normalizeWS(s string) string {
	// collapse whitespace to single spaces to keep chunk boundaries stable-ish
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func hashText(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// SplitRunes chunks text using rune counts with overlap.
// It tries to break on spaces near the target length when possible.
func SplitRunes(text, source string, opt Options) []Chunk {
	if opt.TargetRunes <= 0 {
		opt = DefaultOptions()
	}
	norm := normalizeWS(text)
	r := []rune(norm)
	if len(r) == 0 {
		return nil
	}

	step := opt.TargetRunes - opt.OverlapRunes
	if step <= 0 {
		step = opt.TargetRunes
	}

	seen := map[string]bool{}
	out := make([]Chunk, 0)
	idx := 0
	for start := 0; start < len(r); start += step {
		end := start + opt.TargetRunes
		if end > len(r) {
			end = len(r)
		}

		// prefer breaking at last space before end (within a window)
		if end < len(r) {
			window := 80
			best := -1
			for i := end; i > start && i > end-window; i-- {
				if unicode.IsSpace(r[i-1]) {
					best = i
					break
				}
			}
			if best != -1 {
				end = best
			}
		}

		chunkText := strings.TrimSpace(string(r[start:end]))
		if len([]rune(chunkText)) < opt.MinRunes {
			break
		}

		id := hashText(source + "\n" + chunkText)
		if opt.Dedupe {
			if seen[id] {
				continue
			}
			seen[id] = true
		}
		out = append(out, Chunk{ID: id, Index: idx, Text: chunkText, Start: start, End: end, Source: source})
		idx++

		if end >= len(r) {
			break
		}
	}
	return out
}
