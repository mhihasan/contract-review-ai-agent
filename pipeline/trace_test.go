package pipeline_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/pipeline"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

func TestRenderTrace_basicOutput(t *testing.T) {
	run := store.AgentRun{
		ID: "run-abc", ClauseID: "clause-xyz",
		Status: "submitted", StepCount: 2,
		UsedTokens: 3120, UsedCostUSD: 0.0094,
		StartedAt: time.Now(),
	}

	msgs0, _ := json.Marshal([]llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{
			{Name: "get_contract_section", Args: json.RawMessage(`{"reference":"7.2"}`)},
		}},
		{Role: llm.RoleTool, Content: "Section 7.2: The Provider shall not be liable…"},
	})
	msgs1, _ := json.Marshal([]llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{
			{Name: "submit_finding", Args: json.RawMessage(`{"risk_level":"high"}`)},
		}},
	})

	var buf bytes.Buffer
	pipeline.RenderTrace(&buf, run, []store.AgentStep{
		{StepIndex: 0, Messages: msgs0},
		{StepIndex: 1, Messages: msgs1},
	})
	out := buf.String()

	for _, want := range []string{
		"agent_run=run-abc", "get_contract_section", "Section 7.2", "submit_finding",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in trace output, got:\n%s", want, out)
		}
	}
}
