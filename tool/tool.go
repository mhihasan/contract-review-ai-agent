package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mhihasan/contract-review-ai-agent/llm"
)

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry(ts ...Tool) *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	for _, t := range ts {
		r.tools[t.Name()] = t
	}
	return r
}

func (r *Registry) Schemas() []llm.ToolSchema {
	out := make([]llm.ToolSchema, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, llm.ToolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		})
	}
	return out
}

func (r *Registry) SubmitFindingSchema() []llm.ToolSchema {
	t, ok := r.tools["submit_finding"]
	if !ok {
		return r.Schemas()
	}
	return []llm.ToolSchema{{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Schema(),
	}}
}

func (r *Registry) Dispatch(ctx context.Context, call llm.ToolCall) (string, error) {
	t, ok := r.tools[call.Name]
	if !ok {
		return fmt.Sprintf("unknown tool %q; available tools: %v", call.Name, r.toolNames()), nil
	}
	return t.Execute(ctx, call.Args)
}

func (r *Registry) toolNames() []string {
	names := make([]string, 0, len(r.tools))
	for k := range r.tools {
		names = append(names, k)
	}
	return names
}
