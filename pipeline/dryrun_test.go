package pipeline_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/pipeline"
)

func TestPrintDryRunPlan_containsAllStages(t *testing.T) {
	var buf bytes.Buffer
	pipeline.PrintDryRunPlan(&buf, pipeline.DryRunPlan{
		ContractID:       "abc123",
		InputChars:       4821,
		EstClauses:       12,
		Concurrency:      5,
		PerAgentMaxSteps: 12,
		BudgetCapUSD:     5.0,
		BudgetCapTokens:  2000000,
	})
	out := buf.String()

	for _, want := range []string{
		"DRY RUN", "abc123", "extract_clauses",
		"analyze", "summarize",
		"No LLM calls made", "No DB writes made",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in dry-run output, got:\n%s", want, out)
		}
	}
}
