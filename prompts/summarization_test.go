package prompts_test

import (
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/prompts"
)

func TestSummarizationPrompt_ContainsSections(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:       "service-agreement.pdf",
		ReviewingParty: "client",
		GoverningLaw:   "New York",
		ContractType:   "MSA",
		RiskCounts:     prompts.RiskCounts{High: 3, Medium: 5, Low: 2},
		ReviewCounts:   prompts.ReviewCounts{Approved: 7, Rejected: 3, Overrides: 1},
		ClauseInputs: []prompts.ClauseInput{
			{
				SequenceNumber:    1,
				Gist:              "Unlimited liability for contractor.",
				RiskLevel:         "high",
				Decision:          "rejected",
				Annotation:        "Unacceptable — must cap at contract value.",
				Recommendations:   "Add: 'Vendor liability shall not exceed the total fees paid in the preceding 12 months.'",
				AmbiguousLanguage: "'unlimited' is undefined",
			},
		},
	}

	out := p.Render()

	required := []string{
		"Executive Summary",
		"Priority Issues",
		"Signing Recommendation",
		"Clause-by-Clause",
		"High: 3",
		"Medium: 5",
		"Low: 2",
		"client",
		"New York",
		"MSA",
		"Unacceptable — must cap at contract value.",
		"Vendor liability shall not exceed",
		"unlimited",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("Render() missing %q", want)
		}
	}
}

func TestSummarizationPrompt_RejectedClauseHasAnnotation(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:       "nda.pdf",
		ReviewingParty: "vendor",
		RiskCounts:     prompts.RiskCounts{High: 1},
		ReviewCounts:   prompts.ReviewCounts{Rejected: 1},
		ClauseInputs: []prompts.ClauseInput{
			{
				SequenceNumber:  2,
				Gist:            "5-year non-compete worldwide.",
				RiskLevel:       "high",
				Decision:        "rejected",
				Annotation:      "Scope too broad — limit to 12 months and home country.",
				Recommendations: "Change to: 'Non-compete shall be limited to 12 months and the Vendor's country of incorporation.'",
			},
		},
	}

	out := p.Render()

	if !strings.Contains(out, "Scope too broad") {
		t.Error("rejected clause annotation missing from rendered prompt")
	}
	if !strings.Contains(out, "12 months") {
		t.Error("recommendation text missing from rendered prompt")
	}
}

func TestSummarizationPrompt_EmptyAnnotationOmitted(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:       "contract.pdf",
		ReviewingParty: "client",
		RiskCounts:     prompts.RiskCounts{Low: 1},
		ReviewCounts:   prompts.ReviewCounts{Approved: 1},
		ClauseInputs: []prompts.ClauseInput{
			{
				SequenceNumber: 1,
				Gist:           "Standard payment terms.",
				RiskLevel:      "low",
				Decision:       "approved",
				Annotation:     "",
			},
		},
	}

	out := p.Render()

	if strings.Contains(out, "Human note:") {
		t.Error("empty annotation should not appear in rendered prompt")
	}
}

func TestSummarizationPrompt_RecommendationsOmittedWhenEmpty(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:       "contract.pdf",
		ReviewingParty: "client",
		ClauseInputs: []prompts.ClauseInput{
			{
				SequenceNumber:  1,
				Gist:            "Standard NDA.",
				RiskLevel:       "low",
				Recommendations: "",
			},
		},
	}

	out := p.Render()

	if strings.Contains(out, "Recommended edit:") {
		t.Error("empty recommendations should not appear in rendered prompt")
	}
}

func TestSummarizationPrompt_DefaultsPartyToClient(t *testing.T) {
	// ReviewingParty intentionally omitted
	p := prompts.SummarizationPrompt{
		Filename:     "contract.pdf",
		ClauseInputs: []prompts.ClauseInput{},
	}

	out := p.Render()

	if !strings.Contains(out, "client") {
		t.Error("ReviewingParty should default to 'client' when empty")
	}
}

func TestSummarizationPrompt_GoverningLawInPrompt(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:       "contract.pdf",
		ReviewingParty: "client",
		GoverningLaw:   "California",
		ContractType:   "Employment Agreement",
		ClauseInputs:   []prompts.ClauseInput{},
	}

	out := p.Render()

	if !strings.Contains(out, "California") {
		t.Error("GoverningLaw missing from rendered prompt")
	}
	if !strings.Contains(out, "Employment Agreement") {
		t.Error("ContractType missing from rendered prompt")
	}
}

func TestSummarizationPrompt_MediumClusterWarningInPriorityIssues(t *testing.T) {
	clauses := make([]prompts.ClauseInput, 4)
	for i := range clauses {
		clauses[i] = prompts.ClauseInput{
			SequenceNumber: i + 1,
			Gist:           "Medium risk clause.",
			RiskLevel:      "medium",
			Decision:       "approved",
		}
	}
	p := prompts.SummarizationPrompt{
		Filename:       "contract.pdf",
		ReviewingParty: "client",
		RiskCounts:     prompts.RiskCounts{Medium: 4},
		ClauseInputs:   clauses,
	}

	out := p.Render()

	if !strings.Contains(out, "medium-risk cluster") {
		t.Error("medium-risk cluster warning missing from Priority Issues instructions when medium count >= 3")
	}
}

func TestSummarizationPrompt_RejectedNoRecommendationShowsFallback(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:       "contract.pdf",
		ReviewingParty: "client",
		ClauseInputs: []prompts.ClauseInput{
			{
				SequenceNumber:  3,
				Gist:            "Perpetual irrevocable IP assignment.",
				RiskLevel:       "high",
				Decision:        "rejected",
				Annotation:      "Unacceptable — IP must revert on termination.",
				Recommendations: "",
			},
		},
	}

	out := p.Render()

	if !strings.Contains(out, "Negotiation required") {
		t.Error("rejected clause with no Recommendations must show 'Negotiation required' fallback")
	}
}
