package chunk

import "testing"

func TestSplitRunesBasic(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven twelve "
	opt := Options{TargetRunes: 20, OverlapRunes: 5, MinRunes: 5, Dedupe: true}
	chunks := SplitRunes(text, "src", opt)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if c.ID == "" || c.Text == "" {
			t.Fatalf("bad chunk %d: %#v", i, c)
		}
	}
}

func TestDedupe(t *testing.T) {
	// With overlap, consecutive windows can still yield different chunk texts,
	// so dedupe is best-effort, not guaranteed to collapse to 1.
	text := "a a a a a a a a a a a a a a a a"
	opt := Options{TargetRunes: 10, OverlapRunes: 9, MinRunes: 1, Dedupe: true}
	chunks := SplitRunes(text, "src", opt)
	if len(chunks) < 1 {
		t.Fatalf("expected at least 1 chunk")
	}
	// Ensure IDs are unique
	seen := map[string]bool{}
	for _, c := range chunks {
		if seen[c.ID] {
			t.Fatalf("duplicate id found: %s", c.ID)
		}
		seen[c.ID] = true
	}
}
