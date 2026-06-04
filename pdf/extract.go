package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	if !bytes.HasPrefix(header, pdfMagic) {
		return fmt.Errorf("%s: %w", path, ErrNotPDF)
	}
	return nil
}

func ExtractText(ctx context.Context, path string) (string, error) {
	_ = ctx
	if err := validatePDF(path); err != nil {
		return "", err
	}
	return "", nil
}
