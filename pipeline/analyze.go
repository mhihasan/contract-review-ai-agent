package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/mhihasan/contract-review-ai-agent/agent"
	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/store"
	"github.com/mhihasan/contract-review-ai-agent/tool"
)

func AnalyzeClauses(
	ctx context.Context,
	client llm.LLM,
	s store.Store,
	contractID string,
	maxSteps int,
	ctxMgr *agent.ContextManager,
	budget *agent.Budget,
	concurrency int,
) error {
	contract, err := s.GetContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}
	if contract.Status == domain.StatusAnalyzed {
		slog.Info("analyze skipped, already done", "contract_id", contractID, "status", contract.Status)
		return nil
	}
	if contract.Status != domain.StatusClausesExtracted && contract.Status != domain.StatusAnalyzing {
		return fmt.Errorf("contract %s has status %s, expected %s", contractID, contract.Status, domain.StatusClausesExtracted)
	}

	clauses, err := s.GetClauses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get clauses: %w", err)
	}

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusAnalyzing); err != nil {
		return fmt.Errorf("update status to analyzing: %w", err)
	}

	toAnalyze, err := clausesNeedingAnalysis(ctx, s, clauses)
	if err != nil {
		return fmt.Errorf("filter clauses needing analysis: %w", err)
	}
	if concurrency < 1 {
		concurrency = 1
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for _, clause := range toAnalyze {
		clause := clause
		g.Go(func() error {
			reg := tool.NewRegistry(
				tool.NewGetDefinition(s, contractID),
				tool.NewGetContractSection(s, contractID),
				tool.NewSearchClauseLibrary(s, contractID),
				tool.NewLookupStandardClause(s, contractID),
			)

			a := agent.NewWithBudget(client, reg, maxSteps, ctxMgr, budget)
			result, runErr := a.Run(gCtx, agent.AnalyzeClauseTask{
				ContractID: contractID,
				ClauseID:   clause.ID,
				ClauseText: clause.Text,
			})
			if runErr != nil {
				slog.Warn("agent run failed", "clause_id", clause.ID, "err", runErr)
				if err := s.SaveAnalysis(ctx, failedAnalysis(clause.ID, runErr.Error())); err != nil {
					slog.Error("failed to save failed analysis", "clause_id", clause.ID, "err", err)
				}
				return nil
			}

			slog.Info("clause analysed", "clause_id", clause.ID, "stop", result.Stop, "steps", result.Steps)

			if result.Stop != "submitted" {
				slog.Warn("clause not submitted", "clause_id", clause.ID, "stop", result.Stop, "steps", result.Steps)
				if err := s.SaveAnalysis(ctx, failedAnalysis(clause.ID, result.Stop)); err != nil {
					slog.Error("failed to save failed analysis", "clause_id", clause.ID, "err", err)
				}
				return nil
			}

			finding := result.Finding
			finding.ID = uuid.New().String()
			finding.ClauseID = clause.ID
			finding.Status = "ok"
			if err := s.SaveAnalysis(ctx, finding); err != nil {
				slog.Error("failed to save analysis", "clause_id", clause.ID, "err", err)
			}
			return nil
		})
	}

	_ = g.Wait()

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusAnalyzed); err != nil {
		return fmt.Errorf("update status to analyzed: %w", err)
	}

	return nil
}

func clausesNeedingAnalysis(ctx context.Context, s store.Store, clauses []domain.Clause) ([]domain.Clause, error) {
	if len(clauses) == 0 {
		return nil, nil
	}

	analyses, err := s.GetAnalyses(ctx, clauses[0].ContractID)
	if err != nil {
		return nil, err
	}

	alreadyAnalyzed := make(map[string]struct{}, len(analyses))
	for _, a := range analyses {
		if a.Status == "analyzed" || a.Status == "ok" {
			alreadyAnalyzed[a.ClauseID] = struct{}{}
		}
	}

	toAnalyze := make([]domain.Clause, 0, len(clauses))
	for _, c := range clauses {
		if _, exists := alreadyAnalyzed[c.ID]; exists {
			continue
		}
		toAnalyze = append(toAnalyze, c)
	}

	return toAnalyze, nil
}

func failedAnalysis(clauseID, reason string) domain.ClauseAnalysis {
	return domain.ClauseAnalysis{
		ID:          uuid.New().String(),
		ClauseID:    clauseID,
		Status:      "failed",
		Explanation: reason,
	}
}
