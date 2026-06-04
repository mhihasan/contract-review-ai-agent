package llm

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAI struct {
	client *openai.Client
	model  string
}

var _ LLM = (*OpenAI)(nil)

func NewOpenAI(apiKey, model string) *OpenAI {
	c := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAI{client: &c, model: model}
}

func (o *OpenAI) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	params := openai.ChatCompletionNewParams{
		Model:    o.model,
		Messages: toOpenAIMessages(req.Messages),
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature != 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if len(req.Tools) > 0 {
		params.Tools = toOpenAITools(req.Tools)
	}

	raw, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openai completion: %w", err)
	}
	if len(raw.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("openai returned no choices")
	}

	choice := raw.Choices[0]
	resp := CompletionResponse{
		Content:      choice.Message.Content,
		StopReason:   mapOpenAIStopReason(string(choice.FinishReason)),
		InputTokens:  int(raw.Usage.PromptTokens),
		OutputTokens: int(raw.Usage.CompletionTokens),
		Model:        raw.Model,
		Provider:     "openai",
	}
	for _, tc := range choice.Message.ToolCalls {
		resp.ToolCalls = append(resp.ToolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: []byte(tc.Function.Arguments),
		})
	}
	return resp, nil
}

func toOpenAIMessages(msgs []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			out = append(out, openai.SystemMessage(m.Content))
		case RoleUser:
			out = append(out, openai.UserMessage(m.Content))
		case RoleAssistant:
			out = append(out, openai.AssistantMessage(m.Content))
		case RoleTool:
			out = append(out, openai.ToolMessage(m.Content, m.ToolCallID))
		}
	}
	return out
}

func toOpenAITools(tools []ToolSchema) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, len(tools))
	for i, t := range tools {
		var params shared.FunctionParameters
		if len(t.Parameters) > 0 {
			_ = json.Unmarshal(t.Parameters, &params)
		}
		out[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  params,
		})
	}
	return out
}

func mapOpenAIStopReason(r string) string {
	switch r {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	default:
		return r
	}
}
