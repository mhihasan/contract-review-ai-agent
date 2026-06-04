package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ledongthuc "github.com/ledongthuc/pdf"
)

var ErrNotPDF = errors.New("not a PDF file")

var pdfMagic = []byte("%PDF-")

func validatePDF(path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return fmt.Errorf("%s: %w", path, ErrNotPDF)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, len(pdfMagic))
	if _, err := f.Read(header); err != nil {
		return fmt.Errorf("read header %s: %w", path, err)
	}
	if !bytes.Equal(header, pdfMagic) {
		return fmt.Errorf("%s: %w", path, ErrNotPDF)
	}
	return nil
}

func ExtractText(ctx context.Context, path string) (string, error) {
	if err := validatePDF(path); err != nil {
		return "", err
	}

	text, err := extractWithLibrary(path)
	if err == nil && strings.TrimSpace(text) != "" {
		return text, nil
	}

	if fallback, ok := extractWithPdftotext(ctx, path); ok && strings.TrimSpace(fallback) != "" {
		return fallback, nil
	}

	if err != nil {
		return "", fmt.Errorf("extract %s: %w", path, err)
	}
	return text, nil
}

func extractWithLibrary(path string) (string, error) {
	f, r, err := ledongthuc.Open(path)
	if err != nil {
		return "", fmt.Errorf("pdf open: %w", err)
	}
	defer func() { _ = f.Close() }()

	reader, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("pdf plain text: %w", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return "", fmt.Errorf("pdf read: %w", err)
	}
	return buf.String(), nil
}

func extractWithPdftotext(ctx context.Context, path string) (string, bool) {
	bin, err := exec.LookPath("pdftotext")
	if err != nil {
		return "", false
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, "-layout", path, "-")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}
	return out.String(), true
}
