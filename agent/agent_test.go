package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/agent"
	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/tool"
)

type memoryStore struct {
	contracts map[string]domain.Contract
	clauses   map[string][]domain.Clause
	library   []domain.LibraryClause
	analyses  []domain.ClauseAnalysis
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		contracts: make(map[string]domain.Contract),
		clauses:   make(map[string][]domain.Clause),
	}
}

func (m *memoryStore) CreateContract(_ context.Context, _, _ string) (domain.Contract, error) {
	panic("not implemented")
}
func (m *memoryStore) GetContract(_ context.Context, id string) (domain.Contract, error) {
	c, ok := m.contracts[id]
	if !ok {
		return domain.Contract{}, fmt.Errorf("contract %q not found", id)
	}
	return c, nil
}
func (m *memoryStore) UpdateContractStatus(_ context.Context, _ string, _ domain.ContractStatus) error {
	return nil
}
func (m *memoryStore) UpdateContractText(_ context.Context, _, _ string) error { return nil }
func (m *memoryStore) SaveClauses(_ context.Context, contractID string, clauses []domain.Clause) error {
	m.clauses[contractID] = clauses
	return nil
}
func (m *memoryStore) GetClauses(_ context.Context, contractID string) ([]domain.Clause, error) {
	return m.clauses[contractID], nil
}
func (m *memoryStore) SaveAnalysis(_ context.Context, a domain.ClauseAnalysis) error {
	m.analyses = append(m.analyses, a)
	return nil
}
func (m *memoryStore) GetAnalyses(_ context.Context, _ string) ([]domain.ClauseAnalysis, error) {
	return m.analyses, nil
}
func (m *memoryStore) SaveReview(_ context.Context, _ domain.Review) error { return nil }
func (m *memoryStore) GetReviews(_ context.Context, _ string) ([]domain.Review, error) {
	return nil, nil
}
func (m *memoryStore) SaveSummary(_ context.Context, _, _ string) error { return nil }
func (m *memoryStore) GetSummary(_ context.Context, _ string) (domain.Summary, error) {
	return domain.Summary{}, nil
}
func (m *memoryStore) SearchClauseLibrary(_ context.Context, _ string) ([]domain.LibraryClause, error) {
	return m.library, nil
}
func (m *memoryStore) GetStandardClause(_ context.Context, clauseType string) (domain.LibraryClause, error) {
	for _, c := range m.library {
		if c.ClauseType == clauseType {
			return c, nil
		}
	}
	return domain.LibraryClause{}, fmt.Errorf("not found")
}

func seedStore(contractID string) *memoryStore {
	ms := newMemoryStore()
	ms.contracts[contractID] = domain.Contract{ID: contractID}
	ms.clauses[contractID] = []domain.Clause{
		{ID: "cl-1", ContractID: contractID, SequenceNumber: 1, Text: `"Effective Date" means the date first written above.`},
		{ID: "cl-2", ContractID: contractID, SequenceNumber: 2, Text: "Section 7.2: Liability. Neither party shall be liable for indirect damages."},
	}
	ms.library = []domain.LibraryClause{
		{ID: "lib-1", ClauseType: "liability", StandardText: "Liability shall not exceed fees paid in prior 12 months.", Notes: "Standard cap."},
	}
	return ms
}

func buildRegistry(ms *memoryStore, contractID string) *tool.Registry {
	return tool.NewRegistry(
		tool.NewGetDefinition(ms, contractID),
		tool.NewGetContractSection(ms, contractID),
		tool.NewSearchClauseLibrary(ms, contractID),
		tool.NewLookupStandardClause(ms, contractID),
	)
}

func submitFindingCall(callID string) llm.ToolCall {
	args, _ := json.Marshal(map[string]string{
		"risk_level":         "high",
		"explanation":        "The liability clause lacks a cap, creating unlimited exposure.",
		"ambiguous_language": "Neither party shall be liable — scope is ambiguous.",
		"recommendations":    "Add a mutual liability cap tied to fees paid in the prior 12 months.",
	})
	return llm.ToolCall{ID: callID, Name: "submit_finding", Args: json.RawMessage(args)}
}

