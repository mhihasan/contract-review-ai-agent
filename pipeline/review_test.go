package pipeline_test

import (
	"context"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/pipeline"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

func TestParseDecision(t *testing.T) {
	tests := []struct {
		input   string
		want    pipeline.Decision
		wantErr bool
	}{
		{"a", pipeline.DecisionApproved, false},
		{"r", pipeline.DecisionRejected, false},
		{"n", pipeline.DecisionNote, false},
		{"A", pipeline.DecisionApproved, false},
		{"R", pipeline.DecisionRejected, false},
		{"N", pipeline.DecisionNote, false},
		{"", pipeline.Decision(""), true},
		{"x", pipeline.Decision(""), true},
		{"approve", pipeline.Decision(""), true},
	}

	for _, tt := range tests {
		got, err := pipeline.ParseDecision(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDecision(%q): wantErr=%v, got err=%v", tt.input, tt.wantErr, err)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseDecision(%q): want %q, got %q", tt.input, tt.want, got)
		}
	}
}

// resumeStore is a minimal fake implementing only what RunResume needs.
type resumeStore struct {
	store.Store
	contract  domain.Contract
	clauses   []domain.Clause
	reviews   []domain.Review
	newStatus domain.ContractStatus
}

func (f *resumeStore) GetContract(_ context.Context, _ string) (domain.Contract, error) {
	return f.contract, nil
}
func (f *resumeStore) GetClauses(_ context.Context, _ string) ([]domain.Clause, error) {
	return f.clauses, nil
}
func (f *resumeStore) GetReviews(_ context.Context, _ string) ([]domain.Review, error) {
	return f.reviews, nil
}
func (f *resumeStore) UpdateContractStatus(_ context.Context, _ string, status domain.ContractStatus) error {
	f.newStatus = status
	return nil
}

func TestRunResume_PartialReviews(t *testing.T) {
	fs := &resumeStore{
		contract: domain.Contract{ID: "c1", Status: domain.StatusReviewPending},
		clauses:  []domain.Clause{{ID: "cl1"}, {ID: "cl2"}},
		reviews:  []domain.Review{{ClauseID: "cl1", Decision: "approved"}},
	}
	err := pipeline.RunResume(context.Background(), fs, "c1", func(_ context.Context, _ store.Store, _ string) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for partial reviews, got nil")
	}
}

func TestRunResume_Complete(t *testing.T) {
	var summarizeCalled bool
	fs := &resumeStore{
		contract: domain.Contract{ID: "c1", Status: domain.StatusReviewPending},
		clauses:  []domain.Clause{{ID: "cl1"}, {ID: "cl2"}},
		reviews: []domain.Review{
			{ClauseID: "cl1", Decision: "approved"},
			{ClauseID: "cl2", Decision: "rejected"},
		},
	}
	err := pipeline.RunResume(context.Background(), fs, "c1", func(_ context.Context, _ store.Store, _ string) error {
		summarizeCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fs.newStatus != domain.StatusReviewComplete {
		t.Errorf("expected status review_complete, got %s", fs.newStatus)
	}
	if !summarizeCalled {
		t.Error("expected summarize to be called")
	}
}
