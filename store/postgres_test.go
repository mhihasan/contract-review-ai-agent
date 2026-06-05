package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return store.NewPostgresStore(pool)
}

func TestCreateAndGetContract(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contract, err := s.CreateContract(ctx, "test.pdf", "raw contract text")
	if err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	if contract.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if contract.Status != domain.StatusUploaded {
		t.Errorf("expected status %q, got %q", domain.StatusUploaded, contract.Status)
	}

	got, err := s.GetContract(ctx, contract.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if got.Filename != "test.pdf" {
		t.Errorf("expected filename 'test.pdf', got %q", got.Filename)
	}
}

func TestUpdateContractStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contract, err := s.CreateContract(ctx, "status.pdf", "text")
	if err != nil {
		t.Fatalf("CreateContract: %v", err)
	}

	if err := s.UpdateContractStatus(ctx, contract.ID, domain.StatusExtracting); err != nil {
		t.Fatalf("UpdateContractStatus: %v", err)
	}

	got, err := s.GetContract(ctx, contract.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if got.Status != domain.StatusExtracting {
		t.Errorf("expected status %q, got %q", domain.StatusExtracting, got.Status)
	}
}

func TestSaveAndGetClauses(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contract, _ := s.CreateContract(ctx, "clauses.pdf", "text")

	clauses := []domain.Clause{
		{ID: uuid.New().String(), ContractID: contract.ID, SequenceNumber: 1, Text: "clause one"},
		{ID: uuid.New().String(), ContractID: contract.ID, SequenceNumber: 2, Text: "clause two"},
	}
	if err := s.SaveClauses(ctx, contract.ID, clauses); err != nil {
		t.Fatalf("SaveClauses: %v", err)
	}

	got, err := s.GetClauses(ctx, contract.ID)
	if err != nil {
		t.Fatalf("GetClauses: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(got))
	}
	if got[0].Text != "clause one" {
		t.Errorf("expected 'clause one', got %q", got[0].Text)
	}
}

func TestSaveAndGetAnalysis(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contract, _ := s.CreateContract(ctx, "analysis.pdf", "text")
	clauses := []domain.Clause{
		{ID: uuid.New().String(), ContractID: contract.ID, SequenceNumber: 1, Text: "risky clause"},
	}
	_ = s.SaveClauses(ctx, contract.ID, clauses)

	riskHigh := domain.RiskHigh
	analysis := domain.ClauseAnalysis{
		ID:                uuid.New().String(),
		ClauseID:          clauses[0].ID,
		RiskLevel:         &riskHigh,
		Explanation:       "contains indemnity",
		AmbiguousLanguage: "reasonable efforts",
		Recommendations:   "define 'reasonable'",
		Status:            "ok",
	}
	if err := s.SaveAnalysis(ctx, analysis); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}

	got, err := s.GetAnalyses(ctx, contract.ID)
	if err != nil {
		t.Fatalf("GetAnalyses: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 analysis, got %d", len(got))
	}
	if got[0].RiskLevel == nil || *got[0].RiskLevel != domain.RiskHigh {
		t.Errorf("expected risk level 'high'")
	}
}

func TestSaveAndGetReview(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contract, _ := s.CreateContract(ctx, "review.pdf", "text")
	clauses := []domain.Clause{
		{ID: uuid.New().String(), ContractID: contract.ID, SequenceNumber: 1, Text: "clause"},
	}
	_ = s.SaveClauses(ctx, contract.ID, clauses)

	review := domain.Review{
		ID:         uuid.New().String(),
		ClauseID:   clauses[0].ID,
		Decision:   "approved",
		Annotation: "looks fine",
	}
	if err := s.SaveReview(ctx, review); err != nil {
		t.Fatalf("SaveReview: %v", err)
	}

	got, err := s.GetReviews(ctx, contract.ID)
	if err != nil {
		t.Fatalf("GetReviews: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 review, got %d", len(got))
	}
	if got[0].Decision != "approved" {
		t.Errorf("expected 'approved', got %q", got[0].Decision)
	}
}

func TestSaveAndGetSummary(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contract, _ := s.CreateContract(ctx, "summary.pdf", "text")

	if err := s.SaveSummary(ctx, contract.ID, "this contract is risky"); err != nil {
		t.Fatalf("SaveSummary: %v", err)
	}

	got, err := s.GetSummary(ctx, contract.ID)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if got.Content != "this contract is risky" {
		t.Errorf("expected summary content, got %q", got.Content)
	}
}

func TestSaveSummary_Upsert(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contract, _ := s.CreateContract(ctx, "upsert.pdf", "text")

	_ = s.SaveSummary(ctx, contract.ID, "first summary")
	if err := s.SaveSummary(ctx, contract.ID, "updated summary"); err != nil {
		t.Fatalf("second SaveSummary: %v", err)
	}

	got, _ := s.GetSummary(ctx, contract.ID)
	if got.Content != "updated summary" {
		t.Errorf("expected upserted content, got %q", got.Content)
	}
}

func TestPostgresStore_SearchClauseLibrary(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	s := store.NewPostgresStore(pool)
	results, err := s.SearchClauseLibrary(ctx, "liability")
	if err != nil {
		t.Fatalf("SearchClauseLibrary: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'liability'")
	}
	for _, r := range results {
		if r.ID == "" || r.ClauseType == "" || r.StandardText == "" {
			t.Errorf("incomplete row: %+v", r)
		}
	}
}

func TestPostgresStore_GetStandardClause(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	s := store.NewPostgresStore(pool)
	c, err := s.GetStandardClause(ctx, "liability")
	if err != nil {
		t.Fatalf("GetStandardClause: %v", err)
	}
	if c.ClauseType != "liability" {
		t.Errorf("ClauseType = %q, want %q", c.ClauseType, "liability")
	}
	if c.StandardText == "" {
		t.Error("StandardText must not be empty")
	}
}

func TestAgentRunPersistence(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := store.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	s := store.NewPostgresStore(pool)

	contract, err := s.CreateContract(ctx, "test.pdf", "test text")
	if err != nil {
		t.Fatalf("create contract: %v", err)
	}
	clauseID := "clause-persist-" + contract.ID
	err = s.SaveClauses(ctx, contract.ID, []domain.Clause{
		{ID: clauseID, ContractID: contract.ID, SequenceNumber: 1, Text: "test clause"},
	})
	if err != nil {
		t.Fatalf("save clauses: %v", err)
	}

	runID := "run-persist-" + contract.ID
	agentRunID := "agentrun-persist-" + contract.ID

	t.Run("StartRun and FinishRun", func(t *testing.T) {
		if err := s.StartRun(ctx, runID, contract.ID); err != nil {
			t.Fatalf("start run: %v", err)
		}
		if err := s.FinishRun(ctx, runID, "completed"); err != nil {
			t.Fatalf("finish run: %v", err)
		}
	})

	t.Run("StartAgentRun", func(t *testing.T) {
		if err := s.StartAgentRun(ctx, agentRunID, clauseID, runID); err != nil {
			t.Fatalf("start agent run: %v", err)
		}
	})

	t.Run("AppendAgentStep", func(t *testing.T) {
		msgs := []byte(`[{"Role":"user","Content":"hello"}]`)
		usage := []byte(`{"input":10,"output":5}`)
		if err := s.AppendAgentStep(ctx, agentRunID, 0, msgs, usage); err != nil {
			t.Fatalf("append step 0: %v", err)
		}
		if err := s.AppendAgentStep(ctx, agentRunID, 1, msgs, usage); err != nil {
			t.Fatalf("append step 1: %v", err)
		}
	})

	t.Run("LoadAgentRun running", func(t *testing.T) {
		run, steps, found, err := s.LoadAgentRun(ctx, clauseID)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if run.Status != "running" {
			t.Errorf("expected running, got %q", run.Status)
		}
		if len(steps) != 2 {
			t.Errorf("expected 2 steps, got %d", len(steps))
		}
	})

	t.Run("FinishAgentRun and LoadAgentRun submitted", func(t *testing.T) {
		if err := s.FinishAgentRun(ctx, agentRunID, "submitted", 2, 30, 0.001); err != nil {
			t.Fatalf("finish: %v", err)
		}
		run, _, found, err := s.LoadAgentRun(ctx, clauseID)
		if err != nil {
			t.Fatalf("load after finish: %v", err)
		}
		if !found {
			t.Fatal("expected found=true after finish")
		}
		if run.Status != "submitted" {
			t.Errorf("expected submitted, got %q", run.Status)
		}
	})

	t.Run("LoadAgentRun not found", func(t *testing.T) {
		_, _, found, err := s.LoadAgentRun(ctx, "nonexistent-clause-xyz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Error("expected found=false for nonexistent clause")
		}
	})
}

func TestCreateContractWithOptions(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	got, err := s.CreateContractWithOptions(ctx, "review.pdf", "text", true)
	if err != nil {
		t.Fatalf("CreateContractWithOptions: %v", err)
	}
	if !got.RequiresReview {
		t.Error("expected RequiresReview=true")
	}

	fetched, err := s.GetContract(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if !fetched.RequiresReview {
		t.Error("expected fetched RequiresReview=true")
	}

	noReview, err := s.CreateContractWithOptions(ctx, "plain.pdf", "text", false)
	if err != nil {
		t.Fatalf("CreateContractWithOptions (false): %v", err)
	}
	if noReview.RequiresReview {
		t.Error("expected RequiresReview=false")
	}
}
