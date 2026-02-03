package ingest

import (
	"os"
	"path/filepath"
	"strings"
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

func TestExtractTextPDF(t *testing.T) {
	// Test with the fixture PDF
	p := filepath.Join("testdata", "sample.pdf")
	kind, text, sum, err := ExtractText(p)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "pdf" {
		t.Fatalf("expected kind=pdf, got %q", kind)
	}
	if text == "" {
		t.Fatal("expected non-empty text")
	}
	if !strings.Contains(text, "Hello PDF World") {
		t.Fatalf("expected text to contain 'Hello PDF World', got %q", text)
	}
	if sum == "" {
		t.Fatal("expected non-empty checksum")
	}
}

func TestExtractTextPDFEncrypted(t *testing.T) {
	// Test with encrypted PDF - should fail gracefully
	p := filepath.Join("testdata", "encrypted.pdf")
	_, _, _, err := ExtractText(p)
	if err == nil {
		t.Fatal("expected error for encrypted PDF")
	}
	if !strings.Contains(err.Error(), "encrypted") {
		t.Fatalf("expected error message to contain 'encrypted', got %q", err.Error())
	}
}
