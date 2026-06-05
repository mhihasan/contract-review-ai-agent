package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/mhihasan/contract-review-ai-agent/agent"
	"github.com/mhihasan/contract-review-ai-agent/config"
	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/pdf"
	"github.com/mhihasan/contract-review-ai-agent/pipeline"
	"github.com/mhihasan/contract-review-ai-agent/store"
	"github.com/mhihasan/contract-review-ai-agent/tool"
)

const defaultMaxSteps = 12

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		fmt.Fprintln(os.Stderr, "  process <path/to/contract.pdf> [--review]  run the full pipeline")
		fmt.Fprintln(os.Stderr, "  review <contract_id>                        review clauses interactively")
		fmt.Fprintln(os.Stderr, "  resume <contract_id>                        complete review and advance")
		fmt.Fprintln(os.Stderr, "  extract <path/to/contract.pdf>              debug: PDF extraction only")
		fmt.Fprintln(os.Stderr, "  extract-clauses <contract_id>               debug: clause splitting only")
		fmt.Fprintln(os.Stderr, "  analyze <contract_id>                       run analysis across all clauses")
		fmt.Fprintln(os.Stderr, "  analyze-clause <contract_id> <clause_id>    debug: run agent on one clause")
		fmt.Fprintln(os.Stderr, "  status <contract_id>                        show contract and clause agent_run states")
		fmt.Fprintln(os.Stderr, "  summarize <contract_id>                     generate the final summary report")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "process":
		fs := flag.NewFlagSet("process", flag.ExitOnError)
		requiresReview := fs.Bool("review", false, "pause for human review after analysis")
		_ = fs.Parse(os.Args[2:])
		if len(fs.Args()) < 1 {
			fmt.Fprintln(os.Stderr, "usage: process <path/to/contract.pdf> [--review]")
			os.Exit(1)
		}
		if err := runProcess(ctx, cfg, client, s, fs.Args()[0], *requiresReview); err != nil {
			slog.Error("process failed", "error", err)
			os.Exit(1)
		}

	case "review":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: review <contract_id>")
			os.Exit(1)
		}
		if err := pipeline.RunReview(ctx, s, os.Args[2], os.Stdin); err != nil {
			slog.Error("review failed", "error", err)
			os.Exit(1)
		}

	case "resume":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: resume <contract_id>")
			os.Exit(1)
		}
		if err := pipeline.RunResume(ctx, s, os.Args[2], func(ctx context.Context, s store.Store, id string) error {
			return pipeline.RunSummarize(ctx, s, client, id, cfg.SummaryClauseTokenBudget, cfg.LLMModel, ".")
		}); err != nil {
			slog.Error("resume failed", "error", err)
			os.Exit(1)
		}

	case "extract":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: extract <path/to/contract.pdf>")
			os.Exit(1)
		}
		id, err := pipeline.RunExtract(ctx, s, pdf.ExtractText, os.Args[2], false)
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

	case "analyze":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: analyze <contract_id>")
			os.Exit(1)
		}
		if err := runAnalyze(ctx, cfg, client, s, os.Args[2]); err != nil {
			slog.Error("analyze failed", "error", err)
			os.Exit(1)
		}

	case "analyze-clause":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: analyze-clause <contract_id> <clause_id>")
			os.Exit(1)
		}
		if err := runAnalyzeClause(ctx, cfg, client, s, os.Args[2], os.Args[3]); err != nil {
			slog.Error("analyze-clause failed", "error", err)
			os.Exit(1)
		}

	case "status":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: status <contract_id>")
			os.Exit(1)
		}
		if err := runStatus(ctx, s, os.Args[2]); err != nil {
			slog.Error("status failed", "error", err)
			os.Exit(1)
		}

	case "summarize":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: summarize <contract_id>")
			os.Exit(1)
		}
		if err := pipeline.RunSummarize(ctx, s, client, os.Args[2], cfg.SummaryClauseTokenBudget, cfg.LLMModel, "."); err != nil {
			slog.Error("summarize failed", "error", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runProcess(ctx context.Context, cfg config.Config, client llm.LLM, s store.Store, pdfPath string, requiresReview bool) error {
	contractID, err := pipeline.RunExtract(ctx, s, pdf.ExtractText, pdfPath, requiresReview)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	slog.Info("extracted", "contract_id", contractID)

	if err := pipeline.ExtractClauses(ctx, client, s, contractID); err != nil {
		return fmt.Errorf("extract-clauses: %w", err)
	}
	slog.Info("clauses extracted", "contract_id", contractID)

	if err := runAnalyze(ctx, cfg, client, s, contractID); err != nil {
		return fmt.Errorf("analyze-clauses: %w", err)
	}
	slog.Info("clauses analyzed", "contract_id", contractID)

	return runPostAnalysis(ctx, cfg, client, s, contractID, requiresReview)
}

func runPostAnalysis(ctx context.Context, cfg config.Config, client llm.LLM, s store.Store, contractID string, requiresReview bool) error {
	contract, err := s.GetContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}

	if contract.Status == domain.StatusReviewPending {
		slog.Info("review in progress — run: go run . review <id>", "contract_id", contractID)
		return nil
	}
	if contract.Status == domain.StatusReviewComplete || contract.Status == domain.StatusDone {
		slog.Info("already complete", "contract_id", contractID, "status", contract.Status)
		return nil
	}

	if requiresReview {
		if err := s.UpdateContractStatus(ctx, contractID, domain.StatusReviewPending); err != nil {
			return fmt.Errorf("set review_pending: %w", err)
		}
		slog.Info("review required — run: go run . review <id>", "contract_id", contractID)
		return nil
	}

	return pipeline.RunSummarize(ctx, s, client, contractID, cfg.SummaryClauseTokenBudget, cfg.LLMModel, ".")
}

func runAnalyze(ctx context.Context, cfg config.Config, client llm.LLM, s store.Store, contractID string) error {
	ctxMgr := agent.NewContextManager(
		cfg.LLMModel,
		cfg.ContextWindow,
		cfg.CompactRatio,
		cfg.KeepRecent,
		client,
	)
	budget := agent.NewBudget(cfg.RunMaxTokens, cfg.RunMaxCostUSD, cfg.RunMaxSteps)
	if err := pipeline.AnalyzeClauses(
		ctx,
		client,
		s,
		contractID,
		defaultMaxSteps,
		ctxMgr,
		budget,
		cfg.AnalysisConcurrency,
	); err != nil {
		return fmt.Errorf("analyze clauses: %w", err)
	}
	return nil
}

func runStatus(ctx context.Context, s store.Store, contractID string) error {
	contract, err := s.GetContract(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}
	fmt.Printf("Contract: %s  status=%s\n", contract.ID, contract.Status)

	clauses, err := s.GetClauses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get clauses: %w", err)
	}
	fmt.Printf("Clauses: %d\n", len(clauses))

	var submitted, running, failed, pending int
	for _, c := range clauses {
		run, _, found, err := s.LoadAgentRun(ctx, c.ID)
		if err != nil {
			return fmt.Errorf("load agent run for clause %s: %w", c.ID, err)
		}
		if !found {
			pending++
			fmt.Printf("  clause %s  status=pending\n", c.ID)
			continue
		}
		switch run.Status {
		case "submitted":
			submitted++
		case "running":
			running++
		default:
			failed++
		}
		fmt.Printf("  clause %s  status=%s  steps=%d  tokens=%d  cost=$%.4f\n",
			c.ID, run.Status, run.StepCount, run.UsedTokens, run.UsedCostUSD)
	}
	fmt.Printf("\nSummary: submitted=%d running=%d failed=%d pending=%d\n",
		submitted, running, failed, pending)
	return nil
}

