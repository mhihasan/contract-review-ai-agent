package tool_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/tool"
)

func seedStore() (*MemoryStore, string) {
	ms := newMemoryStore()
	contractID := "contract-abc"
	ms.contracts[contractID] = domain.Contract{ID: contractID}
	ms.clauses[contractID] = []domain.Clause{
		{ID: "cl-1", ContractID: contractID, SequenceNumber: 1, Text: `"Effective Date" means the date first written above.`},
		{ID: "cl-2", ContractID: contractID, SequenceNumber: 2, Text: "Section 2: Payment. All invoices are due within 30 days."},
		{ID: "cl-3", ContractID: contractID, SequenceNumber: 3, Text: "Section 7.2: Liability. Neither party shall be liable for indirect damages."},
	}
	ms.library = []domain.LibraryClause{
		{ID: "lib-1", ClauseType: "liability", StandardText: "Liability shall not exceed fees paid in prior 12 months.", Notes: "Standard cap."},
		{ID: "lib-2", ClauseType: "indemnity", StandardText: "Each party shall indemnify the other for third-party claims.", Notes: "Mutual indemnity."},
	}
	return ms, contractID
}

func TestGetDefinition_Name(t *testing.T) {
	ms, id := seedStore()
	td := tool.NewGetDefinition(ms, id)
	if td.Name() != "get_definition" {
		t.Errorf("Name() = %q, want %q", td.Name(), "get_definition")
	}
}

func TestGetDefinition_Schema_ValidJSON(t *testing.T) {
	ms, id := seedStore()
	td := tool.NewGetDefinition(ms, id)
	if !json.Valid(td.Schema()) {
		t.Errorf("Schema() is not valid JSON: %s", td.Schema())
	}
}

func TestGetDefinition_Found(t *testing.T) {
	ms, id := seedStore()
	td := tool.NewGetDefinition(ms, id)
	result, err := td.Execute(context.Background(), json.RawMessage(`{"term":"Effective Date"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "date first written above") {
		t.Errorf("expected definition text in result, got: %s", result)
	}
}

func TestGetDefinition_NotFound(t *testing.T) {
	ms, id := seedStore()
	td := tool.NewGetDefinition(ms, id)
	result, err := td.Execute(context.Background(), json.RawMessage(`{"term":"Unicorn Term"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not found") {
		t.Errorf("expected 'not found' in result, got: %s", result)
	}
}

func TestGetDefinition_BadArgs(t *testing.T) {
	ms, id := seedStore()
	td := tool.NewGetDefinition(ms, id)
	result, err := td.Execute(context.Background(), json.RawMessage(`not-json`))
	if err != nil {
		t.Fatalf("Execute must not return Go error for bad args, got: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty error string for bad args")
	}
}

func TestGetContractSection_Found(t *testing.T) {
	ms, id := seedStore()
	ts := tool.NewGetContractSection(ms, id)
	result, err := ts.Execute(context.Background(), json.RawMessage(`{"reference":"Section 7.2"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "indirect damages") {
		t.Errorf("expected section text in result, got: %s", result)
	}
}

func TestGetContractSection_NotFound(t *testing.T) {
	ms, id := seedStore()
	ts := tool.NewGetContractSection(ms, id)
	result, err := ts.Execute(context.Background(), json.RawMessage(`{"reference":"Section 99"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not found") {
		t.Errorf("expected 'not found' in result, got: %s", result)
	}
}

func TestSearchClauseLibrary_ReturnsMatches(t *testing.T) {
	ms, id := seedStore()
	ts := tool.NewSearchClauseLibrary(ms, id)
	result, err := ts.Execute(context.Background(), json.RawMessage(`{"query":"liability"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "liability") {
		t.Errorf("expected 'liability' in result, got: %s", result)
	}
}

func TestSearchClauseLibrary_NoMatches(t *testing.T) {
	ms, id := seedStore()
	ts := tool.NewSearchClauseLibrary(ms, id)
	result, err := ts.Execute(context.Background(), json.RawMessage(`{"query":"zxqwerty"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(result, "error") {
		t.Errorf("unexpected error string for no-match search: %s", result)
	}
}

func TestLookupStandardClause_Found(t *testing.T) {
	ms, id := seedStore()
	tl := tool.NewLookupStandardClause(ms, id)
	result, err := tl.Execute(context.Background(), json.RawMessage(`{"clause_type":"liability"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "liability") {
		t.Errorf("expected liability clause text in result, got: %s", result)
	}
}

func TestLookupStandardClause_NotFound(t *testing.T) {
	ms, id := seedStore()
	tl := tool.NewLookupStandardClause(ms, id)
	result, err := tl.Execute(context.Background(), json.RawMessage(`{"clause_type":"nonexistent"}`))
	if err != nil {
		t.Fatalf("Execute must not return Go error for missing clause type, got: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not found") {
		t.Errorf("expected 'not found' in result, got: %s", result)
	}
}
