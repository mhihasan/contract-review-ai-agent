package prompts_test

import (
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/prompts"
)

func TestSummarizationPrompt_ContainsSections(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:     "service-agreement.pdf",
		RiskCounts:   prompts.RiskCounts{High: 3, Medium: 5, Low: 2},
		ReviewCounts: prompts.ReviewCounts{Approved: 7, Rejected: 3, Overrides: 1},
		ClauseInputs: []prompts.ClauseInput{
			{
				SequenceNumber: 1,
				Gist:           "Unlimited liability for contractor.",
				RiskLevel:      "high",
				Decision:       "rejected",
				Annotation:     "Unacceptable — must cap at contract value.",
			},
		},
	}

	out := p.Render()

	required := []string{
		"Executive Summary",
		"Key Findings",
		"Risk Breakdown",
		"Clause-by-Clause",
		"High: 3",
		"Medium: 5",
		"Low: 2",
		"Approved: 7",
		"Rejected: 3",
		"Overrides: 1",
		"Unacceptable — must cap at contract value.",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("Render() missing %q", want)
		}
	}
}

func TestSummarizationPrompt_RejectedClauseHasAnnotation(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:     "nda.pdf",
		RiskCounts:   prompts.RiskCounts{High: 1},
		ReviewCounts: prompts.ReviewCounts{Rejected: 1},
		ClauseInputs: []prompts.ClauseInput{
			{
				SequenceNumber: 2,
				Gist:           "5-year non-compete worldwide.",
				RiskLevel:      "high",
				Decision:       "rejected",
				Annotation:     "Scope too broad — limit to 12 months and home country.",
			},
		},
	}

	out := p.Render()

	if !strings.Contains(out, "Scope too broad") {
		t.Error("rejected clause annotation missing from rendered prompt")
	}
}

func TestSummarizationPrompt_EmptyAnnotationOmitted(t *testing.T) {
	p := prompts.SummarizationPrompt{
		Filename:     "contract.pdf",
		RiskCounts:   prompts.RiskCounts{Low: 1},
		ReviewCounts: prompts.ReviewCounts{Approved: 1},
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

	if strings.Contains(out, "Annotation:") {
		t.Error("empty annotation should not appear in rendered prompt")
	}
}
