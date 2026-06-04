package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/mhihasan/contract-review-ai-agent/config"
	"github.com/mhihasan/contract-review-ai-agent/pdf"
	"github.com/mhihasan/contract-review-ai-agent/pipeline"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

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

	switch os.Args[1] {
	case "extract":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
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
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <command> [args]\n\ncommands:\n  extract <path/to/contract.pdf>\n", os.Args[0])
}
