package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/store/db"
)

var _ Store = (*PostgresStore)(nil)

type PostgresStore struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{q: db.New(pool), pool: pool}
}

func (s *PostgresStore) CreateContract(ctx context.Context, filename, rawText string) (domain.Contract, error) {
	row, err := s.q.CreateContract(ctx, db.CreateContractParams{
		ID:       uuid.New().String(),
		Filename: filename,
		RawText:  rawText,
		Status:   domain.StatusUploaded.String(),
	})
	if err != nil {
		return domain.Contract{}, fmt.Errorf("create contract: %w", err)
	}
	return toDomainContract(row), nil
}

func (s *PostgresStore) GetContract(ctx context.Context, id string) (domain.Contract, error) {
	row, err := s.q.GetContract(ctx, id)
	if err != nil {
		return domain.Contract{}, fmt.Errorf("get contract: %w", err)
	}
	return toDomainContract(row), nil
}

func (s *PostgresStore) UpdateContractStatus(ctx context.Context, id string, status domain.ContractStatus) error {
	if err := s.q.UpdateContractStatus(ctx, db.UpdateContractStatusParams{
		ID:     id,
		Status: status.String(),
	}); err != nil {
		return fmt.Errorf("update contract status: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveClauses(ctx context.Context, contractID string, clauses []domain.Clause) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.q.WithTx(tx)
	for _, c := range clauses {
		if _, err := q.CreateClause(ctx, db.CreateClauseParams{
			ID:             c.ID,
			ContractID:     contractID,
			SequenceNumber: int32(c.SequenceNumber),
			Text:           c.Text,
		}); err != nil {
			return fmt.Errorf("create clause: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit clauses: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetClauses(ctx context.Context, contractID string) ([]domain.Clause, error) {
	rows, err := s.q.GetClausesByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("get clauses: %w", err)
	}
	out := make([]domain.Clause, len(rows))
	for i, r := range rows {
		out[i] = toDomainClause(r)
	}
	return out, nil
}

func (s *PostgresStore) SaveAnalysis(ctx context.Context, a domain.ClauseAnalysis) error {
	riskLevel := pgtype.Text{Valid: false}
	if a.RiskLevel != nil {
		riskLevel = pgtype.Text{String: a.RiskLevel.String(), Valid: true}
	}
	if _, err := s.q.CreateClauseAnalysis(ctx, db.CreateClauseAnalysisParams{
		ID:                a.ID,
		ClauseID:          a.ClauseID,
		RiskLevel:         riskLevel,
		Explanation:       pgtype.Text{String: a.Explanation, Valid: a.Explanation != ""},
		AmbiguousLanguage: pgtype.Text{String: a.AmbiguousLanguage, Valid: a.AmbiguousLanguage != ""},
		Recommendations:   pgtype.Text{String: a.Recommendations, Valid: a.Recommendations != ""},
		Status:            a.Status,
	}); err != nil {
		return fmt.Errorf("save analysis: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetAnalyses(ctx context.Context, contractID string) ([]domain.ClauseAnalysis, error) {
	rows, err := s.q.GetAnalysesByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("get analyses: %w", err)
	}
	out := make([]domain.ClauseAnalysis, len(rows))
	for i, r := range rows {
		out[i] = toDomainAnalysis(r)
	}
	return out, nil
}

func (s *PostgresStore) SaveReview(ctx context.Context, r domain.Review) error {
	if _, err := s.q.CreateReview(ctx, db.CreateReviewParams{
		ID:         r.ID,
		ClauseID:   r.ClauseID,
		Decision:   r.Decision,
		Annotation: pgtype.Text{String: r.Annotation, Valid: r.Annotation != ""},
	}); err != nil {
		return fmt.Errorf("save review: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetReviews(ctx context.Context, contractID string) ([]domain.Review, error) {
	rows, err := s.q.GetReviewsByContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("get reviews: %w", err)
	}
	out := make([]domain.Review, len(rows))
	for i, r := range rows {
		out[i] = toDomainReview(r)
	}
	return out, nil
}

func (s *PostgresStore) SaveSummary(ctx context.Context, contractID, content string) error {
	if _, err := s.q.UpsertSummary(ctx, db.UpsertSummaryParams{
		ID:         uuid.New().String(),
		ContractID: contractID,
		Content:    content,
	}); err != nil {
		return fmt.Errorf("save summary: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetSummary(ctx context.Context, contractID string) (domain.Summary, error) {
	row, err := s.q.GetSummaryByContract(ctx, contractID)
	if err != nil {
		return domain.Summary{}, fmt.Errorf("get summary: %w", err)
	}
	return toDomainSummary(row), nil
}

func toDomainContract(r db.Contract) domain.Contract {
	return domain.Contract{
		ID:        r.ID,
		Filename:  r.Filename,
		RawText:   r.RawText,
		Status:    domain.ContractStatus(r.Status),
		CreatedAt: r.CreatedAt.Time,
	}
}

func toDomainClause(r db.Clause) domain.Clause {
	return domain.Clause{
		ID:             r.ID,
		ContractID:     r.ContractID,
		SequenceNumber: int(r.SequenceNumber),
		Text:           r.Text,
	}
}

func toDomainAnalysis(r db.ClauseAnalysis) domain.ClauseAnalysis {
	a := domain.ClauseAnalysis{
		ID:       r.ID,
		ClauseID: r.ClauseID,
		Status:   r.Status,
	}
	if r.RiskLevel.Valid {
		rl := domain.RiskLevel(r.RiskLevel.String)
		a.RiskLevel = &rl
	}
	if r.Explanation.Valid {
		a.Explanation = r.Explanation.String
	}
	if r.AmbiguousLanguage.Valid {
		a.AmbiguousLanguage = r.AmbiguousLanguage.String
	}
	if r.Recommendations.Valid {
		a.Recommendations = r.Recommendations.String
	}
	return a
}

func toDomainReview(r db.Review) domain.Review {
	review := domain.Review{
		ID:       r.ID,
		ClauseID: r.ClauseID,
		Decision: r.Decision,
	}
	if r.Annotation.Valid {
		review.Annotation = r.Annotation.String
	}
	return review
}

func toDomainSummary(r db.Summary) domain.Summary {
	return domain.Summary{
		ID:         r.ID,
		ContractID: r.ContractID,
		Content:    r.Content,
		CreatedAt:  r.CreatedAt.Time,
	}
}
