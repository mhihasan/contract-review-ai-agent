package llm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mhihasan/contract-review-ai-agent/llm"
)

type countingLLM struct {
	calls   int
	failFor int
	resp    llm.CompletionResponse
}

func (c *countingLLM) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	c.calls++
	if c.calls <= c.failFor {
		return llm.CompletionResponse{}, &llm.TransientError{Code: 429, Msg: "rate limited"}
	}
	return c.resp, nil
}

type permanentFailLLM struct{ calls int }

func (p *permanentFailLLM) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	p.calls++
	return llm.CompletionResponse{}, &llm.PermanentError{Code: 401, Msg: "unauthorized"}
}

func TestRetryingLLM_SucceedsAfterTransientFailures(t *testing.T) {
	inner := &countingLLM{failFor: 2, resp: llm.CompletionResponse{Content: "ok"}}
	r := llm.NewRetryingLLM(inner, 3, 1*time.Millisecond)

	resp, err := r.Complete(context.Background(), llm.CompletionRequest{})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected 'ok', got %q", resp.Content)
	}
	if inner.calls != 3 {
		t.Errorf("expected inner called 3 times, got %d", inner.calls)
	}
}

func TestRetryingLLM_ContextCancelStopsRetry(t *testing.T) {
	inner := &countingLLM{failFor: 10}
	r := llm.NewRetryingLLM(inner, 5, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := r.Complete(ctx, llm.CompletionRequest{})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("should have returned quickly on cancelled ctx, took %v", elapsed)
	}
}

func TestRetryingLLM_NoRetryOnPermanentError(t *testing.T) {
	permInner := &permanentFailLLM{}
	r := llm.NewRetryingLLM(permInner, 3, 1*time.Millisecond)

	_, err := r.Complete(context.Background(), llm.CompletionRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if permInner.calls != 1 {
		t.Errorf("expected 1 call (no retry on permanent error), got %d", permInner.calls)
	}
}

func TestRetryingLLM_ExhaustsRetries(t *testing.T) {
	inner := &countingLLM{failFor: 10}
	r := llm.NewRetryingLLM(inner, 3, 1*time.Millisecond)

	_, err := r.Complete(context.Background(), llm.CompletionRequest{})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if inner.calls != 4 {
		t.Errorf("expected 4 calls (1 initial + 3 retries), got %d", inner.calls)
	}
}