func TestAgent_HappyPath_SubmittedAfterToolCalls(t *testing.T) {
	contractID := "contract-happy"
	ms := seedStore(contractID)
	reg := buildRegistry(ms, contractID)

	fake := &llm.Fake{
		Script: []llm.CompletionResponse{
			{
				ToolCalls: []llm.ToolCall{
					{ID: "call-1", Name: "get_contract_section", Args: json.RawMessage(`{"reference":"Section 7.2"}`)},
				},
			},
			{
				ToolCalls: []llm.ToolCall{
					{ID: "call-2", Name: "lookup_standard_clause", Args: json.RawMessage(`{"clause_type":"liability"}`)},
				},
			},
			{
				ToolCalls: []llm.ToolCall{submitFindingCall("call-3")},
			},
		},
	}

	a := agent.New(fake, reg, 10, nil)
	result, err := a.Run(context.Background(), agent.AnalyzeClauseTask{
		ContractID: contractID,
		ClauseID:   "cl-2",
		ClauseText: "Section 7.2: Liability. Neither party shall be liable for indirect damages.",
	})

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Stop != "submitted" {
		t.Errorf("Stop = %q, want %q", result.Stop, "submitted")
	}
	if result.Steps != 3 {
		t.Errorf("Steps = %d, want 3", result.Steps)
	}
	if result.Finding.RiskLevel == nil {
		t.Fatal("Finding.RiskLevel must not be nil after successful submission")
	}
	if string(*result.Finding.RiskLevel) != "high" {
		t.Errorf("RiskLevel = %q, want %q", *result.Finding.RiskLevel, "high")
	}
	if result.Finding.Explanation == "" {
		t.Error("Finding.Explanation must not be empty")
	}
}

func TestAgent_MaxSteps_LoopCannotRunForever(t *testing.T) {
	contractID := "contract-runaway"
	ms := seedStore(contractID)
	reg := buildRegistry(ms, contractID)

	maxSteps := 4
	script := make([]llm.CompletionResponse, maxSteps)
	for i := range script {
		script[i] = llm.CompletionResponse{
			ToolCalls: []llm.ToolCall{
				{ID: fmt.Sprintf("call-%d", i), Name: "get_contract_section", Args: json.RawMessage(`{"reference":"Section 7.2"}`)},
			},
		}
	}

	fake := &llm.Fake{Script: script}
	a := agent.New(fake, reg, maxSteps, nil)
	result, err := a.Run(context.Background(), agent.AnalyzeClauseTask{
		ContractID: contractID,
		ClauseID:   "cl-2",
		ClauseText: "Section 7.2: Liability clause.",
	})

	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result.Stop != "max_steps" {
		t.Errorf("Stop = %q, want %q", result.Stop, "max_steps")
	}
	if result.Steps != maxSteps {
		t.Errorf("Steps = %d, want %d", result.Steps, maxSteps)
	}
}

func TestAgent_InvalidFinding_AgentGetsAnotherTurn(t *testing.T) {
	contractID := "contract-retry"
	ms := seedStore(contractID)
	reg := buildRegistry(ms, contractID)

	badArgs, _ := json.Marshal(map[string]string{
		"risk_level":         "critical",
		"explanation":        "bad level",
		"ambiguous_language": "",
		"recommendations":    "fix it",
	})
	goodArgs, _ := json.Marshal(map[string]string{
		"risk_level":         "medium",
		"explanation":        "Corrected assessment.",
		"ambiguous_language": "",
		"recommendations":    "Accept as-is.",
	})

	fake := &llm.Fake{
		Script: []llm.CompletionResponse{
			{ToolCalls: []llm.ToolCall{{ID: "call-bad", Name: "submit_finding", Args: json.RawMessage(badArgs)}}},
			{ToolCalls: []llm.ToolCall{{ID: "call-good", Name: "submit_finding", Args: json.RawMessage(goodArgs)}}},
		},
	}

	a := agent.New(fake, reg, 10, nil)
	result, err := a.Run(context.Background(), agent.AnalyzeClauseTask{
		ContractID: contractID,
		ClauseID:   "cl-2",
		ClauseText: "Liability clause.",
	})

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Stop != "submitted" {
		t.Errorf("Stop = %q, want %q", result.Stop, "submitted")
	}
	if result.Steps != 2 {
		t.Errorf("Steps = %d, want 2", result.Steps)
	}
	if string(*result.Finding.RiskLevel) != "medium" {
		t.Errorf("RiskLevel = %q, want %q", *result.Finding.RiskLevel, "medium")
	}
}

