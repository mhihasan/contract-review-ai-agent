package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mhihasan/contract-review-ai-agent/store"
)

type GetDefinition struct {
	store      store.Store
	contractID string
}

var _ Tool = (*GetDefinition)(nil)

func NewGetDefinition(s store.Store, contractID string) *GetDefinition {
	return &GetDefinition{store: s, contractID: contractID}
}

func (t *GetDefinition) Name() string { return "get_definition" }
func (t *GetDefinition) Description() string {
	return "Look up a defined term in the contract. Returns the definition text or 'not found'."
}
func (t *GetDefinition) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"term": {"type": "string", "description": "The defined term to look up"}
		},
		"required": ["term"]
	}`)
}

func (t *GetDefinition) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var req struct {
		Term string `json:"term"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return fmt.Sprintf("invalid args: %v", err), nil
	}
	if req.Term == "" {
		return "invalid args: term is required", nil
	}

	clauses, err := t.store.GetClauses(ctx, t.contractID)
	if err != nil {
		return "", fmt.Errorf("get clauses: %w", err)
	}

	termLower := strings.ToLower(req.Term)
	for _, c := range clauses {
		text := strings.ToLower(c.Text)
		if strings.Contains(text, fmt.Sprintf(`"%s" means`, termLower)) ||
			strings.Contains(text, fmt.Sprintf("'%s' means", termLower)) ||
			strings.Contains(text, fmt.Sprintf("%s means", termLower)) {
			return c.Text, nil
		}
	}
	return fmt.Sprintf("definition of %q not found in contract", req.Term), nil
}

type GetContractSection struct {
	store      store.Store
	contractID string
}

var _ Tool = (*GetContractSection)(nil)

func NewGetContractSection(s store.Store, contractID string) *GetContractSection {
	return &GetContractSection{store: s, contractID: contractID}
}

func (t *GetContractSection) Name() string { return "get_contract_section" }
func (t *GetContractSection) Description() string {
	return "Retrieve a specific section or clause from the contract by reference (e.g. 'Section 7.2' or sequence number)."
}
func (t *GetContractSection) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"reference": {"type": "string", "description": "Section name, number, or clause sequence number"}
		},
		"required": ["reference"]
	}`)
}

func (t *GetContractSection) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var req struct {
		Reference string `json:"reference"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return fmt.Sprintf("invalid args: %v", err), nil
	}
	if req.Reference == "" {
		return "invalid args: reference is required", nil
	}

	clauses, err := t.store.GetClauses(ctx, t.contractID)
	if err != nil {
		return "", fmt.Errorf("get clauses: %w", err)
	}

	refLower := strings.ToLower(req.Reference)
	for _, c := range clauses {
		if strings.Contains(strings.ToLower(c.Text), refLower) {
			return c.Text, nil
		}
		if fmt.Sprintf("%d", c.SequenceNumber) == req.Reference {
			return c.Text, nil
		}
	}
	return fmt.Sprintf("section %q not found in contract", req.Reference), nil
}

type SearchClauseLibrary struct {
	store      store.Store
	contractID string
}

var _ Tool = (*SearchClauseLibrary)(nil)

func NewSearchClauseLibrary(s store.Store, contractID string) *SearchClauseLibrary {
	return &SearchClauseLibrary{store: s, contractID: contractID}
}

func (t *SearchClauseLibrary) Name() string { return "search_clause_library" }
func (t *SearchClauseLibrary) Description() string {
	return "Search the standard clause library for clauses matching a query. Returns type and excerpt of top matches."
}
func (t *SearchClauseLibrary) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "Search query — clause type name or keywords"}
		},
		"required": ["query"]
	}`)
}

func (t *SearchClauseLibrary) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var req struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return fmt.Sprintf("invalid args: %v", err), nil
	}
	if req.Query == "" {
		return "invalid args: query is required", nil
	}

	results, err := t.store.SearchClauseLibrary(ctx, req.Query)
	if err != nil {
		return "", fmt.Errorf("search clause library: %w", err)
	}
	if len(results) == 0 {
		return fmt.Sprintf("no clauses found matching %q", req.Query), nil
	}

	var sb strings.Builder
	for _, r := range results {
		excerpt := r.StandardText
		if len(excerpt) > 200 {
			excerpt = excerpt[:200] + "..."
		}
		fmt.Fprintf(&sb, "[%s] %s\n", r.ClauseType, excerpt)
	}
	return strings.TrimSpace(sb.String()), nil
}

type LookupStandardClause struct {
	store      store.Store
	contractID string
}

var _ Tool = (*LookupStandardClause)(nil)

func NewLookupStandardClause(s store.Store, contractID string) *LookupStandardClause {
	return &LookupStandardClause{store: s, contractID: contractID}
}

func (t *LookupStandardClause) Name() string { return "lookup_standard_clause" }
func (t *LookupStandardClause) Description() string {
	return "Retrieve the full standard baseline text for a clause type (e.g. 'liability', 'indemnity') for comparison."
}
func (t *LookupStandardClause) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"clause_type": {"type": "string", "description": "The clause type to retrieve (e.g. 'liability', 'indemnity', 'termination')"}
		},
		"required": ["clause_type"]
	}`)
}

func (t *LookupStandardClause) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var req struct {
		ClauseType string `json:"clause_type"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return fmt.Sprintf("invalid args: %v", err), nil
	}
	if req.ClauseType == "" {
		return "invalid args: clause_type is required", nil
	}

	c, err := t.store.GetStandardClause(ctx, req.ClauseType)
	if err != nil {
		return fmt.Sprintf("standard clause for type %q not found", req.ClauseType), nil
	}
	result := fmt.Sprintf("Type: %s\n\nStandard text:\n%s", c.ClauseType, c.StandardText)
	if c.Notes != "" {
		result += fmt.Sprintf("\n\nNotes: %s", c.Notes)
	}
	return result, nil
}