func runAnalyzeClause(ctx context.Context, cfg config.Config, client llm.LLM, s store.Store, contractID, clauseID string) error {
	clauses, err := s.GetClauses(ctx, contractID)
	if err != nil {
		return fmt.Errorf("get clauses: %w", err)
	}

	var target *domain.Clause
	for i := range clauses {
		if clauses[i].ID == clauseID {
			target = &clauses[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("clause %q not found", clauseID)
	}

	reg := tool.NewRegistry(
		tool.NewGetDefinition(s, contractID),
		tool.NewGetContractSection(s, contractID),
		tool.NewSearchClauseLibrary(s, contractID),
		tool.NewLookupStandardClause(s, contractID),
	)

	ctxMgr := agent.NewContextManager(
		cfg.LLMModel,
		cfg.ContextWindow,
		cfg.CompactRatio,
		cfg.KeepRecent,
		client,
	)
	a := agent.NewWithStore(client, reg, defaultMaxSteps, ctxMgr, nil, s)
	result, err := a.Run(ctx, agent.AnalyzeClauseTask{
		ContractID: target.ContractID,
		ClauseID:   target.ID,
		ClauseText: target.Text,
	})
	if err != nil {
		return fmt.Errorf("agent run: %w", err)
	}

	fmt.Printf("Stop:  %s\n", result.Stop)
	fmt.Printf("Steps: %d\n", result.Steps)
	if result.Stop == "submitted" {
		fmt.Printf("Risk:  %s\n", *result.Finding.RiskLevel)
		fmt.Printf("Explanation: %s\n", result.Finding.Explanation)
		if result.Finding.AmbiguousLanguage != "" {
			fmt.Printf("Ambiguous: %s\n", result.Finding.AmbiguousLanguage)
		}
		fmt.Printf("Recommendations: %s\n", result.Finding.Recommendations)
	}
	return nil
}
