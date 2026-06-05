package llm

import (
	"fmt"
	"time"

	"github.com/mhihasan/contract-review-ai-agent/config"
)

func New(cfg config.Config) (LLM, error) {
	var inner LLM
	switch cfg.LLMProvider {
	case "openai":
		inner = NewOpenAI(cfg.OpenAIAPIKey, cfg.LLMModel)
	case "anthropic":
		inner = NewAnthropic(cfg.AnthropicAPIKey, cfg.LLMModel)
	default:
		return nil, fmt.Errorf("unknown LLM provider: %q", cfg.LLMProvider)
	}
	maxRetries := cfg.LLMMaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	baseDelay := time.Duration(cfg.LLMRetryBaseMS) * time.Millisecond
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}
	return NewRetryingLLM(inner, maxRetries, baseDelay), nil
}
