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
	SequenceNumber    int
	Gist              string
	RiskLevel         string
	Decision          string
	Annotation        string
	Recommendations   string
	AmbiguousLanguage string
}

type SummarizationPrompt struct {
	Filename       string
	ReviewingParty string // "client" or "vendor"
	GoverningLaw   string // e.g. "New York", "England & Wales"; empty = not specified
	ContractType   string // e.g. "MSA", "NDA", "Employment Agreement"; empty = not specified
	RiskCounts     RiskCounts
	ReviewCounts   ReviewCounts
	ClauseInputs   []ClauseInput
}

func (p SummarizationPrompt) Render() string {
	party := p.ReviewingParty
	if party == "" {
		party = "client"
	}

	var b strings.Builder

	fmt.Fprintf(&b, "You are a senior contract lawyer reviewing on behalf of the %s.\n", party)
	fmt.Fprintf(&b, "Contract file: %q\n", p.Filename)
	if p.ContractType != "" {
		fmt.Fprintf(&b, "Contract type: %s\n", p.ContractType)
	}
	if p.GoverningLaw != "" {
		fmt.Fprintf(&b, "Governing law: %s\n", p.GoverningLaw)
	} else {
		fmt.Fprint(&b, "Governing law: not specified — flag any recommendations that depend on jurisdiction.\n")
	}
	fmt.Fprint(&b, "\n")
	fmt.Fprint(&b, "Every risk, issue, and recommendation below must be framed from the "+party+"'s perspective.\n\n")

	fmt.Fprint(&b, "The pre-computed counts below are authoritative — reproduce them verbatim; do not recount.\n\n")
	fmt.Fprintf(&b, "Risk counts: High: %d, Medium: %d, Low: %d\n", p.RiskCounts.High, p.RiskCounts.Medium, p.RiskCounts.Low)
	fmt.Fprintf(&b, "Review counts: Approved: %d, Rejected: %d, Overrides: %d\n\n", p.ReviewCounts.Approved, p.ReviewCounts.Rejected, p.ReviewCounts.Overrides)

	fmt.Fprint(&b, "## Clause Data\n\n")
	for _, c := range p.ClauseInputs {
		fmt.Fprintf(&b, "### Clause %d [%s]\n", c.SequenceNumber, strings.ToUpper(c.RiskLevel))
		fmt.Fprintf(&b, "Issue: %s\n", c.Gist)
		if c.AmbiguousLanguage != "" {
			fmt.Fprintf(&b, "Ambiguous language: %s\n", c.AmbiguousLanguage)
		}
		if c.Recommendations != "" {
			fmt.Fprintf(&b, "Recommended edit: %s\n", c.Recommendations)
		} else if c.Decision == "rejected" {
			fmt.Fprint(&b, "Recommended edit: Negotiation required — no draft language provided.\n")
		}
		if c.Decision != "" {
			fmt.Fprintf(&b, "Human decision: %s\n", c.Decision)
		}
		if c.Annotation != "" {
			fmt.Fprintf(&b, "Human note: %s\n", c.Annotation)
		}
		fmt.Fprint(&b, "\n")
	}

	fmt.Fprint(&b, "---\n\n")
	fmt.Fprint(&b, "Using ONLY the data above, produce a professional contract review report in clean markdown.\n")
	fmt.Fprint(&b, "Start with `# Contract Review Report` and include exactly these five sections:\n\n")
	fmt.Fprint(&b, "1. **Executive Summary** — 3-5 sentences: overall risk profile from the "+party+"'s perspective, top concerns, and a clear signing recommendation (Do not sign / Sign with changes / Sign as-is).\n\n")
	fmt.Fprint(&b, "2. **Signing Recommendation** — one of: `⛔ Do Not Sign`, `⚠️ Sign With Changes`, or `✅ Sign As-Is`. Follow with 1-2 sentences explaining why.\n\n")

	mediumClusterNote := ""
	if p.RiskCounts.Medium >= 3 {
		mediumClusterNote = fmt.Sprintf(" Also include a medium-risk cluster warning: flag that there are %d medium-risk clauses that collectively require negotiation even if none individually triggers a 'Do Not Sign'.", p.RiskCounts.Medium)
	}
	fmt.Fprintf(&b, "3. **Priority Issues** — bullet list of ALL high-risk clauses. Each bullet: clause number, one-sentence issue, and the recommended edit verbatim from the data above (if the recommended edit says 'Negotiation required', reproduce that verbatim). If no high-risk clauses exist, write 'No high-risk clauses identified.'%s\n\n", mediumClusterNote)

	fmt.Fprint(&b, "4. **Risk Breakdown** — reproduce the counts verbatim:\n")
	fmt.Fprint(&b, "   - High / Medium / Low risk clause counts\n")
	fmt.Fprint(&b, "   - Approved / Rejected / Overrides (human review) counts\n\n")
	fmt.Fprint(&b, "5. **Clause-by-Clause Detail** — markdown table with columns: Clause | Risk | Decision | Issue | Recommended Edit. Rejected clauses must include the human note in the Issue cell. If a clause was rejected and has no recommended edit, the Recommended Edit cell must say 'Negotiation required — no draft language provided' rather than being left blank.\n\n")
	fmt.Fprint(&b, "Rules: No preamble. No commentary outside the five sections. Output clean markdown only.\n")

	return b.String()
}
