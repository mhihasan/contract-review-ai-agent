package config_test

import (
	"os"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/config"
)

func TestLoad_HappyPath(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.OpenAIAPIKey != "sk-test" {
		t.Errorf("expected OpenAIAPIKey=sk-test, got %q", cfg.OpenAIAPIKey)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("expected DatabaseURL=postgres://localhost/test, got %q", cfg.DatabaseURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel=info, got %q", cfg.LogLevel)
	}
}

func TestLoad_MissingOpenAIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing OPENAI_API_KEY, got nil")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("DATABASE_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}
}

func TestLoad_MissingBothRequired(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("DATABASE_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing both required env vars, got nil")
	}
}

func TestLoad_CustomLogLevel(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel=debug, got %q", cfg.LogLevel)
	}
}

func TestLoad_LLMFields(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_MODEL", "gpt-4o-mini")
	t.Setenv("LLM_TEMPERATURE", "0.5")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLMProvider != "openai" {
		t.Errorf("LLMProvider = %q, want %q", cfg.LLMProvider, "openai")
	}
	if cfg.LLMModel != "gpt-4o-mini" {
		t.Errorf("LLMModel = %q, want %q", cfg.LLMModel, "gpt-4o-mini")
	}
	if cfg.LLMTemperature != 0.5 {
		t.Errorf("LLMTemperature = %v, want 0.5", cfg.LLMTemperature)
	}
}

func TestLoad_contextDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	os.Unsetenv("AGENT_CONTEXT_WINDOW") //nolint:errcheck
	os.Unsetenv("AGENT_COMPACT_RATIO")  //nolint:errcheck
	os.Unsetenv("AGENT_KEEP_RECENT")    //nolint:errcheck

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ContextWindow != 100000 {
		t.Errorf("ContextWindow default = %d, want 100000", cfg.ContextWindow)
	}
	if cfg.CompactRatio != 0.8 {
		t.Errorf("CompactRatio default = %f, want 0.8", cfg.CompactRatio)
	}
	if cfg.KeepRecent != 4 {
		t.Errorf("KeepRecent default = %d, want 4", cfg.KeepRecent)
	}
}

func TestLoad_contextFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("AGENT_CONTEXT_WINDOW", "50000")
	t.Setenv("AGENT_COMPACT_RATIO", "0.7")
	t.Setenv("AGENT_KEEP_RECENT", "6")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ContextWindow != 50000 {
		t.Errorf("ContextWindow = %d, want 50000", cfg.ContextWindow)
	}
	if cfg.CompactRatio != 0.7 {
		t.Errorf("CompactRatio = %f, want 0.7", cfg.CompactRatio)
	}
	if cfg.KeepRecent != 6 {
		t.Errorf("KeepRecent = %d, want 6", cfg.KeepRecent)
	}
}

func TestConfig_BudgetEnvVars_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("OPENAI_API_KEY", "test-key")
	// Unset budget vars to confirm defaults.
	for _, key := range []string{
		"AGENT_MAX_TOKENS", "AGENT_MAX_COST_USD",
		"RUN_MAX_TOKENS", "RUN_MAX_COST_USD", "RUN_MAX_STEPS",
	} {
		t.Setenv(key, "")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AgentMaxTokens != 50000 {
		t.Errorf("AgentMaxTokens = %d, want 50000", cfg.AgentMaxTokens)
	}
	if cfg.AgentMaxCostUSD != 0.50 {
		t.Errorf("AgentMaxCostUSD = %v, want 0.50", cfg.AgentMaxCostUSD)
	}
	if cfg.RunMaxTokens != 2000000 {
		t.Errorf("RunMaxTokens = %d, want 2000000", cfg.RunMaxTokens)
	}
	if cfg.RunMaxCostUSD != 5.00 {
		t.Errorf("RunMaxCostUSD = %v, want 5.00", cfg.RunMaxCostUSD)
	}
	if cfg.RunMaxSteps != 10000 {
		t.Errorf("RunMaxSteps = %d, want 10000", cfg.RunMaxSteps)
	}
}

func TestConfig_BudgetEnvVars_Override(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("AGENT_MAX_TOKENS", "99999")
	t.Setenv("AGENT_MAX_COST_USD", "1.23")
	t.Setenv("RUN_MAX_TOKENS", "888888")
	t.Setenv("RUN_MAX_COST_USD", "9.99")
	t.Setenv("RUN_MAX_STEPS", "42")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AgentMaxTokens != 99999 {
		t.Errorf("AgentMaxTokens = %d, want 99999", cfg.AgentMaxTokens)
	}
	if cfg.AgentMaxCostUSD != 1.23 {
		t.Errorf("AgentMaxCostUSD = %v, want 1.23", cfg.AgentMaxCostUSD)
	}
	if cfg.RunMaxTokens != 888888 {
		t.Errorf("RunMaxTokens = %d, want 888888", cfg.RunMaxTokens)
	}
	if cfg.RunMaxCostUSD != 9.99 {
		t.Errorf("RunMaxCostUSD = %v, want 9.99", cfg.RunMaxCostUSD)
	}
	if cfg.RunMaxSteps != 42 {
		t.Errorf("RunMaxSteps = %d, want 42", cfg.RunMaxSteps)
	}
}

func TestAnalysisConcurrencyDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("ANALYSIS_CONCURRENCY", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AnalysisConcurrency != 5 {
		t.Errorf("want 5, got %d", cfg.AnalysisConcurrency)
	}
}

func TestAnalysisConcurrencyFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("ANALYSIS_CONCURRENCY", "10")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AnalysisConcurrency != 10 {
		t.Errorf("want 10, got %d", cfg.AnalysisConcurrency)
	}
}

func TestRetryDefaults(t *testing.T) {
	os.Unsetenv("LLM_MAX_RETRIES")   //nolint:errcheck
	os.Unsetenv("LLM_RETRY_BASE_MS") //nolint:errcheck
	// Set required env vars so Load() succeeds
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "test-key")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMMaxRetries != 3 {
		t.Errorf("expected LLMMaxRetries=3, got %d", cfg.LLMMaxRetries)
	}
	if cfg.LLMRetryBaseMS != 500 {
		t.Errorf("expected LLMRetryBaseMS=500, got %d", cfg.LLMRetryBaseMS)
	}
}
