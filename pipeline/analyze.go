package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mhihasan/contract-review-ai-agent/agent"
	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/store"
	"github.com/mhihasan/contract-review-ai-agent/tool"
)

func AnalyzeClauses(ctx context.Context, client llm.LLM, s store.Store, contractID string, maxSteps int) error {
	clauses, err := s.GetClauses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get clauses: %w", err)
	}

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusAnalyzing); err != nil {
		return fmt.Errorf("update status to analyzing: %w", err)
	}

	reg := tool.NewRegistry(
		tool.NewGetDefinition(s, contractID),
		tool.NewGetContractSection(s, contractID),
		tool.NewSearchClauseLibrary(s, contractID),
		tool.NewLookupStandardClause(s, contractID),
	)

	for _, clause := range clauses {
		a := agent.New(client, reg, maxSteps, nil)
		result, err := a.Run(ctx, agent.AnalyzeClauseTask{
			ContractID: contractID,
			ClauseID:   clause.ID,
			ClauseText: clause.Text,
		})
		if err != nil {
			return fmt.Errorf("agent run for clause %s: %w", clause.ID, err)
		}

		slog.Info("clause analysed", "clause_id", clause.ID, "stop", result.Stop, "steps", result.Steps)

		if result.Stop != "submitted" {
			slog.Warn("clause not submitted", "clause_id", clause.ID, "stop", result.Stop, "steps", result.Steps)
			continue
		}
		result.Finding.ClauseID = clause.ID
		if err := s.SaveAnalysis(ctx, result.Finding); err != nil {
			return fmt.Errorf("save analysis for clause %s: %w", clause.ID, err)
		}
	}

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusAnalyzed); err != nil {
		return fmt.Errorf("update status to analyzed: %w", err)
	}

	return nil
}
