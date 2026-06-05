package pipeline

import (
	"fmt"
	"io"
)

type DryRunPlan struct {
	ContractID       string
	InputChars       int
	EstClauses       int
	Concurrency      int
	PerAgentMaxSteps int
	BudgetCapUSD     float64
	BudgetCapTokens  int
}

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) println(s string) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, s)
}

func PrintDryRunPlan(w io.Writer, p DryRunPlan) {
	ew := &errWriter{w: w}
	ew.printf("[DRY RUN] contract_id=%s\n", p.ContractID)
	ew.printf("  stage=extract_clauses  input=%d chars  expected_output=~%d clauses\n",
		p.InputChars, p.EstClauses)
	ew.printf("  stage=analyze          clauses=%d  concurrency=%d  agents=%d  per_agent_max_steps=%d\n",
		p.EstClauses, p.Concurrency, p.EstClauses, p.PerAgentMaxSteps)
	ew.printf("                         est_calls≈%d-%d  budget_cap=$%.2f / %d tokens\n",
		p.EstClauses, p.EstClauses*4, p.BudgetCapUSD, p.BudgetCapTokens)
	ew.printf("  stage=summarize        est_input_tokens≈%d\n", p.EstClauses*300)
	ew.println("  No LLM calls made. No DB writes made.")
}
