package prompts_test

import (
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/prompts"
)

func TestClauseAnalysisAgentPrompt_Render_ContainsRequiredPhrases(t *testing.T) {
	p := prompts.ClauseAnalysisAgentPrompt{
		ClauseText: "Section 7: Liability. Neither party shall be liable for indirect damages.",
		ContractID: "contract-abc",
	}
	rendered := p.Render()

	if rendered == "" {
		t.Fatal("Render() returned empty string")
	}

	required := []string{
		"contract-risk analyst",
		"submit_finding",
		"get_contract_section",
		"get_definition",
		"lookup_standard_clause",
		"risk_level",
		"high",
		"medium",
		"low",
		"Section 7: Liability",
	}
	for _, phrase := range required {
		if !strings.Contains(rendered, phrase) {
			t.Errorf("Render() missing required phrase: %q", phrase)
		}
	}
}

func TestClauseAnalysisAgentPrompt_Render_IncludesContractID(t *testing.T) {
	p := prompts.ClauseAnalysisAgentPrompt{
		ClauseText: "Payment is due within 30 days.",
		ContractID: "contract-xyz",
	}
	rendered := p.Render()
	if !strings.Contains(rendered, "contract-xyz") {
		t.Errorf("Render() does not contain contractID %q", "contract-xyz")
	}
}
