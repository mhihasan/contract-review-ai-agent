package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

type Decision string

const (
	DecisionApproved Decision = "approved"
	DecisionRejected Decision = "rejected"
	DecisionNote     Decision = "note"
)

func ParseDecision(input string) (Decision, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "a":
		return DecisionApproved, nil
	case "r":
		return DecisionRejected, nil
	case "n":
		return DecisionNote, nil
	default:
		return "", fmt.Errorf("invalid input %q: enter a, r, or n", input)
	}
}

func RunReview(ctx context.Context, s store.Store, contractID string, in io.Reader) error {
	contract, err := s.GetContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}
	if contract.Status != domain.StatusReviewPending {
		return fmt.Errorf("contract %s has status %s, expected review_pending", contractID, contract.Status)
	}

	clauses, err := s.GetClauses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get clauses: %w", err)
	}

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
	reviewed := make(map[string]struct{}, len(reviews))
	for _, r := range reviews {
		reviewed[r.ClauseID] = struct{}{}
	}

	reader := bufio.NewReader(in)
	total := len(clauses)

	for _, clause := range clauses {
		if _, done := reviewed[clause.ID]; done {
			continue
		}

		analysis := analysisByClause[clause.ID]
		riskLabel := "UNKNOWN"
		if analysis.RiskLevel != nil {
			riskLabel = strings.ToUpper(string(*analysis.RiskLevel)) + " RISK"
		}

		fmt.Printf("\nClause %d/%d [%s]\n", clause.SequenceNumber, total, riskLabel)
		fmt.Printf("Text: %q\n", truncate(clause.Text, 200))
		fmt.Printf("Risk: %s\n", analysis.Explanation)
		if analysis.AmbiguousLanguage != "" {
			fmt.Printf("Ambiguous: %s\n", analysis.AmbiguousLanguage)
		}
		if analysis.Recommendations != "" {
			fmt.Printf("Recommendation: %s\n", analysis.Recommendations)
		}

		var annotation string
		var decision Decision

		for {
			fmt.Print("\n[n]ote / Decision [a]pprove / [r]eject: ")
			line, _ := reader.ReadString('\n')
			d, err := ParseDecision(strings.TrimSpace(line))
			if err != nil {
				fmt.Println("  Enter a (approve), r (reject), or n (note then a/r).")
				continue
			}
			if d == DecisionNote {
				fmt.Print("Annotation: ")
				noteLine, _ := reader.ReadString('\n')
				annotation = strings.TrimSpace(noteLine)
				fmt.Print("Decision [a]pprove / [r]eject: ")
				for {
					arLine, _ := reader.ReadString('\n')
					arD, arErr := ParseDecision(strings.TrimSpace(arLine))
					if arErr != nil || arD == DecisionNote {
						fmt.Print("  Enter a (approve) or r (reject): ")
						continue
					}
					decision = arD
					break
				}
			} else {
				decision = d
			}
			break
		}

		review := domain.Review{
			ID:         uuid.New().String(),
			ClauseID:   clause.ID,
			Decision:   string(decision),
			Annotation: annotation,
		}
		if err := s.SaveReview(ctx, review); err != nil {
			return fmt.Errorf("save review for clause %s: %w", clause.ID, err)
		}
		slog.Info("review saved", "clause_id", clause.ID, "decision", decision)
	}

	fmt.Println("\nAll clauses reviewed. Run: go run . resume", contractID)
	return nil
}

type SummarizeFunc func(ctx context.Context, s store.Store, contractID string) error

func RunResume(ctx context.Context, s store.Store, contractID string, summarize SummarizeFunc) error {
	contract, err := s.GetContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}
	if contract.Status != domain.StatusReviewPending {
		return fmt.Errorf("contract %s has status %s, expected review_pending", contractID, contract.Status)
	}

	clauses, err := s.GetClauses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get clauses: %w", err)
	}

	reviews, err := s.GetReviews(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get reviews: %w", err)
	}

	reviewedSet := make(map[string]struct{}, len(reviews))
	for _, r := range reviews {
		reviewedSet[r.ClauseID] = struct{}{}
	}

	var missing int
	for _, c := range clauses {
		if _, ok := reviewedSet[c.ID]; !ok {
			missing++
		}
	}
	if missing > 0 {
		return fmt.Errorf("%d clause(s) still need a decision — run: go run . review %s", missing, contractID)
	}

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusReviewComplete); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	return summarize(ctx, s, contractID)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
