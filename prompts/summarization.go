package prompts

import (
	"fmt"
	"strings"
)

type RiskCounts struct {
	High   int
	Medium int
	Low    int
}

type ReviewCounts struct {
	Approved  int
	Rejected  int
	Overrides int
}

type ClauseInput struct {
	SequenceNumber int
	Gist           string
	RiskLevel      string
	Decision       string
	Annotation     string
}

type SummarizationPrompt struct {
	Filename     string
	RiskCounts   RiskCounts
	ReviewCounts ReviewCounts
	ClauseInputs []ClauseInput
}

func (p SummarizationPrompt) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "You are a contract-risk analyst. Produce a professional markdown report for the contract %q.\n\n", p.Filename)
	fmt.Fprint(&b, "The counts below were computed from the database — use them verbatim in the report; do not recount.\n\n")

	fmt.Fprint(&b, "## Risk Breakdown\n\n")
	fmt.Fprintf(&b, "- High: %d\n", p.RiskCounts.High)
	fmt.Fprintf(&b, "- Medium: %d\n", p.RiskCounts.Medium)
	fmt.Fprintf(&b, "- Low: %d\n\n", p.RiskCounts.Low)

	fmt.Fprint(&b, "## Review Summary\n\n")
	fmt.Fprintf(&b, "- Approved: %d\n", p.ReviewCounts.Approved)
	fmt.Fprintf(&b, "- Rejected: %d\n", p.ReviewCounts.Rejected)
	fmt.Fprintf(&b, "- Overrides: %d\n\n", p.ReviewCounts.Overrides)

	fmt.Fprint(&b, "## Clause-by-Clause Detail\n\n")
	for _, c := range p.ClauseInputs {
		fmt.Fprintf(&b, "**Clause %d** | Risk: %s | Decision: %s\n", c.SequenceNumber, c.RiskLevel, c.Decision)
		fmt.Fprintf(&b, "Gist: %s\n", c.Gist)
		if c.Annotation != "" {
			fmt.Fprintf(&b, "Annotation: %s\n", c.Annotation)
		}
		fmt.Fprint(&b, "\n")
	}

	fmt.Fprint(&b, "---\n\n")
	fmt.Fprint(&b, "Using the data above, write a complete contract review report with exactly these four sections:\n\n")
	fmt.Fprint(&b, "1. **Executive Summary** — overall risk profile in 2-4 sentences.\n")
	fmt.Fprint(&b, "2. **Key Findings** — top 3-5 issues; note where the reviewer agreed with or overrode the AI assessment.\n")
	fmt.Fprint(&b, "3. **Risk Breakdown** — reproduce the high/medium/low and approved/rejected counts verbatim from above.\n")
	fmt.Fprint(&b, "4. **Clause-by-Clause Detail** — one row per clause; rejected clauses must include the annotation text.\n\n")
	fmt.Fprint(&b, "Output clean markdown only. No preamble. Start with `# Contract Review Report`.\n")

	return b.String()
}
