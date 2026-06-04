package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/prompts"
	"github.com/mhihasan/contract-review-ai-agent/tool"
)

type AnalyzeClauseTask struct {
	ContractID string
	ClauseID   string
	ClauseText string
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type Result struct {
	Finding domain.ClauseAnalysis
	Steps   int
	Stop    string
	Usage   Usage
}

type Agent struct {
	llm      llm.LLM
	tools    *tool.Registry
	maxSteps int
}

func New(l llm.LLM, tools *tool.Registry, maxSteps int) *Agent {
	return &Agent{llm: l, tools: tools, maxSteps: maxSteps}
}

func (a *Agent) Run(ctx context.Context, task AnalyzeClauseTask) (Result, error) {
	msgs := buildInitialMessages(task)
	var usage Usage

	for step := 0; step < a.maxSteps; step++ {
		if ctx.Err() != nil {
			return Result{Steps: step, Stop: "cancelled", Usage: usage}, ctx.Err()
		}

		resp, err := a.llm.Complete(ctx, llm.CompletionRequest{
			Messages:    msgs,
			Tools:       a.tools.Schemas(),
			MaxTokens:   1024,
			Temperature: 0.2,
		})
		if err != nil {
			return Result{}, fmt.Errorf("llm complete at step %d: %w", step, err)
		}
		usage.InputTokens += resp.InputTokens
		usage.OutputTokens += resp.OutputTokens

		msgs = append(msgs, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			msgs = append(msgs, llm.Message{
				Role:    llm.RoleUser,
				Content: "Use a tool, or call submit_finding to finish.",
			})
			continue
		}

		for _, call := range resp.ToolCalls {
			if call.Name == "submit_finding" {
				finding, verr := parseAndValidateFinding(task, call.Args)
				if verr != nil {
					msgs = append(msgs, toolResultMessage(call.ID, "invalid: "+verr.Error()))
					continue
				}
				return Result{Finding: finding, Steps: step + 1, Stop: "submitted", Usage: usage}, nil
			}
			result, err := a.tools.Dispatch(ctx, call)
			if err != nil {
				result = "tool error: " + err.Error()
			}
			msgs = append(msgs, toolResultMessage(call.ID, result))
		}
	}

	return Result{Steps: a.maxSteps, Stop: "max_steps", Usage: usage}, nil
}

func buildInitialMessages(task AnalyzeClauseTask) []llm.Message {
	p := prompts.ClauseAnalysisAgentPrompt{
		ClauseText: task.ClauseText,
		ContractID: task.ContractID,
	}
	return []llm.Message{
		{Role: llm.RoleUser, Content: p.Render()},
	}
}

func toolResultMessage(callID, content string) llm.Message {
	return llm.Message{
		Role:       llm.RoleTool,
		Content:    content,
		ToolCallID: callID,
	}
}

type submitFindingArgs struct {
	RiskLevel         string `json:"risk_level"`
	Explanation       string `json:"explanation"`
	AmbiguousLanguage string `json:"ambiguous_language"`
	Recommendations   string `json:"recommendations"`
}

func parseAndValidateFinding(task AnalyzeClauseTask, raw json.RawMessage) (domain.ClauseAnalysis, error) {
	var args submitFindingArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return domain.ClauseAnalysis{}, fmt.Errorf("unmarshal finding args: %w", err)
	}

	risk, err := domain.ParseRiskLevel(args.RiskLevel)
	if err != nil {
		return domain.ClauseAnalysis{}, fmt.Errorf("risk_level: %w", err)
	}
	if args.Explanation == "" {
		return domain.ClauseAnalysis{}, fmt.Errorf("explanation is required")
	}
	if args.Recommendations == "" {
		return domain.ClauseAnalysis{}, fmt.Errorf("recommendations is required")
	}

	return domain.ClauseAnalysis{
		ClauseID:          task.ClauseID,
		RiskLevel:         &risk,
		Explanation:       args.Explanation,
		AmbiguousLanguage: args.AmbiguousLanguage,
		Recommendations:   args.Recommendations,
		Status:            "analyzed",
	}, nil
}
