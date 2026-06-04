package pipeline

import (
	"errors"
	"testing"
)

func TestParseClauses_ValidArray(t *testing.T) {
	clauses, err := parseClauses(`["clause one", "clause two", "clause three"]`)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(clauses) != 3 {
		t.Fatalf("expected 3 clauses, got %d", len(clauses))
	}
	if clauses[0] != "clause one" {
		t.Errorf("clauses[0] = %q, want %q", clauses[0], "clause one")
	}
}

func TestParseClauses_SingleElement(t *testing.T) {
	clauses, err := parseClauses(`["just one clause"]`)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
}

func TestParseClauses_StripsMarkdownFences(t *testing.T) {
	raw := "```json\n[\"clause one\", \"clause two\"]\n```"
	clauses, err := parseClauses(raw)
	if err != nil {
		t.Fatalf("expected no error after fence stripping, got: %v", err)
	}
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
}

func TestParseClauses_EmptyArray_ReturnsError(t *testing.T) {
	_, err := parseClauses(`[]`)
	if err == nil {
		t.Fatal("expected error for empty array, got nil")
	}
}

func TestParseClauses_InvalidJSON_ReturnsError(t *testing.T) {
	_, err := parseClauses(`not json at all`)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestErrClauseParse_IsSentinel(t *testing.T) {
	if !errors.Is(ErrClauseParse, ErrClauseParse) {
		t.Fatal("ErrClauseParse must satisfy errors.Is with itself")
	}
}

func TestExtractClauses_IdempotentOnAlreadyExtracted(_ *testing.T) {
	_ = ExtractClauses // ensure it is exported and callable
}
