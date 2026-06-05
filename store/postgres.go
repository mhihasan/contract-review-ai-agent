package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (s *PostgresStore) UpdateContractText(ctx context.Context, id, rawText string) error {
	if err := s.q.UpdateContractText(ctx, db.UpdateContractTextParams{
		ID:      id,
		RawText: rawText,
	}); err != nil {
		return fmt.Errorf("update contract text: %w", err)
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

func (s *PostgresStore) SearchClauseLibrary(ctx context.Context, query string) ([]domain.LibraryClause, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, clause_type, standard_text, notes FROM clause_library
         WHERE clause_type ILIKE $1 OR standard_text ILIKE $1
         ORDER BY clause_type
         LIMIT 5`,
		"%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search clause library: %w", err)
	}
	defer rows.Close()

	var out []domain.LibraryClause
	for rows.Next() {
		var c domain.LibraryClause
		if err := rows.Scan(&c.ID, &c.ClauseType, &c.StandardText, &c.Notes); err != nil {
			return nil, fmt.Errorf("scan clause library row: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search clause library rows: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) GetStandardClause(ctx context.Context, clauseType string) (domain.LibraryClause, error) {
	var c domain.LibraryClause
	err := s.pool.QueryRow(ctx,
		`SELECT id, clause_type, standard_text, notes FROM clause_library WHERE clause_type = $1 LIMIT 1`,
		clauseType,
	).Scan(&c.ID, &c.ClauseType, &c.StandardText, &c.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.LibraryClause{}, fmt.Errorf("get standard clause: %w", ErrNotFound)
		}
		return domain.LibraryClause{}, fmt.Errorf("get standard clause: %w", err)
	}
	return c, nil
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

func (s *PostgresStore) StartRun(ctx context.Context, id, contractID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO runs (id, contract_id, status) VALUES ($1, $2, 'running')`,
		id, contractID,
	)
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}
	return nil
}

func (s *PostgresStore) FinishRun(ctx context.Context, id, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE runs SET status=$2, ended_at=now() WHERE id=$1`,
		id, status,
	)
	if err != nil {
		return fmt.Errorf("finish run: %w", err)
	}
	return nil
}

func (s *PostgresStore) StartAgentRun(ctx context.Context, id, clauseID, runID string) error {
	var runIDVal interface{} = runID
	if runID == "" {
		runIDVal = nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO agent_runs (id, clause_id, run_id, status) VALUES ($1, $2, $3, 'running')`,
		id, clauseID, runIDVal,
	)
	if err != nil {
		return fmt.Errorf("start agent run: %w", err)
	}
	return nil
}

func (s *PostgresStore) AppendAgentStep(ctx context.Context, agentRunID string, stepIndex int, messagesJSON, usageJSON []byte) error {
	stepID := uuid.New().String()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO agent_steps (id, agent_run_id, step_index, messages_json, usage_json)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (agent_run_id, step_index) DO UPDATE
           SET messages_json=EXCLUDED.messages_json, usage_json=EXCLUDED.usage_json`,
		stepID, agentRunID, stepIndex, messagesJSON, usageJSON,
	)
	if err != nil {
		return fmt.Errorf("append agent step: %w", err)
	}
	return nil
}

func (s *PostgresStore) FinishAgentRun(ctx context.Context, id, status string, stepCount, usedTokens int, usedCostUSD float64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE agent_runs
         SET status=$2, ended_at=now(), step_count=$3, used_tokens=$4, used_cost_usd=$5
         WHERE id=$1`,
		id, status, stepCount, usedTokens, usedCostUSD,
	)
	if err != nil {
		return fmt.Errorf("finish agent run: %w", err)
	}
	return nil
}

func (s *PostgresStore) LoadAgentRun(ctx context.Context, clauseID string) (AgentRun, []AgentStep, bool, error) {
	var run AgentRun
	err := s.pool.QueryRow(ctx,
		`SELECT id, clause_id, COALESCE(run_id,''), status, step_count, used_tokens,
                used_cost_usd::float8, started_at, ended_at
         FROM agent_runs
         WHERE clause_id=$1
         ORDER BY started_at DESC LIMIT 1`,
		clauseID,
	).Scan(&run.ID, &run.ClauseID, &run.RunID, &run.Status, &run.StepCount,
		&run.UsedTokens, &run.UsedCostUSD, &run.StartedAt, &run.EndedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentRun{}, nil, false, nil
		}
		return AgentRun{}, nil, false, fmt.Errorf("load agent run: %w", err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, agent_run_id, step_index, messages_json, usage_json, created_at
         FROM agent_steps WHERE agent_run_id=$1 ORDER BY step_index`,
		run.ID,
	)
	if err != nil {
		return AgentRun{}, nil, false, fmt.Errorf("load agent steps: %w", err)
	}
	defer rows.Close()

	var steps []AgentStep
	for rows.Next() {
		var step AgentStep
		if err := rows.Scan(&step.ID, &step.AgentRunID, &step.StepIndex,
			&step.Messages, &step.UsageJSON, &step.CreatedAt); err != nil {
			return AgentRun{}, nil, false, fmt.Errorf("scan agent step: %w", err)
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return AgentRun{}, nil, false, fmt.Errorf("agent steps rows: %w", err)
	}

	return run, steps, true, nil
}

func (s *PostgresStore) GetStoredFinding(ctx context.Context, clauseID string) (domain.ClauseAnalysis, error) {
	var a domain.ClauseAnalysis
	var riskLevel pgtype.Text
	var explanation, ambiguous, recommendations pgtype.Text
	err := s.pool.QueryRow(ctx,
		`SELECT id, clause_id, risk_level, explanation, ambiguous_language, recommendations, status
         FROM clause_analyses WHERE clause_id=$1 ORDER BY id DESC LIMIT 1`,
		clauseID,
	).Scan(&a.ID, &a.ClauseID, &riskLevel, &explanation, &ambiguous, &recommendations, &a.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ClauseAnalysis{}, ErrNotFound
		}
		return domain.ClauseAnalysis{}, fmt.Errorf("get stored finding: %w", err)
	}
	if riskLevel.Valid {
		rl := domain.RiskLevel(riskLevel.String)
		a.RiskLevel = &rl
	}
	if explanation.Valid {
		a.Explanation = explanation.String
	}
	if ambiguous.Valid {
		a.AmbiguousLanguage = ambiguous.String
	}
	if recommendations.Valid {
		a.Recommendations = recommendations.String
	}
	return a, nil
}
