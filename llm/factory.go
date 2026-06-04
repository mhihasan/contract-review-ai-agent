package llm

import (
	"fmt"

	"github.com/mhihasan/contract-review-ai-agent/config"
)

func New(cfg config.Config) (LLM, error) {
	switch cfg.LLMProvider {
	case "openai":
		return NewOpenAI(cfg.OpenAIAPIKey, cfg.LLMModel), nil
	case "anthropic":
		return NewAnthropic(cfg.AnthropicAPIKey, cfg.LLMModel), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %q", cfg.LLMProvider)
	}
}
