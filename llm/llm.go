package llm

import (
	"context"
	"encoding/json"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type ToolSchema struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type ToolCall struct {
	ID   string
	Name string
	Args json.RawMessage
}

type CompletionRequest struct {
	Messages    []Message
	Tools       []ToolSchema
	MaxTokens   int
	Temperature float64
}

type CompletionResponse struct {
	Content      string
	ToolCalls    []ToolCall
	StopReason   string
	InputTokens  int
	OutputTokens int
	Model        string
	Provider     string
}

type LLM interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
