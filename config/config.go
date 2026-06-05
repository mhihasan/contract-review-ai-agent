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
	ContextWindow   int
	CompactRatio    float64
	KeepRecent      int
	AgentMaxTokens  int
	AgentMaxCostUSD float64
	RunMaxTokens    int
	RunMaxCostUSD   float64
	RunMaxSteps     int
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

	contextWindow := 100000
	if s := os.Getenv("AGENT_CONTEXT_WINDOW"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			contextWindow = v
		}
	}

	compactRatio := 0.8
	if s := os.Getenv("AGENT_COMPACT_RATIO"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			compactRatio = v
		}
	}

	keepRecent := 4
	if s := os.Getenv("AGENT_KEEP_RECENT"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			keepRecent = v
		}
	}

	agentMaxTokens := 50000
	if s := os.Getenv("AGENT_MAX_TOKENS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			agentMaxTokens = v
		}
	}

	agentMaxCostUSD := 0.50
	if s := os.Getenv("AGENT_MAX_COST_USD"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			agentMaxCostUSD = v
		}
	}

	runMaxTokens := 500000
	if s := os.Getenv("RUN_MAX_TOKENS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			runMaxTokens = v
		}
	}

	runMaxCostUSD := 5.00
	if s := os.Getenv("RUN_MAX_COST_USD"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			runMaxCostUSD = v
		}
	}

	runMaxSteps := 200
	if s := os.Getenv("RUN_MAX_STEPS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			runMaxSteps = v
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
		ContextWindow:   contextWindow,
		CompactRatio:    compactRatio,
		KeepRecent:      keepRecent,
		AgentMaxTokens:  agentMaxTokens,
		AgentMaxCostUSD: agentMaxCostUSD,
		RunMaxTokens:    runMaxTokens,
		RunMaxCostUSD:   runMaxCostUSD,
		RunMaxSteps:     runMaxSteps,
	}, nil
}
