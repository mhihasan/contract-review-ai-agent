package cost_test

import (
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/cost"
)

func TestEstimate_KnownModels(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		in       int
		out      int
		want     float64
	}{
		{"openai", "gpt-4o", 1_000_000, 0, 2.50},
		{"openai", "gpt-4o", 0, 1_000_000, 10.00},
		{"openai", "gpt-4o", 500_000, 500_000, 6.25},
		{"openai", "gpt-4o-mini", 1_000_000, 0, 0.15},
		{"openai", "gpt-4o-mini", 0, 1_000_000, 0.60},
		{"anthropic", "claude-sonnet-4-6", 1_000_000, 0, 3.00},
		{"anthropic", "claude-sonnet-4-6", 0, 1_000_000, 15.00},
		{"anthropic", "claude-haiku-4-5-20251001", 1_000_000, 0, 0.80},
		{"anthropic", "claude-haiku-4-5-20251001", 0, 1_000_000, 4.00},
	}

	for _, tt := range tests {
		got := cost.Estimate(tt.provider, tt.model, tt.in, tt.out)
		if got != tt.want {
			t.Errorf("Estimate(%q, %q, %d, %d) = %v, want %v",
				tt.provider, tt.model, tt.in, tt.out, got, tt.want)
		}
	}
}

func TestEstimate_UnknownModel_ReturnsZero(t *testing.T) {
	got := cost.Estimate("openai", "gpt-99-ultra", 100_000, 50_000)
	if got != 0 {
		t.Errorf("expected 0 for unknown model, got %v", got)
	}
}
