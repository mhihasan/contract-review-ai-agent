package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/prompts"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

var ErrClauseParse = errors.New("clause extraction: could not parse JSON after retries")

func parseClauses(raw string) ([]string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	var clauses []string
	if err := json.Unmarshal([]byte(s), &clauses); err != nil {
		return nil, err
	}
	if len(clauses) == 0 {
		return nil, errors.New("model returned empty clause array")
	}
	return clauses, nil
}

func extractClausesFromLLM(ctx context.Context, client llm.LLM, contractText string) ([]string, error) {
	prompt := prompts.ClauseExtractionPrompt{ContractText: contractText}
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := client.Complete(ctx, llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: llm.RoleUser, Content: prompt.Render()},
			},
			MaxTokens:   8192,
			Temperature: 0.2,
			Timeout:     5 * time.Minute,
		})
		if err != nil {
			return nil, err
		}
		clauses, perr := parseClauses(resp.Content)
		if perr == nil {
			return clauses, nil
		}
	}
	return nil, ErrClauseParse
}

func ExtractClauses(ctx context.Context, client llm.LLM, s store.Store, contractID string) error {
	slog.Debug("starting clause split", "contract_id", contractID)
	start := time.Now()
	defer func() { slog.Debug("clause split done", "duration_ms", time.Since(start).Milliseconds()) }()

	contract, err := s.GetContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}

	if contract.Status == domain.StatusClausesExtracted {
		fmt.Printf("contract %s already has clauses extracted — no-op\n", contractID)
		return nil
	}
	if contract.Status != domain.StatusExtracted {
		return fmt.Errorf("contract %s has status %s, expected %s", contractID, contract.Status, domain.StatusExtracted)
	}

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusAnalyzingClauses); err != nil {
		return fmt.Errorf("update status to analyzing_clauses: %w", err)
	}

	texts, err := extractClausesFromLLM(ctx, client, contract.RawText)
	if err != nil {
		return fmt.Errorf("extract clauses from llm: %w", err)
	}

	clauses := make([]domain.Clause, len(texts))
	for i, text := range texts {
		clauses[i] = domain.Clause{
			ID:             uuid.New().String(),
			ContractID:     contractID,
			SequenceNumber: i + 1,
			Text:           text,
		}
	}

	if err := s.SaveClauses(ctx, contractID, clauses); err != nil {
		return fmt.Errorf("save clauses: %w", err)
	}

	if err := s.UpdateContractStatus(ctx, contractID, domain.StatusClausesExtracted); err != nil {
		return fmt.Errorf("update status to clauses_extracted: %w", err)
	}

	fmt.Printf("extracted %d clauses from contract %s\n", len(clauses), contractID)
	return nil
}
