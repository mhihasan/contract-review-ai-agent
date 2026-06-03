package llm

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go/v3"
)

func Hello(ctx context.Context, client *openai.Client, model string) (string, error) {
	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Say hello and tell me one interesting fact about Go programming language. Be brief."),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
