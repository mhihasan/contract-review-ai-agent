package llm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/llm"
)

func TestFake_ReturnsScriptedResponses(t *testing.T) {
	f := &llm.Fake{
		Script: []llm.CompletionResponse{
			{Content: "first", StopReason: "end_turn"},
			{Content: "second", StopReason: "end_turn"},
		},
	}

	ctx := context.Background()
	req := llm.CompletionRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}}

	r1, err := f.Complete(ctx, req)
	if err != nil {
		t.Fatalf("call 1: unexpected error: %v", err)
	}
	if r1.Content != "first" {
		t.Errorf("call 1 content = %q, want %q", r1.Content, "first")
	}

	r2, err := f.Complete(ctx, req)
	if err != nil {
		t.Fatalf("call 2: unexpected error: %v", err)
	}
	if r2.Content != "second" {
		t.Errorf("call 2 content = %q, want %q", r2.Content, "second")
	}
}

func TestFake_RecordsRequests(t *testing.T) {
	f := &llm.Fake{
		Script: []llm.CompletionResponse{{Content: "ok", StopReason: "end_turn"}},
	}
	ctx := context.Background()
	req := llm.CompletionRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "record me"}}}

	_, _ = f.Complete(ctx, req)

	if len(f.Calls) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(f.Calls))
	}
	if f.Calls[0].Messages[0].Content != "record me" {
		t.Errorf("recorded content = %q, want %q", f.Calls[0].Messages[0].Content, "record me")
	}
}

func TestFake_ForcedError(t *testing.T) {
	sentinel := errors.New("transport exploded")
	f := &llm.Fake{Err: sentinel}

	ctx := context.Background()
	_, err := f.Complete(ctx, llm.CompletionRequest{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestFake_ExhaustedScript_ReturnsError(t *testing.T) {
	f := &llm.Fake{
		Script: []llm.CompletionResponse{{Content: "only one", StopReason: "end_turn"}},
	}
	ctx := context.Background()
	_, _ = f.Complete(ctx, llm.CompletionRequest{})
	_, err := f.Complete(ctx, llm.CompletionRequest{})
	if err == nil {
		t.Fatal("expected error when script is exhausted, got nil")
	}
}

func TestFake_ImplementsLLM(_ *testing.T) {
	var _ llm.LLM = &llm.Fake{}
}
