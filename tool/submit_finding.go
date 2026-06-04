package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

type Finding struct {
	RiskLevel         string
	Explanation       string
	AmbiguousLanguage string
	Recommendations   string
}

type SubmitFinding struct {
	OnFinding func(Finding)
}

var _ Tool = (*SubmitFinding)(nil)

func NewSubmitFinding(onFinding func(Finding)) *SubmitFinding {
	return &SubmitFinding{OnFinding: onFinding}
}

func (sf *SubmitFinding) Name() string { return "submit_finding" }
func (sf *SubmitFinding) Description() string {
	return "Submit the final structured finding for this clause. Call this when analysis is complete. risk_level must be 'high', 'medium', or 'low'."
}
func (sf *SubmitFinding) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"risk_level": {
				"type": "string",
				"enum": ["high", "medium", "low"],
				"description": "The assessed risk level for this clause"
			},
			"explanation": {
				"type": "string",
				"description": "Clear explanation of the risk and why it matters"
			},
			"ambiguous_language": {
				"type": "string",
				"description": "Any ambiguous or problematic language identified (empty string if none)"
			},
			"recommendations": {
				"type": "string",
				"description": "Specific recommendations for improving or accepting the clause"
			}
		},
		"required": ["risk_level", "explanation", "ambiguous_language", "recommendations"]
	}`)
}

func (sf *SubmitFinding) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var req struct {
		RiskLevel         string `json:"risk_level"`
		Explanation       string `json:"explanation"`
		AmbiguousLanguage string `json:"ambiguous_language"`
		Recommendations   string `json:"recommendations"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return fmt.Sprintf("invalid args: %v", err), nil
	}

	switch req.RiskLevel {
	case "high", "medium", "low":
	default:
		return fmt.Sprintf("invalid risk_level %q: must be 'high', 'medium', or 'low'", req.RiskLevel), nil
	}
	if req.Explanation == "" {
		return "invalid args: explanation is required", nil
	}
	if req.Recommendations == "" {
		return "invalid args: recommendations is required", nil
	}

	f := Finding{
		RiskLevel:         req.RiskLevel,
		Explanation:       req.Explanation,
		AmbiguousLanguage: req.AmbiguousLanguage,
		Recommendations:   req.Recommendations,
	}
	if sf.OnFinding != nil {
		sf.OnFinding(f)
	}
	return "finding submitted", nil
}
