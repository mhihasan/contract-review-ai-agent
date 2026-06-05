package store

import (
	"context"
	"errors"
	"time"

	"github.com/mhihasan/contract-review-ai-agent/domain"
)

var ErrNotFound = errors.New("not found")

type AgentRun struct {
	ID          string
	ClauseID    string
	RunID       string
	Status      string
	StepCount   int
	UsedTokens  int
	UsedCostUSD float64
	StartedAt   time.Time
	EndedAt     *time.Time
}

type AgentStep struct {
	ID         string
	AgentRunID string
	StepIndex  int
	Messages   []byte
	UsageJSON  []byte
	CreatedAt  time.Time
}

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

	StartRun(ctx context.Context, id, contractID string) error
	FinishRun(ctx context.Context, id, status string) error

	StartAgentRun(ctx context.Context, id, clauseID, runID string) error
	AppendAgentStep(ctx context.Context, agentRunID string, stepIndex int, messagesJSON, usageJSON []byte) error
	FinishAgentRun(ctx context.Context, id, status string, stepCount, usedTokens int, usedCostUSD float64) error
	LoadAgentRun(ctx context.Context, clauseID string) (AgentRun, []AgentStep, bool, error)
	GetStoredFinding(ctx context.Context, clauseID string) (domain.ClauseAnalysis, error)
}
