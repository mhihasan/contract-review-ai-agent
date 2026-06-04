package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/tokens"
)

func makeMessages(n int, contentSize int) []llm.Message {
	content := strings.Repeat("word ", contentSize)
	msgs := make([]llm.Message, n)
	for i := range msgs {
		role := llm.RoleUser
		if i%2 == 1 {
			role = llm.RoleAssistant
		}
		msgs[i] = llm.Message{Role: role, Content: content}
	}
	return msgs
}

func TestPrepare_underThreshold_returnsUnchanged(t *testing.T) {
	cm := NewContextManager("gpt-4o-mini", 100000, 0.8, 4, &llm.Fake{})

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "short"},
		{Role: llm.RoleAssistant, Content: "ok"},
	}
	got, err := cm.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(got) != len(msgs) {
		t.Fatalf("expected %d msgs, got %d", len(msgs), len(got))
	}
}

func TestPrepare_overWindow_reducesBelow(t *testing.T) {
	window := 500
	cm := NewContextManager("gpt-4o-mini", window, 0.8, 2, &llm.Fake{})

	// build a history that is definitely over the window
	msgs := makeMessages(30, 20)
	before := tokens.CountMessages("gpt-4o-mini", msgs)
	if before <= window {
		t.Skipf("generated msgs only %d tokens; need more than %d for this test", before, window)
	}

	got, err := cm.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	after := tokens.CountMessages("gpt-4o-mini", got)
	if after > window {
		t.Errorf("after Prepare: %d tokens, want <= %d", after, window)
	}
}

func TestPrepare_preservesFirstAndRecentMessages(t *testing.T) {
	window := 300
	keepRecent := 2
	cm := NewContextManager("gpt-4o-mini", window, 0.8, keepRecent, &llm.Fake{})

	msgs := makeMessages(20, 20)
	msgs[0] = llm.Message{Role: llm.RoleUser, Content: "TASK_SENTINEL_MESSAGE"}

	got, err := cm.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	if got[0].Content != "TASK_SENTINEL_MESSAGE" {
		t.Errorf("first message was not preserved; got %q", got[0].Content)
	}

	last := msgs[len(msgs)-1]
	found := false
	for _, m := range got {
		if m.Content == last.Content && m.Role == last.Role {
			found = true
			break
		}
	}
	if !found {
		t.Error("last message was not preserved in output")
	}
}

func TestPrepare_summarizePath_collapsesMidBlock(t *testing.T) {
	window := 300
	keepRecent := 2
	summaryText := "SUMMARIZED_MIDDLE_BLOCK"
	fake := &llm.Fake{
		Script: []llm.CompletionResponse{
			{Content: summaryText},
		},
	}
	cm := NewContextManager("gpt-4o-mini", window, 0.8, keepRecent, fake)

	msgs := makeMessages(20, 20)

	got, err := cm.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	found := false
	for _, m := range got {
		if m.Content == summaryText {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a summary message in output, none found")
	}
}

func TestPrepare_toolResultTruncation(t *testing.T) {
	window := 300
	cm := NewContextManager("gpt-4o-mini", window, 0.8, 1, &llm.Fake{})

	bulkyContent := strings.Repeat("x", 2000)
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "task"},
		{Role: llm.RoleTool, Content: bulkyContent, ToolCallID: "call-1"},
		{Role: llm.RoleTool, Content: bulkyContent, ToolCallID: "call-2"},
		{Role: llm.RoleAssistant, Content: "recent"},
	}

	got, err := cm.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	for _, m := range got {
		if m.Role == llm.RoleTool && len(m.Content) >= len(bulkyContent) {
			t.Errorf("tool result was not truncated; content length %d", len(m.Content))
		}
	}
}
