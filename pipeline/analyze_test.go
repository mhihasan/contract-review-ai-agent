package pipeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/pipeline"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

type pipelineMemoryStore struct {
	contracts map[string]domain.Contract
	clauses   map[string][]domain.Clause
	library   []domain.LibraryClause
	analyses  []domain.ClauseAnalysis
	statuses  map[string]domain.ContractStatus
}

func newPipelineStore() *pipelineMemoryStore {
	return &pipelineMemoryStore{
		contracts: make(map[string]domain.Contract),
		clauses:   make(map[string][]domain.Clause),
		statuses:  make(map[string]domain.ContractStatus),
	}
}

func (m *pipelineMemoryStore) CreateContract(_ context.Context, _, _ string) (domain.Contract, error) {
	panic("not implemented")
}
func (m *pipelineMemoryStore) GetContract(_ context.Context, id string) (domain.Contract, error) {
	c, ok := m.contracts[id]
	if !ok {
		return domain.Contract{}, fmt.Errorf("not found")
	}
	c.Status = m.statuses[id]
	return c, nil
}
func (m *pipelineMemoryStore) UpdateContractStatus(_ context.Context, id string, s domain.ContractStatus) error {
	m.statuses[id] = s
	return nil
}
func (m *pipelineMemoryStore) UpdateContractText(_ context.Context, _, _ string) error { return nil }
func (m *pipelineMemoryStore) SaveClauses(_ context.Context, contractID string, clauses []domain.Clause) error {
	m.clauses[contractID] = clauses
	return nil
}
func (m *pipelineMemoryStore) GetClauses(_ context.Context, contractID string) ([]domain.Clause, error) {
	return m.clauses[contractID], nil
}
func (m *pipelineMemoryStore) SaveAnalysis(_ context.Context, a domain.ClauseAnalysis) error {
	m.analyses = append(m.analyses, a)
	return nil
}
func (m *pipelineMemoryStore) GetAnalyses(_ context.Context, _ string) ([]domain.ClauseAnalysis, error) {
	return m.analyses, nil
}
func (m *pipelineMemoryStore) SaveReview(_ context.Context, _ domain.Review) error { return nil }
func (m *pipelineMemoryStore) GetReviews(_ context.Context, _ string) ([]domain.Review, error) {
	return nil, nil
}
func (m *pipelineMemoryStore) SaveSummary(_ context.Context, _, _ string) error { return nil }
func (m *pipelineMemoryStore) GetSummary(_ context.Context, _ string) (domain.Summary, error) {
	return domain.Summary{}, nil
}
func (m *pipelineMemoryStore) SearchClauseLibrary(_ context.Context, _ string) ([]domain.LibraryClause, error) {
	return m.library, nil
}
func (m *pipelineMemoryStore) GetStandardClause(_ context.Context, clauseType string) (domain.LibraryClause, error) {
	for _, c := range m.library {
		if c.ClauseType == clauseType {
			return c, nil
		}
	}
	return domain.LibraryClause{}, fmt.Errorf("not found")
}

var _ store.Store = (*pipelineMemoryStore)(nil)

func submitFindingResponse(callID string) llm.CompletionResponse {
	args, _ := json.Marshal(map[string]string{
		"risk_level":         "low",
		"explanation":        "Standard clause with no unusual risk.",
		"ambiguous_language": "",
		"recommendations":    "Accept as-is.",
	})
	return llm.CompletionResponse{
		ToolCalls: []llm.ToolCall{{ID: callID, Name: "submit_finding", Args: json.RawMessage(args)}},
	}
}

func TestAnalyzeClauses_AnalysesEveryClause(t *testing.T) {
	contractID := "contract-pipeline"
	ms := newPipelineStore()
	ms.contracts[contractID] = domain.Contract{ID: contractID, Status: domain.StatusClausesExtracted}
	ms.statuses[contractID] = domain.StatusClausesExtracted
	ms.clauses[contractID] = []domain.Clause{
		{ID: "cl-1", ContractID: contractID, SequenceNumber: 1, Text: "Payment is due in 30 days."},
		{ID: "cl-2", ContractID: contractID, SequenceNumber: 2, Text: "Either party may terminate on 30 days notice."},
	}

	fake := &llm.Fake{
		Script: []llm.CompletionResponse{
			submitFindingResponse("call-1"),
			submitFindingResponse("call-2"),
		},
	}

	if err := pipeline.AnalyzeClauses(context.Background(), fake, ms, contractID, 10); err != nil {
		t.Fatalf("AnalyzeClauses: %v", err)
	}

	if len(ms.analyses) != 2 {
		t.Errorf("expected 2 saved analyses, got %d", len(ms.analyses))
	}
	for _, a := range ms.analyses {
		if a.RiskLevel == nil {
			t.Error("analysis has nil RiskLevel")
		}
		if a.Status != "analyzed" {
			t.Errorf("analysis status = %q, want %q", a.Status, "analyzed")
		}
	}

	finalStatus := ms.statuses[contractID]
	if finalStatus != domain.StatusAnalyzed {
		t.Errorf("contract status = %q, want %q", finalStatus, domain.StatusAnalyzed)
	}
}

func TestAnalyzeClauses_NoClauses_NoError(t *testing.T) {
	contractID := "contract-empty"
	ms := newPipelineStore()
	ms.contracts[contractID] = domain.Contract{ID: contractID, Status: domain.StatusClausesExtracted}
	ms.statuses[contractID] = domain.StatusClausesExtracted

	fake := &llm.Fake{Script: nil}

	if err := pipeline.AnalyzeClauses(context.Background(), fake, ms, contractID, 10); err != nil {
		t.Fatalf("AnalyzeClauses with no clauses: %v", err)
	}
	if len(ms.analyses) != 0 {
		t.Errorf("expected 0 analyses, got %d", len(ms.analyses))
	}
}
