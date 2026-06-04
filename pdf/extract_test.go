package pdf

import (
	"context"
	"errors"
	"testing"
)

func TestExtractText_RejectsNonPDF(t *testing.T) {
	_, err := ExtractText(context.Background(), "testdata/sample.txt")
	if !errors.Is(err, ErrNotPDF) {
		t.Fatalf("want ErrNotPDF, got %v", err)
	}
}

func TestExtractText_MissingFile(t *testing.T) {
	_, err := ExtractText(context.Background(), "testdata/does-not-exist.pdf")
	if err == nil {
		t.Fatal("want error for missing file, got nil")
	}
	if errors.Is(err, ErrNotPDF) {
		t.Fatalf("missing file should not be ErrNotPDF, got %v", err)
	}
}
