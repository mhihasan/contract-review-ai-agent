package tokens_test

import (
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/tokens"
)

func TestCount_nonzero(t *testing.T) {
	n := tokens.Count("gpt-4o-mini", "Hello, world!")
	if n <= 0 {
		t.Fatalf("Count returned %d, want > 0", n)
	}
}

func TestCount_longerTextHasMoreTokens(t *testing.T) {
	short := tokens.Count("gpt-4o-mini", "Hi")
	long := tokens.Count("gpt-4o-mini", "This is a much longer sentence that should produce more tokens than a single greeting.")
	if long <= short {
		t.Fatalf("expected long (%d) > short (%d)", long, short)
	}
}

func TestCountMessages_sumsPlusOverhead(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "Analyze this clause."},
		{Role: llm.RoleAssistant, Content: "I will look it up."},
	}
	n := tokens.CountMessages("gpt-4o-mini", msgs)
	if n <= 0 {
		t.Fatalf("CountMessages returned %d, want > 0", n)
	}
}

func TestCountMessages_moreMessagesMoreTokens(t *testing.T) {
	one := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello."},
	}
	two := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello."},
		{Role: llm.RoleAssistant, Content: "I have reviewed the clause and found several issues worth noting."},
	}
	n1 := tokens.CountMessages("gpt-4o-mini", one)
	n2 := tokens.CountMessages("gpt-4o-mini", two)
	if n2 <= n1 {
		t.Fatalf("expected two-message count (%d) > one-message count (%d)", n2, n1)
	}
}

func TestCount_unknownModelFallback(t *testing.T) {
	n := tokens.Count("claude-3-5-sonnet-20241022", "Hello, world!")
	if n <= 0 {
		t.Fatalf("Count fallback returned %d, want > 0", n)
	}
}