func TestAgent_ToolResultMatchesToolCallID(t *testing.T) {
	contractID := "contract-id-check"
	ms := seedStore(contractID)
	reg := buildRegistry(ms, contractID)

	fake := &llm.Fake{
		Script: []llm.CompletionResponse{
			{ToolCalls: []llm.ToolCall{
				{ID: "unique-call-id-99", Name: "get_contract_section", Args: json.RawMessage(`{"reference":"Section 7.2"}`)},
			}},
			{ToolCalls: []llm.ToolCall{submitFindingCall("final-call")}},
		},
	}

	a := agent.New(fake, reg, 10, nil)
	_, err := a.Run(context.Background(), agent.AnalyzeClauseTask{
		ContractID: contractID,
		ClauseID:   "cl-2",
		ClauseText: "Liability clause.",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) < 2 {
		t.Fatalf("expected at least 2 LLM calls, got %d", len(fake.Calls))
	}
	secondCallMsgs := fake.Calls[1].Messages
	var foundToolResult bool
	for _, msg := range secondCallMsgs {
		if msg.Role == llm.RoleTool && msg.ToolCallID == "unique-call-id-99" {
			foundToolResult = true
		}
	}
	if !foundToolResult {
		t.Error("second LLM call did not contain a RoleTool message with ToolCallID='unique-call-id-99'")
	}
}

func TestAgent_ProseResponse_NudgesModel(t *testing.T) {
	contractID := "contract-prose"
	ms := seedStore(contractID)
	reg := buildRegistry(ms, contractID)

	fake := &llm.Fake{
		Script: []llm.CompletionResponse{
			{Content: "I think this clause looks fine.", ToolCalls: nil},
			{ToolCalls: []llm.ToolCall{submitFindingCall("call-final")}},
		},
	}

	a := agent.New(fake, reg, 10, nil)
	result, err := a.Run(context.Background(), agent.AnalyzeClauseTask{
		ContractID: contractID,
		ClauseID:   "cl-2",
		ClauseText: "Liability clause.",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Stop != "submitted" {
		t.Errorf("Stop = %q, want %q", result.Stop, "submitted")
	}

	if len(fake.Calls) < 2 {
		t.Fatalf("expected at least 2 LLM calls, got %d", len(fake.Calls))
	}
	secondCallMsgs := fake.Calls[1].Messages
	var foundNudge bool
	for _, msg := range secondCallMsgs {
		if msg.Role == llm.RoleUser && msg.Content != "" {
			foundNudge = true
		}
	}
	if !foundNudge {
		t.Error("expected a user nudge message in the second LLM call after a prose response")
	}
}

func TestAgent_Cancellation_StopsPromptly(t *testing.T) {
	contractID := "contract-cancel"
	ms := seedStore(contractID)
	reg := buildRegistry(ms, contractID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fake := &llm.Fake{Script: []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCall{
			{ID: "c1", Name: "get_contract_section", Args: json.RawMessage(`{"reference":"Section 7.2"}`)},
		}},
	}}

	a := agent.New(fake, reg, 10, nil)
	result, err := a.Run(ctx, agent.AnalyzeClauseTask{
		ContractID: contractID,
		ClauseID:   "cl-2",
		ClauseText: "Liability clause.",
	})

	if result.Stop != "cancelled" {
		t.Errorf("Stop = %q, want %q", result.Stop, "cancelled")
	}
	if err == nil {
		t.Error("expected non-nil error when context is cancelled")
	}
}
