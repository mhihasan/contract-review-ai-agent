package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/agent"
	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

type pipelineMemoryStore struct {
	mu        sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.contracts[id]
	if !ok {
		return domain.Contract{}, fmt.Errorf("not found")
	}
	c.Status = m.statuses[id]
	return c, nil
}

func (m *pipelineMemoryStore) UpdateContractStatus(_ context.Context, id string, s domain.ContractStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[id] = s
	return nil
}

func (m *pipelineMemoryStore) UpdateContractText(_ context.Context, _, _ string) error { return nil }

func (m *pipelineMemoryStore) SaveClauses(_ context.Context, contractID string, clauses []domain.Clause) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clauses[contractID] = clauses
	return nil
}

func (m *pipelineMemoryStore) GetClauses(_ context.Context, contractID string) ([]domain.Clause, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := append([]domain.Clause(nil), m.clauses[contractID]...)
	return out, nil
}

func (m *pipelineMemoryStore) SaveAnalysis(_ context.Context, a domain.ClauseAnalysis) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.analyses = append(m.analyses, a)
	return nil
}

func (m *pipelineMemoryStore) GetAnalyses(_ context.Context, _ string) ([]domain.ClauseAnalysis, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := append([]domain.ClauseAnalysis(nil), m.analyses...)
	return out, nil
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

func (m *pipelineMemoryStore) StartRun(_ context.Context, _, _ string) error {
	panic("not implemented")
}

func (m *pipelineMemoryStore) FinishRun(_ context.Context, _, _ string) error {
	panic("not implemented")
}

func (m *pipelineMemoryStore) StartAgentRun(_ context.Context, _, _, _ string) error {
	panic("not implemented")
}

func (m *pipelineMemoryStore) AppendAgentStep(_ context.Context, _ string, _ int, _, _ []byte) error {
	panic("not implemented")
}

func (m *pipelineMemoryStore) FinishAgentRun(_ context.Context, _ string, _ string, _, _ int, _ float64) error {
	panic("not implemented")
}

func (m *pipelineMemoryStore) LoadAgentRun(_ context.Context, _ string) (store.AgentRun, []store.AgentStep, bool, error) {
	return store.AgentRun{}, nil, false, fmt.Errorf("not implemented")
}

func (m *pipelineMemoryStore) GetStoredFinding(_ context.Context, _ string) (domain.ClauseAnalysis, error) {
	return domain.ClauseAnalysis{}, fmt.Errorf("not implemented")
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
		Provider:  "openai",
		Model:     "gpt-4o-mini",
	}
}

func setupContractWithClauses(ms *pipelineMemoryStore, contractID string, count int) {
	ms.contracts[contractID] = domain.Contract{ID: contractID, Status: domain.StatusClausesExtracted}
	ms.statuses[contractID] = domain.StatusClausesExtracted
	clauses := make([]domain.Clause, 0, count)
	for i := 0; i < count; i++ {
		clauses = append(clauses, domain.Clause{
			ID:             fmt.Sprintf("cl-%d", i+1),
			ContractID:     contractID,
			SequenceNumber: i + 1,
			Text:           fmt.Sprintf("Clause %d text", i+1),
		})
	}
	ms.clauses[contractID] = clauses
}

func TestClausesNeedingAnalysis(t *testing.T) {
	contractID := "ct1"
	clauses := []domain.Clause{
		{ID: "c1", ContractID: contractID},
		{ID: "c2", ContractID: contractID},
		{ID: "c3", ContractID: contractID},
	}
	ms := newPipelineStore()
	ms.analyses = []domain.ClauseAnalysis{{ClauseID: "c1", Status: "ok"}}

	got, err := clausesNeedingAnalysis(context.Background(), ms, clauses)
	if err != nil {
		t.Fatalf("clausesNeedingAnalysis: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ID != "c2" || got[1].ID != "c3" {
		t.Fatalf("unexpected clauses: %+v", got)
	}
}

func TestAnalyzeClauses_ConcurrentIsolation(t *testing.T) {
	contractID := "contract-pipeline"
	ms := newPipelineStore()
	setupContractWithClauses(ms, contractID, 5)

	fake := &llm.Fake{
		Script: []llm.CompletionResponse{
			submitFindingResponse("call-1"),
			submitFindingResponse("call-2"),
			submitFindingResponse("call-3"),
			submitFindingResponse("call-4"),
		},
		Err: fmt.Errorf("simulated llm failure"),
	}

	err := AnalyzeClauses(
		context.Background(),
		fake,
		ms,
		contractID,
		10,
		nil,
		agent.NewBudget(1_000_000, 10, 1_000),
		3,
	)
	if err != nil {
		t.Fatalf("AnalyzeClauses: %v", err)
	}

	ms.mu.Lock()
	analyses := append([]domain.ClauseAnalysis(nil), ms.analyses...)
	finalStatus := ms.statuses[contractID]
	ms.mu.Unlock()

	if len(analyses) != 5 {
		t.Fatalf("expected 5 analyses, got %d", len(analyses))
	}

	failedCount := 0
	okCount := 0
	for _, a := range analyses {
		switch a.Status {
		case "failed":
			failedCount++
			if a.RiskLevel != nil {
				t.Fatal("failed row must have nil risk level")
			}
		default:
			okCount++
			if a.RiskLevel == nil {
				t.Fatal("successful row must have risk level")
			}
		}
	}

	if failedCount != 1 {
		t.Fatalf("failedCount = %d, want 1", failedCount)
	}
	if okCount != 4 {
		t.Fatalf("okCount = %d, want 4", okCount)
	}
	if finalStatus != domain.StatusAnalyzed {
		t.Fatalf("contract status = %q, want %q", finalStatus, domain.StatusAnalyzed)
	}
}

func TestAnalyzeClauses_NoClauses_NoError(t *testing.T) {
	contractID := "contract-empty"
	ms := newPipelineStore()
	ms.contracts[contractID] = domain.Contract{ID: contractID, Status: domain.StatusClausesExtracted}
	ms.statuses[contractID] = domain.StatusClausesExtracted

	fake := &llm.Fake{Script: nil}

	err := AnalyzeClauses(
		context.Background(),
		fake,
		ms,
		contractID,
		10,
		nil,
		agent.NewBudget(1_000_000, 10, 1_000),
		2,
	)
	if err != nil {
		t.Fatalf("AnalyzeClauses with no clauses: %v", err)
	}

	if len(ms.analyses) != 0 {
		t.Errorf("expected 0 analyses, got %d", len(ms.analyses))
	}
}

func TestSharedBudgetRaceFree(_ *testing.T) {
	budget := agent.NewBudget(1_000_000, 10.0, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			budget.Record("openai", "gpt-4o-mini", 100, 50)
			_ = budget.Exceeded()
			_ = budget.Snapshot()
		}()
	}
	wg.Wait()
}
