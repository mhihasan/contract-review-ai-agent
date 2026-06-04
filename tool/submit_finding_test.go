package tool_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/tool"
)

func TestSubmitFinding_Name(t *testing.T) {
	sf := tool.NewSubmitFinding(func(_ tool.Finding) {})
	if sf.Name() != "submit_finding" {
		t.Errorf("Name() = %q, want %q", sf.Name(), "submit_finding")
	}
}

func TestSubmitFinding_Schema_ValidJSON(t *testing.T) {
	sf := tool.NewSubmitFinding(func(_ tool.Finding) {})
	if !json.Valid(sf.Schema()) {
		t.Errorf("Schema() is not valid JSON: %s", sf.Schema())
	}
}

func TestSubmitFinding_ValidInput_CallsCallback(t *testing.T) {
	var received tool.Finding
	sf := tool.NewSubmitFinding(func(f tool.Finding) { received = f })

	args := json.RawMessage(`{
		"risk_level": "high",
		"explanation": "The liability cap is missing entirely, creating unlimited exposure.",
		"ambiguous_language": "The phrase 'reasonable efforts' is undefined.",
		"recommendations": "Add a mutual liability cap tied to fees paid in the prior 12 months."
	}`)

	result, err := sf.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty success result")
	}
	if received.RiskLevel != "high" {
		t.Errorf("RiskLevel = %q, want %q", received.RiskLevel, "high")
	}
	if received.Explanation == "" {
		t.Error("Explanation must not be empty")
	}
}

func TestSubmitFinding_InvalidRiskLevel_ReturnsErrorString(t *testing.T) {
	sf := tool.NewSubmitFinding(func(_ tool.Finding) {})
	args := json.RawMessage(`{
		"risk_level": "critical",
		"explanation": "something",
		"ambiguous_language": "",
		"recommendations": "fix it"
	}`)

	result, err := sf.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute must not return Go error for bad risk_level, got: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "risk_level") {
		t.Errorf("expected 'risk_level' in error string, got: %s", result)
	}
}

func TestSubmitFinding_MissingExplanation_ReturnsErrorString(t *testing.T) {
	sf := tool.NewSubmitFinding(func(_ tool.Finding) {})
	args := json.RawMessage(`{
		"risk_level": "low",
		"explanation": "",
		"ambiguous_language": "",
		"recommendations": "none"
	}`)

	result, err := sf.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute must not return Go error for missing explanation, got: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "explanation") {
		t.Errorf("expected 'explanation' in error string, got: %s", result)
	}
}

func TestSubmitFinding_MissingRecommendations_ReturnsErrorString(t *testing.T) {
	sf := tool.NewSubmitFinding(func(_ tool.Finding) {})
	args := json.RawMessage(`{
		"risk_level": "medium",
		"explanation": "something notable",
		"ambiguous_language": "",
		"recommendations": ""
	}`)

	result, err := sf.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute must not return Go error for missing recommendations, got: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "recommendations") {
		t.Errorf("expected 'recommendations' in error string, got: %s", result)
	}
}

func TestSubmitFinding_BadArgs_ReturnsErrorString(t *testing.T) {
	sf := tool.NewSubmitFinding(func(_ tool.Finding) {})
	result, err := sf.Execute(context.Background(), json.RawMessage(`not-json`))
	if err != nil {
		t.Fatalf("Execute must not return Go error for bad JSON, got: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty error string for bad args")
	}
}

func TestSubmitFinding_AllThreeRiskLevels(t *testing.T) {
	for _, level := range []string{"high", "medium", "low"} {
		sf := tool.NewSubmitFinding(func(_ tool.Finding) {})
		args := json.RawMessage(`{"risk_level":"` + level + `","explanation":"test","ambiguous_language":"none","recommendations":"none"}`)
		_, err := sf.Execute(context.Background(), args)
		if err != nil {
			t.Errorf("risk_level %q: unexpected error: %v", level, err)
		}
	}
}
