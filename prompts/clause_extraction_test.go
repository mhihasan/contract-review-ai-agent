package prompts_test

import (
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/prompts"
)

func TestClauseExtractionPrompt_Render_ContainsContractText(t *testing.T) {
	p := prompts.ClauseExtractionPrompt{ContractText: "This is the contract body."}
	got := p.Render()
	if !strings.Contains(got, "This is the contract body.") {
		t.Errorf("Render() does not contain the contract text; got:\n%s", got)
	}
}

func TestClauseExtractionPrompt_Render_DemandsJSON(t *testing.T) {
	p := prompts.ClauseExtractionPrompt{ContractText: "anything"}
	got := p.Render()
	if !strings.Contains(got, "JSON") {
		t.Errorf("Render() does not mention JSON; got:\n%s", got)
	}
}
