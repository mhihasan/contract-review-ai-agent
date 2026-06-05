package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

type extractFunc func(ctx context.Context, path string) (string, error)

func RunExtract(ctx context.Context, s store.Store, extract extractFunc, path string, requiresReview bool) (string, error) {
	slog.Debug("starting extract", "path", path)
	start := time.Now()
	defer func() { slog.Debug("extract done", "duration_ms", time.Since(start).Milliseconds()) }()

	c, err := s.CreateContractWithOptions(ctx, filepath.Base(path), "", requiresReview)
	if err != nil {
		return "", fmt.Errorf("create contract: %w", err)
	}
	slog.Info("extract created contract", "contract_id", c.ID, "filename", path)
	return c.ID, runExtractStages(ctx, s, extract, c.ID, path)
}

func RunExtractContract(ctx context.Context, s store.Store, extract extractFunc, id, path string) (string, error) {
	c, err := s.GetContract(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get contract: %w", err)
	}
	if statusRank(c.Status) >= statusRank(domain.StatusExtracted) {
		slog.Info("extract skipped, already done", "contract_id", id, "status", c.Status.String())
		return id, nil
	}
	return id, runExtractStages(ctx, s, extract, id, path)
}

func runExtractStages(ctx context.Context, s store.Store, extract extractFunc, id, path string) error {
	if err := s.UpdateContractStatus(ctx, id, domain.StatusExtracting); err != nil {
		return fmt.Errorf("set extracting: %w", err)
	}

	start := time.Now()
	text, err := extract(ctx, path)
	if err != nil {
		return fmt.Errorf("extract text: %w", err)
	}
	slog.Info("extract finished",
		"contract_id", id,
		"duration_ms", time.Since(start).Milliseconds(),
		"char_count", len(text),
	)

	if err := s.UpdateContractText(ctx, id, text); err != nil {
		return fmt.Errorf("persist text: %w", err)
	}
	if err := s.UpdateContractStatus(ctx, id, domain.StatusExtracted); err != nil {
		return fmt.Errorf("set extracted: %w", err)
	}
	slog.Info("extract complete", "contract_id", id, "status", domain.StatusExtracted.String())
	return nil
}

func statusRank(s domain.ContractStatus) int {
	switch s {
	case domain.StatusUploaded:
		return 0
	case domain.StatusExtracting:
		return 1
	case domain.StatusExtracted:
		return 2
	case domain.StatusAnalyzingClauses,
		domain.StatusClausesExtracted,
		domain.StatusAnalyzing,
		domain.StatusAnalyzed,
		domain.StatusReviewPending,
		domain.StatusReviewComplete,
		domain.StatusSummarizing,
		domain.StatusDone:
		return 3
	}
	return -1
}
