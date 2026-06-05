package store

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/mhihasan/contract-review-ai-agent/domain"
)

type FakeStore struct {
	mu        sync.Mutex
	contracts map[string]domain.Contract
	clauses   map[string][]domain.Clause
	analyses  map[string]domain.ClauseAnalysis
	reviews   map[string]domain.Review
	summaries map[string]domain.Summary
}

func NewFakeStore() *FakeStore {
	return &FakeStore{
		contracts: make(map[string]domain.Contract),
		clauses:   make(map[string][]domain.Clause),
		analyses:  make(map[string]domain.ClauseAnalysis),
		reviews:   make(map[string]domain.Review),
		summaries: make(map[string]domain.Summary),
	}
}

func (f *FakeStore) CreateContract(ctx context.Context, filename, rawText string) (domain.Contract, error) {
	return f.CreateContractWithOptions(ctx, filename, rawText, false)
}

func (f *FakeStore) CreateContractWithOptions(_ context.Context, filename, rawText string, requiresReview bool) (domain.Contract, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := domain.Contract{
		ID:             uuid.New().String(),
		Filename:       filename,
		RawText:        rawText,
		Status:         domain.StatusUploaded,
		RequiresReview: requiresReview,
	}
	f.contracts[c.ID] = c
	return c, nil
}

func (f *FakeStore) GetContract(_ context.Context, id string) (domain.Contract, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.contracts[id]
	if !ok {
		return domain.Contract{}, ErrNotFound
	}
	return c, nil
}

func (f *FakeStore) UpdateContractStatus(_ context.Context, id string, status domain.ContractStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.contracts[id]
	if !ok {
		return ErrNotFound
	}
	c.Status = status
	f.contracts[id] = c
	return nil
}

func (f *FakeStore) UpdateContractText(_ context.Context, id, rawText string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.contracts[id]
	if !ok {
		return ErrNotFound
	}
	c.RawText = rawText
	f.contracts[id] = c
	return nil
}

func (f *FakeStore) SaveClauses(_ context.Context, contractID string, clauses []domain.Clause) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clauses[contractID] = clauses
	return nil
}

func (f *FakeStore) GetClauses(_ context.Context, contractID string) ([]domain.Clause, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clauses[contractID], nil
}

func (f *FakeStore) SaveAnalysis(_ context.Context, a domain.ClauseAnalysis) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.analyses[a.ClauseID] = a
	return nil
}

func (f *FakeStore) GetAnalyses(_ context.Context, contractID string) ([]domain.ClauseAnalysis, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	clauses := f.clauses[contractID]
	var result []domain.ClauseAnalysis
	for _, c := range clauses {
		if a, ok := f.analyses[c.ID]; ok {
			result = append(result, a)
		}
	}
	return result, nil
}

func (f *FakeStore) SaveReview(_ context.Context, r domain.Review) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviews[r.ClauseID] = r
	return nil
}

func (f *FakeStore) GetReviews(_ context.Context, contractID string) ([]domain.Review, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	clauses := f.clauses[contractID]
	var result []domain.Review
	for _, c := range clauses {
		if r, ok := f.reviews[c.ID]; ok {
			result = append(result, r)
		}
	}
	return result, nil
}

func (f *FakeStore) SaveSummary(_ context.Context, contractID, content string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.summaries[contractID] = domain.Summary{
		ID:         uuid.New().String(),
		ContractID: contractID,
		Content:    content,
	}
	return nil
}

func (f *FakeStore) GetSummary(_ context.Context, contractID string) (domain.Summary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.summaries[contractID]
	if !ok {
		return domain.Summary{}, ErrNotFound
	}
	return s, nil
}

func (f *FakeStore) SearchClauseLibrary(_ context.Context, _ string) ([]domain.LibraryClause, error) {
	return nil, nil
}

func (f *FakeStore) GetStandardClause(_ context.Context, _ string) (domain.LibraryClause, error) {
	return domain.LibraryClause{}, ErrNotFound
}

func (f *FakeStore) StartRun(_ context.Context, _, _ string) error {
	return nil
}

func (f *FakeStore) FinishRun(_ context.Context, _, _ string) error {
	return nil
}

func (f *FakeStore) StartAgentRun(_ context.Context, _, _, _ string) error {
	return nil
}

func (f *FakeStore) AppendAgentStep(_ context.Context, _ string, _ int, _, _ []byte) error {
	return nil
}

func (f *FakeStore) FinishAgentRun(_ context.Context, _ string, _ string, _, _ int, _ float64) error {
	return nil
}

func (f *FakeStore) LoadAgentRun(_ context.Context, _ string) (AgentRun, []AgentStep, bool, error) {
	return AgentRun{}, nil, false, nil
}

func (f *FakeStore) GetStoredFinding(_ context.Context, _ string) (domain.ClauseAnalysis, error) {
	return domain.ClauseAnalysis{}, ErrNotFound
}

var _ Store = (*FakeStore)(nil)
