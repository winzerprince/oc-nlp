package ingest

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

var ErrUnsupported = errors.New("unsupported file type")

type Source struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"` // txt for now
	SHA256   string `json:"sha256"`
	TextPath string `json:"textPath"`
}

func WalkPaths(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	var paths []string
	if !info.IsDir() {
		return []string{root}, nil
	}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	return paths, err
}

func NormalizeText(s string) string {
	// minimal normalization: strip NULs and normalize newlines
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func ExtractText(path string) (kind string, text string, sum string, err error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md":
		kind = "text"
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return "", "", "", rerr
		}
		norm := NormalizeText(string(b))
		h := sha256.Sum256([]byte(norm))
		return kind, norm, hex.EncodeToString(h[:]), nil
	case ".pdf":
		kind = "pdf"
		content, rerr := extractPDFText(path)
		if rerr != nil {
			return "", "", "", rerr
		}
		norm := NormalizeText(content)
		h := sha256.Sum256([]byte(norm))
		return kind, norm, hex.EncodeToString(h[:]), nil
	default:
		return "", "", "", ErrUnsupported
	}
}

func extractPDFText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	r, err := pdf.NewReader(f, info.Size())
	if err != nil {
		// Check if it's an encrypted PDF
		if strings.Contains(err.Error(), "encrypted") || strings.Contains(err.Error(), "password") {
			return "", errors.New("encrypted PDF not supported")
		}
		return "", err
	}

	var sb strings.Builder
	numPages := r.NumPage()
	for i := 1; i <= numPages; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			// Skip pages with errors and continue
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func CopyTo(dest string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Flush()
}
