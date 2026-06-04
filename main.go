package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/mhihasan/contract-review-ai-agent/config"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/pdf"
	"github.com/mhihasan/contract-review-ai-agent/pipeline"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	ctx := context.Background()

	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	s := store.NewPostgresStore(pool)

	client, err := llm.New(cfg)
	if err != nil {
		slog.Error("llm init failed", "error", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: contract-review-ai-agent <command> [args]")
		fmt.Fprintln(os.Stderr, "commands:")
		fmt.Fprintln(os.Stderr, "  process <path/to/contract.pdf>   run the full pipeline")
		fmt.Fprintln(os.Stderr, "  extract <path/to/contract.pdf>   debug: PDF extraction only")
		fmt.Fprintln(os.Stderr, "  extract-clauses <contract_id>    debug: clause splitting only")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "process":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: process <path/to/contract.pdf>")
			os.Exit(1)
		}
		if err := runProcess(ctx, cfg, client, s, os.Args[2]); err != nil {
			slog.Error("process failed", "error", err)
			os.Exit(1)
		}

	case "extract":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: extract <path/to/contract.pdf>")
			os.Exit(1)
		}
		id, err := pipeline.RunExtract(ctx, s, pdf.ExtractText, os.Args[2])
		if err != nil {
			if errors.Is(err, pdf.ErrNotPDF) {
				slog.Error("not a PDF file", "path", os.Args[2])
				os.Exit(1)
			}
			slog.Error("extract failed", "error", err)
			os.Exit(1)
		}
		slog.Info("extracted", "contract_id", id)

	case "extract-clauses":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: extract-clauses <contract_id>")
			os.Exit(1)
		}
		if err := pipeline.ExtractClauses(ctx, client, s, os.Args[2]); err != nil {
			slog.Error("extract-clauses failed", "error", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runProcess(ctx context.Context, _ config.Config, client llm.LLM, s store.Store, pdfPath string) error {
	contractID, err := pipeline.RunExtract(ctx, s, pdf.ExtractText, pdfPath)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	slog.Info("extracted", "contract_id", contractID)

	if err := pipeline.ExtractClauses(ctx, client, s, contractID); err != nil {
		return fmt.Errorf("extract-clauses: %w", err)
	}
	slog.Info("clauses extracted", "contract_id", contractID)

	return nil
}
