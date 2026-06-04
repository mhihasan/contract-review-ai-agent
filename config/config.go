package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	OpenAIAPIKey    string
	AnthropicAPIKey string
	DatabaseURL     string
	LogLevel        string
	LLMProvider     string
	LLMModel        string
	LLMTemperature  float64
}

func Load() (Config, error) {
	var missing []string

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok || dbURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "openai"
	}

	var openAIKey, anthropicKey string
	switch provider {
	case "openai":
		openAIKey = os.Getenv("OPENAI_API_KEY")
		if openAIKey == "" {
			missing = append(missing, "OPENAI_API_KEY")
		}
	case "anthropic":
		anthropicKey = os.Getenv("ANTHROPIC_API_KEY")
		if anthropicKey == "" {
			missing = append(missing, "ANTHROPIC_API_KEY")
		}
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	temp := 0.2
	if s := os.Getenv("LLM_TEMPERATURE"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			temp = v
		}
	}

	return Config{
		OpenAIAPIKey:    openAIKey,
		AnthropicAPIKey: anthropicKey,
		DatabaseURL:     dbURL,
		LogLevel:        logLevel,
		LLMProvider:     provider,
		LLMModel:        model,
		LLMTemperature:  temp,
	}, nil
}
