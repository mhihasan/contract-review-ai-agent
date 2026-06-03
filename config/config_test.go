package config_test

import (
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
