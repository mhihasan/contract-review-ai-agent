package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	OpenAIAPIKey string
	DatabaseURL  string
	LogLevel     string
}

func Load() (Config, error) {
	var missing []string

	key, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok || key == "" {
		missing = append(missing, "OPENAI_API_KEY")
	}

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok || dbURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return Config{
		OpenAIAPIKey: key,
		DatabaseURL:  dbURL,
		LogLevel:     logLevel,
	}, nil
}
