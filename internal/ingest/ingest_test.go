package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	in := "a\r\nb\x00c\r"
	out := NormalizeText(in)
	if out != "a\nbc\n" {
		t.Fatalf("got %q", out)
	}
}

func TestWalkPathsFile(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "a.txt")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := WalkPaths(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != p {
		t.Fatalf("unexpected: %#v", paths)
	}
}

func TestExtractTextUnsupported(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "a.bin")
	if err := os.WriteFile(p, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := ExtractText(p)
	if err == nil {
		t.Fatal("expected error")
	}
}
