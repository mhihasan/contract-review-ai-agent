package store

import (
	"context"

	"github.com/mhihasan/contract-review-ai-agent/domain"
)

type Store interface {
	CreateContract(ctx context.Context, filename, rawText string) (domain.Contract, error)
	GetContract(ctx context.Context, id string) (domain.Contract, error)
	UpdateContractStatus(ctx context.Context, id string, status domain.ContractStatus) error
	UpdateContractText(ctx context.Context, id, rawText string) error

	SaveClauses(ctx context.Context, contractID string, clauses []domain.Clause) error
	GetClauses(ctx context.Context, contractID string) ([]domain.Clause, error)

	SaveAnalysis(ctx context.Context, a domain.ClauseAnalysis) error
	GetAnalyses(ctx context.Context, contractID string) ([]domain.ClauseAnalysis, error)

	SaveReview(ctx context.Context, r domain.Review) error
	GetReviews(ctx context.Context, contractID string) ([]domain.Review, error)

	SaveSummary(ctx context.Context, contractID, content string) error
	GetSummary(ctx context.Context, contractID string) (domain.Summary, error)

	SearchClauseLibrary(ctx context.Context, query string) ([]domain.LibraryClause, error)
	GetStandardClause(ctx context.Context, clauseType string) (domain.LibraryClause, error)
}
