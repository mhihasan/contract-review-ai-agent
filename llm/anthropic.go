package llm

import (
	"context"
	"encoding/json"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Anthropic struct {
	client *anthropic.Client
	model  string
}

var _ LLM = (*Anthropic)(nil)

func NewAnthropic(apiKey, model string) *Anthropic {
	c := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Anthropic{client: &c, model: model}
}

func (a *Anthropic) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	system, msgs := splitSystem(req.Messages)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: int64(req.MaxTokens),
		Messages:  msgs,
	}
	if system != "" {
		params.System = []anthropic.TextBlockParam{{Text: system}}
	}
	if len(req.Tools) > 0 {
		params.Tools = toAnthropicTools(req.Tools)
	}

	raw, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("anthropic completion: %w", err)
	}

	resp := CompletionResponse{
		StopReason:   string(raw.StopReason),
		InputTokens:  int(raw.Usage.InputTokens),
		OutputTokens: int(raw.Usage.OutputTokens),
		Model:        string(raw.Model),
		Provider:     "anthropic",
	}
	for _, block := range raw.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			resp.Content = b.Text
		case anthropic.ToolUseBlock:
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:   b.ID,
				Name: b.Name,
				Args: json.RawMessage(b.Input),
			})
		}
	}
	return resp, nil
}

func splitSystem(msgs []Message) (string, []anthropic.MessageParam) {
	var system string
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			system = m.Content
		case RoleUser:
			if m.ToolCallID != "" {
				out = append(out, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false),
				))
			} else {
				out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
			}
		case RoleAssistant:
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		case RoleTool:
		}
	}
	return system, out
}

func toAnthropicTools(tools []ToolSchema) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		var schema interface{}
		_ = json.Unmarshal(t.Parameters, &schema)
		out[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{Properties: schema},
			},
		}
	}
	return out
}
