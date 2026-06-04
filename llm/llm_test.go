package llm_test

import (
	"encoding/json"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/llm"
)

func TestTypes_Compile(_ *testing.T) {
	_ = llm.Message{
		Role:    llm.RoleUser,
		Content: "hello",
	}
	_ = llm.ToolSchema{
		Name:        "my_tool",
		Description: "does something",
		Parameters:  json.RawMessage(`{}`),
	}
	_ = llm.ToolCall{
		ID:   "call_abc",
		Name: "my_tool",
		Args: json.RawMessage(`{"key":"val"}`),
	}
	_ = llm.CompletionRequest{
		Messages:    []llm.Message{{Role: llm.RoleSystem, Content: "sys"}},
		MaxTokens:   512,
		Temperature: 0.2,
	}
	_ = llm.CompletionResponse{
		Content:      "result",
		StopReason:   "end_turn",
		InputTokens:  10,
		OutputTokens: 20,
		Provider:     "openai",
	}
}
