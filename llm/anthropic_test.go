package llm_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/llm"
)

func TestAnthropic_ImplementsLLM(_ *testing.T) {
	var _ llm.LLM = &llm.Anthropic{}
}

func TestAnthropic_PlainCompletion_Integration(t *testing.T) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}

	client := llm.NewAnthropic(key, model)
	resp, err := client.Complete(context.Background(), llm.CompletionRequest{
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "Reply with the single word: pong"}},
		MaxTokens: 16,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.StopReason == "" {
		t.Error("StopReason must not be empty")
	}
	if resp.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "anthropic")
	}
	if resp.InputTokens == 0 {
		t.Error("InputTokens must be > 0")
	}
}

func TestAnthropic_ToolCall_Integration(t *testing.T) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}

	client := llm.NewAnthropic(key, model)
	schema := json.RawMessage(`{
		"type": "object",
		"properties": { "city": { "type": "string" } },
		"required": ["city"]
	}`)
	resp, err := client.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "What is the weather in Paris? Use the get_weather tool."},
		},
		Tools: []llm.ToolSchema{
			{Name: "get_weather", Description: "Get weather for a city", Parameters: schema},
		},
		MaxTokens: 64,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "tool_use")
	}
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected at least one ToolCall")
	}
	tc := resp.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.ID == "" {
		t.Error("ToolCall.ID must not be empty")
	}
	var args map[string]string
	if err := json.Unmarshal(tc.Args, &args); err != nil {
		t.Fatalf("unmarshal tool args: %v", err)
	}
	if args["city"] == "" {
		t.Error("expected 'city' in tool args")
	}
}
