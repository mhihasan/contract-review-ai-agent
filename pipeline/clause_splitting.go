package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	openai "github.com/openai/openai-go/v3"

	"github.com/mhihasan/contract-review-ai-agent/domain"
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

func complete(ctx context.Context, client *openai.Client, model, prompt string) (string, error) {
	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}

func extractClausesFromLLM(ctx context.Context, client *openai.Client, model, contractText string) ([]string, error) {
	prompt := prompts.ClauseExtractionPrompt{ContractText: contractText}
	for attempt := 0; attempt < 3; attempt++ {
		raw, err := complete(ctx, client, model, prompt.Render())
		if err != nil {
			return nil, err
		}
		clauses, perr := parseClauses(raw)
		if perr == nil {
			return clauses, nil
		}
	}
	return nil, ErrClauseParse
}

func ExtractClauses(ctx context.Context, client *openai.Client, model string, s store.Store, contractID string) error {
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

	texts, err := extractClausesFromLLM(ctx, client, model, contract.RawText)
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
