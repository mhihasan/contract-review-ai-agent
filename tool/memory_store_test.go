package tool_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/mhihasan/contract-review-ai-agent/domain"
)

type MemoryStore struct {
	contracts map[string]domain.Contract
	clauses   map[string][]domain.Clause
	library   []domain.LibraryClause
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		contracts: make(map[string]domain.Contract),
		clauses:   make(map[string][]domain.Clause),
	}
}

func (m *MemoryStore) CreateContract(_ context.Context, _, _ string) (domain.Contract, error) {
	return domain.Contract{}, fmt.Errorf("not implemented")
}

func (m *MemoryStore) GetContract(_ context.Context, id string) (domain.Contract, error) {
	c, ok := m.contracts[id]
	if !ok {
		return domain.Contract{}, fmt.Errorf("contract %q not found", id)
	}
	return c, nil
}

func (m *MemoryStore) UpdateContractStatus(_ context.Context, _ string, _ domain.ContractStatus) error {
	return fmt.Errorf("not implemented")
}

func (m *MemoryStore) UpdateContractText(_ context.Context, _, _ string) error {
	return fmt.Errorf("not implemented")
}

func (m *MemoryStore) SaveClauses(_ context.Context, contractID string, clauses []domain.Clause) error {
	m.clauses[contractID] = clauses
	return nil
}

func (m *MemoryStore) GetClauses(_ context.Context, contractID string) ([]domain.Clause, error) {
	return m.clauses[contractID], nil
}

func (m *MemoryStore) SaveAnalysis(_ context.Context, _ domain.ClauseAnalysis) error {
	return fmt.Errorf("not implemented")
}

func (m *MemoryStore) GetAnalyses(_ context.Context, _ string) ([]domain.ClauseAnalysis, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MemoryStore) SaveReview(_ context.Context, _ domain.Review) error {
	return fmt.Errorf("not implemented")
}

func (m *MemoryStore) GetReviews(_ context.Context, _ string) ([]domain.Review, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MemoryStore) SaveSummary(_ context.Context, _, _ string) error {
	return fmt.Errorf("not implemented")
}

func (m *MemoryStore) GetSummary(_ context.Context, _ string) (domain.Summary, error) {
	return domain.Summary{}, fmt.Errorf("not implemented")
}

func (m *MemoryStore) SearchClauseLibrary(_ context.Context, query string) ([]domain.LibraryClause, error) {
	q := strings.ToLower(query)
	var out []domain.LibraryClause
	for _, c := range m.library {
		if strings.Contains(strings.ToLower(c.ClauseType), q) ||
			strings.Contains(strings.ToLower(c.StandardText), q) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *MemoryStore) GetStandardClause(_ context.Context, clauseType string) (domain.LibraryClause, error) {
	for _, c := range m.library {
		if c.ClauseType == clauseType {
			return c, nil
		}
	}
	return domain.LibraryClause{}, fmt.Errorf("clause type %q not found", clauseType)
}
