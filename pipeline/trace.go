package pipeline

import (
	"encoding/json"
	"io"

	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

func RenderTrace(w io.Writer, run store.AgentRun, steps []store.AgentStep) {
	ew := &errWriter{w: w}
	ew.printf("agent_run=%s clause=%s stop=%s steps=%d tokens=%d cost=$%.4f\n",
		run.ID, run.ClauseID, run.Status, run.StepCount, run.UsedTokens, run.UsedCostUSD)

	for _, s := range steps {
		var msgs []llm.Message
		if err := json.Unmarshal(s.Messages, &msgs); err != nil {
			ew.printf("  step %d  [parse error: %v]\n", s.StepIndex, err)
			continue
		}
		for _, m := range msgs {
			switch m.Role {
			case llm.RoleAssistant:
				for _, tc := range m.ToolCalls {
					ew.printf("  step %d  assistant → tool_call %s%s\n",
						s.StepIndex, tc.Name, traceArgs(tc.Args, 80))
				}
				if len(m.ToolCalls) == 0 && m.Content != "" {
					ew.printf("  step %d  assistant → text %q\n",
						s.StepIndex, traceStr(m.Content, 120))
				}
			case llm.RoleTool:
				ew.printf("  step %d  tool       ← %q\n",
					s.StepIndex, traceStr(m.Content, 120))
			case llm.RoleUser, llm.RoleSystem:
				// user/system messages are not rendered in the trace
			}
		}
	}
}

func traceStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func traceArgs(raw json.RawMessage, n int) string {
	if len(raw) == 0 {
		return ""
	}
	r := []rune(string(raw))
	if len(r) <= n {
		return string(raw)
	}
	return string(r[:n]) + "…"
}
