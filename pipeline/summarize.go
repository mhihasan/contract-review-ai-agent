package pipeline

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/prompts"
	"github.com/mhihasan/contract-review-ai-agent/store"
	"github.com/mhihasan/contract-review-ai-agent/tokens"
)

func RunSummarize(ctx context.Context, s store.Store, client llm.LLM, contractID string, clauseTokenBudget int, model string) error {
	contract, err := s.GetContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}

	if contract.Status == domain.StatusDone {
		summary, err := s.GetSummary(ctx, contractID)
		if err != nil {
			return fmt.Errorf("get existing summary: %w", err)
		}
		fmt.Print(summary.Content)
		return nil
	}

	if contract.Status != domain.StatusReviewComplete {
		return fmt.Errorf("contract %s has status %s, expected review_complete", contractID, contract.Status)
	}

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusSummarizing); err != nil {
		return fmt.Errorf("set summarizing: %w", err)
	}

	clauses, err := s.GetClauses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get clauses: %w", err)
	}
	sort.Slice(clauses, func(i, j int) bool {
		return clauses[i].SequenceNumber < clauses[j].SequenceNumber
	})

	analyses, err := s.GetAnalyses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get analyses: %w", err)
	}
	analysisByClause := make(map[string]domain.ClauseAnalysis, len(analyses))
	for _, a := range analyses {
		analysisByClause[a.ClauseID] = a
	}

	reviews, err := s.GetReviews(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get reviews: %w", err)
	}
	reviewByClause := make(map[string]domain.Review, len(reviews))
	for _, r := range reviews {
		reviewByClause[r.ClauseID] = r
	}

	riskCounts, reviewCounts, clauseInputs := buildInputs(clauses, analysisByClause, reviewByClause, model, clauseTokenBudget)

	p := prompts.SummarizationPrompt{
		Filename:     contract.Filename,
		RiskCounts:   riskCounts,
		ReviewCounts: reviewCounts,
		ClauseInputs: clauseInputs,
	}

	resp, err := client.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: p.Render()},
		},
		MaxTokens:   4096,
		Temperature: 0.2,
	})
	if err != nil {
		return fmt.Errorf("llm complete: %w", err)
	}

	content := resp.Content

	if err := s.SaveSummary(ctx, contractID, content); err != nil {
		return fmt.Errorf("save summary: %w", err)
	}

	filename := fmt.Sprintf("summary_%s.md", contractID)
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Print(content)

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusDone); err != nil {
		return fmt.Errorf("set done: %w", err)
	}

	return nil
}

func buildInputs(
	clauses []domain.Clause,
	analysisByClause map[string]domain.ClauseAnalysis,
	reviewByClause map[string]domain.Review,
	model string,
	clauseTokenBudget int,
) (prompts.RiskCounts, prompts.ReviewCounts, []prompts.ClauseInput) {
	var rc prompts.RiskCounts
	var rvc prompts.ReviewCounts
	inputs := make([]prompts.ClauseInput, 0, len(clauses))

	for _, clause := range clauses {
		analysis := analysisByClause[clause.ID]
		review := reviewByClause[clause.ID]

		riskStr := "unknown"
		if analysis.RiskLevel != nil {
			riskStr = string(*analysis.RiskLevel)
			switch *analysis.RiskLevel {
			case domain.RiskHigh:
				rc.High++
			case domain.RiskMedium:
				rc.Medium++
			case domain.RiskLow:
				rc.Low++
			}
		}

		decision := review.Decision
		switch decision {
		case "approved":
			rvc.Approved++
		case "rejected":
			rvc.Rejected++
		}

		if isOverride(analysis.RiskLevel, decision) {
			rvc.Overrides++
		}

		gist := truncateToTokenBudget(analysis.Explanation, model, clauseTokenBudget)

		inputs = append(inputs, prompts.ClauseInput{
			SequenceNumber: clause.SequenceNumber,
			Gist:           gist,
			RiskLevel:      riskStr,
			Decision:       decision,
			Annotation:     review.Annotation,
		})
	}

	return rc, rvc, inputs
}

func isOverride(risk *domain.RiskLevel, decision string) bool {
	if risk == nil {
		return false
	}
	if *risk == domain.RiskHigh && decision == "approved" {
		return true
	}
	if *risk == domain.RiskLow && decision == "rejected" {
		return true
	}
	return false
}

func truncateToTokenBudget(s, model string, budget int) string {
	if tokens.Count(model, s) <= budget {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		candidate := string(runes)
		if tokens.Count(model, candidate) <= budget {
			return candidate + "…"
		}
		runes = runes[:len(runes)-1]
	}
	return ""
}
