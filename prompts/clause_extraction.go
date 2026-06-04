package prompts

import "fmt"

type ClauseExtractionPrompt struct {
	ContractText string
}

func (p ClauseExtractionPrompt) Render() string {
	return fmt.Sprintf(`You are a contract analysis assistant. Your task is to split the following contract text into individual, self-contained clauses.

Rules:
- Each clause must be a complete, meaningful unit that can be understood in isolation.
- Return ONLY a JSON array of strings. No prose, no explanation, no markdown fences.
- Each string in the array is the full text of one clause.
- A short document with a single paragraph must yield exactly one clause.

Example output format:
["The Vendor shall deliver the goods within 30 days of order.", "Payment is due net 30."]

Contract text:
%s`, p.ContractText)
}
