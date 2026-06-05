package pipeline_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/pipeline"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

func TestRunSummarize_WrongStatus(t *testing.T) {
	s := store.NewFakeStore()
	contract, _ := s.CreateContract(context.Background(), "test.pdf", "raw text")
	_ = s.UpdateContractStatus(context.Background(), contract.ID, domain.StatusAnalyzed)

	err := pipeline.RunSummarize(context.Background(), s, llm.NewFake(""), contract.ID, 200, "gpt-4o-mini")
	if err == nil {
		t.Fatal("expected error for wrong status, got nil")
	}
	if !strings.Contains(err.Error(), "expected review_complete") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunSummarize_AlreadyDone_NoOp(t *testing.T) {
	s := store.NewFakeStore()
	contract, _ := s.CreateContract(context.Background(), "test.pdf", "raw text")
	_ = s.UpdateContractStatus(context.Background(), contract.ID, domain.StatusDone)
	_ = s.SaveSummary(context.Background(), contract.ID, "existing summary content")

	callCount := 0
	fakeLLM := llm.NewFakeWithHook("", func() { callCount++ })

	err := pipeline.RunSummarize(context.Background(), s, fakeLLM, contract.ID, 200, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("unexpected error on done contract: %v", err)
	}
	if callCount != 0 {
		t.Errorf("LLM called %d times on a done contract; expected 0", callCount)
	}
}

func TestRunSummarize_ComputesRiskCounts(t *testing.T) {
	s := store.NewFakeStore()
	contract, _ := s.CreateContract(context.Background(), "test.pdf", "raw text")
	_ = s.UpdateContractStatus(context.Background(), contract.ID, domain.StatusReviewComplete)

	highRisk := domain.RiskHigh
	medRisk := domain.RiskMedium
	clauses := []domain.Clause{
		{ID: "c1", ContractID: contract.ID, SequenceNumber: 1, Text: "Clause one."},
		{ID: "c2", ContractID: contract.ID, SequenceNumber: 2, Text: "Clause two."},
		{ID: "c3", ContractID: contract.ID, SequenceNumber: 3, Text: "Clause three."},
	}
	_ = s.SaveClauses(context.Background(), contract.ID, clauses)
	_ = s.SaveAnalysis(context.Background(), domain.ClauseAnalysis{ID: "a1", ClauseID: "c1", RiskLevel: &highRisk, Status: "submitted"})
	_ = s.SaveAnalysis(context.Background(), domain.ClauseAnalysis{ID: "a2", ClauseID: "c2", RiskLevel: &medRisk, Status: "submitted"})
	_ = s.SaveAnalysis(context.Background(), domain.ClauseAnalysis{ID: "a3", ClauseID: "c3", RiskLevel: &highRisk, Status: "submitted"})
	_ = s.SaveReview(context.Background(), domain.Review{ID: "r1", ClauseID: "c1", Decision: "approved"})
	_ = s.SaveReview(context.Background(), domain.Review{ID: "r2", ClauseID: "c2", Decision: "approved"})
	_ = s.SaveReview(context.Background(), domain.Review{ID: "r3", ClauseID: "c3", Decision: "rejected", Annotation: "Too broad."})

	captured := ""
	fakeLLM := llm.NewFakeCapture(&captured)

	err := pipeline.RunSummarize(context.Background(), s, fakeLLM, contract.ID, 200, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(captured, "High: 2") {
		t.Errorf("prompt missing High: 2; got: %s", captured)
	}
	if !strings.Contains(captured, "Medium: 1") {
		t.Errorf("prompt missing Medium: 1; got: %s", captured)
	}
	if !strings.Contains(captured, "Rejected: 1") {
		t.Errorf("prompt missing Rejected: 1; got: %s", captured)
	}
}

func TestRunSummarize_StatusTransitions(t *testing.T) {
	s := store.NewFakeStore()
	contract, _ := s.CreateContract(context.Background(), "test.pdf", "raw text")
	_ = s.UpdateContractStatus(context.Background(), contract.ID, domain.StatusReviewComplete)
	clause := domain.Clause{ID: "c1", ContractID: contract.ID, SequenceNumber: 1, Text: "Text."}
	_ = s.SaveClauses(context.Background(), contract.ID, []domain.Clause{clause})
	low := domain.RiskLow
	_ = s.SaveAnalysis(context.Background(), domain.ClauseAnalysis{ID: "a1", ClauseID: "c1", RiskLevel: &low, Status: "submitted"})
	_ = s.SaveReview(context.Background(), domain.Review{ID: "r1", ClauseID: "c1", Decision: "approved"})

	err := pipeline.RunSummarize(context.Background(), s, llm.NewFake("# Report\n\nAll good."), contract.ID, 200, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := s.GetContract(context.Background(), contract.ID)
	if updated.Status != domain.StatusDone {
		t.Errorf("expected status done, got %s", updated.Status)
	}

	summary, _ := s.GetSummary(context.Background(), contract.ID)
	if !strings.Contains(summary.Content, "All good.") {
		t.Errorf("summary content not saved: %q", summary.Content)
	}
}
