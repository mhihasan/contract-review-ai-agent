package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/mhihasan/contract-review-ai-agent/agent"
	"github.com/mhihasan/contract-review-ai-agent/cost"
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
	provider, model string,
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

	slog.Debug("analyzing contract", "contract_id", contractID,
		"total_clauses", len(clauses), "need_analysis", len(toAnalyze))

	if concurrency < 1 {
		concurrency = 1
	}

	var (
		mu          sync.Mutex
		totalTokens int
		totalCost   float64
		okCount     int
		failedCount int
		dispatched  int
	)

	start := time.Now()

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for _, clause := range toAnalyze {
		clause := clause
		g.Go(func() error {
			mu.Lock()
			dispatched++
			idx := dispatched
			mu.Unlock()

			slog.Debug(fmt.Sprintf("[%d/%d] clause %s — starting agent", idx, len(toAnalyze), clause.ID))

			reg := tool.NewRegistry(
				tool.NewGetDefinition(s, contractID),
				tool.NewGetContractSection(s, contractID),
				tool.NewSearchClauseLibrary(s, contractID),
				tool.NewLookupStandardClause(s, contractID),
				tool.NewSubmitFinding(nil),
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
				mu.Lock()
				failedCount++
				mu.Unlock()
				return nil
			}

			clauseTokens := result.Usage.InputTokens + result.Usage.OutputTokens
			clauseCost := cost.Estimate(provider, model, result.Usage.InputTokens, result.Usage.OutputTokens)

			if result.Stop != "submitted" {
				slog.Warn("clause not submitted", "clause_id", clause.ID, "stop", result.Stop, "steps", result.Steps)
				if err := s.SaveAnalysis(ctx, failedAnalysis(clause.ID, result.Stop)); err != nil {
					slog.Error("failed to save failed analysis", "clause_id", clause.ID, "err", err)
				}
				mu.Lock()
				totalTokens += clauseTokens
				totalCost += clauseCost
				failedCount++
				mu.Unlock()
				return nil
			}

			slog.Info(fmt.Sprintf("[%d/%d] clause %s done", idx, len(toAnalyze), clause.ID),
				"stop", result.Stop,
				"steps", result.Steps,
				"tokens", clauseTokens,
				"est_cost_usd", fmt.Sprintf("$%.6f", clauseCost),
			)

			mu.Lock()
			totalTokens += clauseTokens
			totalCost += clauseCost
			okCount++
			mu.Unlock()

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

	slog.Info("run complete",
		"contract_id", contractID,
		"clauses", len(toAnalyze),
		"ok", okCount,
		"failed", failedCount,
		"total_tokens", totalTokens,
		"est_cost_usd", fmt.Sprintf("$%.4f", totalCost),
		"duration_s", fmt.Sprintf("%.1f", time.Since(start).Seconds()),
	)

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
