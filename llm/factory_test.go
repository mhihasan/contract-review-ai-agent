package llm_test

import (
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/config"
	"github.com/mhihasan/contract-review-ai-agent/llm"
)

func TestNew_OpenAI(t *testing.T) {
	cfg := config.Config{
		LLMProvider:  "openai",
		LLMModel:     "gpt-4o-mini",
		OpenAIAPIKey: "sk-test",
	}
	client, err := llm.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestNew_Anthropic(t *testing.T) {
	cfg := config.Config{
		LLMProvider:     "anthropic",
		LLMModel:        "claude-haiku-4-5-20251001",
		AnthropicAPIKey: "sk-ant-test",
	}
	client, err := llm.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestNew_UnknownProvider(t *testing.T) {
	cfg := config.Config{LLMProvider: "groq"}
	_, err := llm.New(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
