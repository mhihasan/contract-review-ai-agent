package domain_test

import (
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/domain"
)

func TestContractStatus_String(t *testing.T) {
	if domain.StatusUploaded.String() != "uploaded" {
		t.Errorf("expected 'uploaded', got %q", domain.StatusUploaded.String())
	}
	if domain.StatusDone.String() != "done" {
		t.Errorf("expected 'done', got %q", domain.StatusDone.String())
	}
}

func TestRiskLevel_String(t *testing.T) {
	if domain.RiskHigh.String() != "high" {
		t.Errorf("expected 'high', got %q", domain.RiskHigh.String())
	}
}

func TestParseRiskLevel_Valid(t *testing.T) {
	cases := []string{"high", "medium", "low"}
	for _, c := range cases {
		r, err := domain.ParseRiskLevel(c)
		if err != nil {
			t.Errorf("ParseRiskLevel(%q) unexpected error: %v", c, err)
		}
		if r.String() != c {
			t.Errorf("ParseRiskLevel(%q) = %q, want %q", c, r.String(), c)
		}
	}
}

func TestParseRiskLevel_Invalid(t *testing.T) {
	_, err := domain.ParseRiskLevel("critical")
	if err == nil {
		t.Error("expected error for invalid risk level, got nil")
	}
}
