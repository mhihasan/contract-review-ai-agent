package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/prompts"
	"github.com/mhihasan/contract-review-ai-agent/store"
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

type agentStore interface {
	LoadAgentRun(ctx context.Context, clauseID string) (store.AgentRun, []store.AgentStep, bool, error)
	StartAgentRun(ctx context.Context, id, clauseID, runID string) error
	AppendAgentStep(ctx context.Context, agentRunID string, stepIndex int, messagesJSON, usageJSON []byte) error
	FinishAgentRun(ctx context.Context, id, status string, stepCount, usedTokens int, usedCostUSD float64) error
	GetStoredFinding(ctx context.Context, clauseID string) (domain.ClauseAnalysis, error)
}

type Agent struct {
	llm      llm.LLM
	tools    *tool.Registry
	maxSteps int
	ctxMgr   *ContextManager
	budget   *Budget
	store    agentStore
}

func New(l llm.LLM, tools *tool.Registry, maxSteps int, ctxMgr *ContextManager) *Agent {
	return NewWithStore(l, tools, maxSteps, ctxMgr, nil, nil)
}

func NewWithBudget(l llm.LLM, tools *tool.Registry, maxSteps int, ctxMgr *ContextManager, budget *Budget) *Agent {
	return NewWithStore(l, tools, maxSteps, ctxMgr, budget, nil)
}

func NewWithStore(l llm.LLM, tools *tool.Registry, maxSteps int, ctxMgr *ContextManager, budget *Budget, s agentStore) *Agent {
	return &Agent{llm: l, tools: tools, maxSteps: maxSteps, ctxMgr: ctxMgr, budget: budget, store: s}
}

func (a *Agent) Run(ctx context.Context, task AnalyzeClauseTask) (Result, error) {
	msgs := buildInitialMessages(task)
	startStep := 0
	agentRunID := uuid.New().String()
	var usage Usage

	if a.store != nil {
		run, steps, found, err := a.store.LoadAgentRun(ctx, task.ClauseID)
		if err != nil {
			return Result{}, fmt.Errorf("load agent run: %w", err)
		}
		if found && run.Status == AgentRunStatusSubmitted {
			finding, err := a.store.GetStoredFinding(ctx, task.ClauseID)
			if err != nil {
				return Result{}, fmt.Errorf("get stored finding: %w", err)
			}
			return Result{Finding: finding, Steps: run.StepCount, Stop: "submitted", Usage: Usage{InputTokens: run.UsedTokens}}, nil
		}
		if found && run.Status == AgentRunStatusRunning {
			agentRunID = run.ID
			if len(steps) > 0 {
				last := steps[len(steps)-1]
				if err := json.Unmarshal(last.Messages, &msgs); err != nil {
					return Result{}, fmt.Errorf("unmarshal messages: %w", err)
				}
				startStep = last.StepIndex + 1
			}
			if a.budget != nil {
				a.budget.RestoreTokens(run.UsedTokens, run.UsedCostUSD)
			}
		} else if !found {
			if err := a.store.StartAgentRun(ctx, agentRunID, task.ClauseID, ""); err != nil {
				return Result{}, fmt.Errorf("start agent run: %w", err)
			}
		}
	}

	finalStop := "max_steps"
	var finalFinding domain.ClauseAnalysis
	stepsRun := 0

	for step := startStep; step < a.maxSteps; step++ {
		if ctx.Err() != nil {
			return Result{Steps: step - startStep, Stop: "cancelled", Usage: usage}, ctx.Err()
		}

		if a.ctxMgr != nil {
			var err error
			msgs, err = a.ctxMgr.Prepare(ctx, msgs)
			if err != nil {
				return Result{}, fmt.Errorf("context prepare at step %d: %w", step, err)
			}
		}

		if a.budget != nil && a.budget.Exceeded() {
			finalStop = "budget"
			stepsRun = step - startStep
			break
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

		if a.budget != nil {
			a.budget.Record(resp.Provider, resp.Model, resp.InputTokens, resp.OutputTokens)
		}

		msgs = append(msgs, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		submitted := false
		if len(resp.ToolCalls) == 0 {
			msgs = append(msgs, llm.Message{
				Role:    llm.RoleUser,
				Content: "Use a tool, or call submit_finding to finish.",
			})
		} else {
			for _, call := range resp.ToolCalls {
				if call.Name == "submit_finding" {
					finding, verr := parseAndValidateFinding(task, call.Args)
					if verr != nil {
						msgs = append(msgs, toolResultMessage(call.ID, "invalid: "+verr.Error()))
						continue
					}
					finalFinding = finding
					finalStop = "submitted"
					submitted = true
					break
				}
				result, err := a.tools.Dispatch(ctx, call)
				if err != nil {
					result = "tool error: " + err.Error()
				}
				msgs = append(msgs, toolResultMessage(call.ID, result))
			}
		}

		if a.store != nil {
			msgsJSON, _ := json.Marshal(msgs)
			usageJSON, _ := json.Marshal(map[string]int{"input": resp.InputTokens, "output": resp.OutputTokens})
			_ = a.store.AppendAgentStep(ctx, agentRunID, step, msgsJSON, usageJSON)
		}

		stepsRun = step - startStep + 1

		if submitted {
			if a.store != nil {
				snap := BudgetSnapshot{}
				if a.budget != nil {
					snap = a.budget.Snapshot()
				}
				_ = a.store.FinishAgentRun(ctx, agentRunID, "submitted", stepsRun, snap.UsedTokens, snap.UsedCostUSD)
			}
			return Result{Finding: finalFinding, Steps: stepsRun, Stop: "submitted", Usage: usage}, nil
		}
	}

	if finalStop == "max_steps" {
		stepsRun = a.maxSteps - startStep
	}

	if a.store != nil {
		snap := BudgetSnapshot{}
		if a.budget != nil {
			snap = a.budget.Snapshot()
		}
		_ = a.store.FinishAgentRun(ctx, agentRunID, finalStop, stepsRun, snap.UsedTokens, snap.UsedCostUSD)
	}
	return Result{Steps: stepsRun, Stop: finalStop, Usage: usage}, nil
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
