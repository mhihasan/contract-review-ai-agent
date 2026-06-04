package tool_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/tool"
)

type stubTool struct {
	name   string
	result string
}

func (s *stubTool) Name() string            { return s.name }
func (s *stubTool) Description() string     { return "stub" }
func (s *stubTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (s *stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return s.result, nil
}

func TestRegistry_Schemas_ContainsAllTools(t *testing.T) {
	a := &stubTool{name: "tool_a"}
	b := &stubTool{name: "tool_b"}
	reg := tool.NewRegistry(a, b)

	schemas := reg.Schemas()
	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}
	names := map[string]bool{}
	for _, s := range schemas {
		names[s.Name] = true
	}
	if !names["tool_a"] || !names["tool_b"] {
		t.Errorf("schemas missing expected names: %v", names)
	}
}

func TestRegistry_Schemas_ValidJSON(t *testing.T) {
	reg := tool.NewRegistry(&stubTool{name: "x"})
	for _, s := range reg.Schemas() {
		if !json.Valid(s.Parameters) {
			t.Errorf("schema %q has invalid JSON Parameters: %s", s.Name, s.Parameters)
		}
	}
}

func TestRegistry_Dispatch_RoutesToCorrectTool(t *testing.T) {
	reg := tool.NewRegistry(
		&stubTool{name: "greet", result: "hello"},
		&stubTool{name: "farewell", result: "goodbye"},
	)
	ctx := context.Background()

	result, err := reg.Dispatch(ctx, llm.ToolCall{Name: "greet", Args: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if result != "hello" {
		t.Errorf("result = %q, want %q", result, "hello")
	}
}

func TestRegistry_Dispatch_UnknownTool_ReturnsErrorString(t *testing.T) {
	reg := tool.NewRegistry(&stubTool{name: "known"})
	ctx := context.Background()

	result, err := reg.Dispatch(ctx, llm.ToolCall{Name: "does_not_exist", Args: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("expected no Go error for unknown tool, got: %v", err)
	}
	if result == "" {
		t.Error("expected a non-empty error string for unknown tool")
	}
}

func TestRegistry_Dispatch_EmptyRegistry_UnknownTool(t *testing.T) {
	reg := tool.NewRegistry()
	ctx := context.Background()

	result, err := reg.Dispatch(ctx, llm.ToolCall{Name: "anything"})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty error string")
	}
}
