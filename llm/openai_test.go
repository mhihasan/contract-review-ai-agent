package llm_test

import (
	"context"
	"os"
	"testing"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/mhihasan/contract-review-ai-agent/llm"
)

func TestHello_ReturnsNonEmptyResponse(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set — skipping live LLM test")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))
	ctx := context.Background()

	text, err := llm.Hello(ctx, &client, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("Hello returned error: %v", err)
	}
	if text == "" {
		t.Fatal("Hello returned an empty string")
	}
	t.Logf("LLM response: %s", text)
}
