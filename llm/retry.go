package llm

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type TransientError struct {
	Code int
	Msg  string
}

func (e *TransientError) Error() string { return fmt.Sprintf("transient %d: %s", e.Code, e.Msg) }

type PermanentError struct {
	Code int
	Msg  string
}

func (e *PermanentError) Error() string { return fmt.Sprintf("permanent %d: %s", e.Code, e.Msg) }

type RetryingLLM struct {
	inner      LLM
	maxRetries int
	baseDelay  time.Duration
}

func NewRetryingLLM(inner LLM, maxRetries int, baseDelay time.Duration) *RetryingLLM {
	return &RetryingLLM{inner: inner, maxRetries: maxRetries, baseDelay: baseDelay}
}

func (r *RetryingLLM) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if ctx.Err() != nil {
			return CompletionResponse{}, ctx.Err()
		}

		resp, err := r.inner.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}

		if _, ok := err.(*PermanentError); ok {
			return CompletionResponse{}, err
		}

		lastErr = err
		if attempt == r.maxRetries {
			break
		}

		delay := r.baseDelay*(1<<attempt) + time.Duration(rand.Int63n(int64(r.baseDelay)))
		select {
		case <-ctx.Done():
			return CompletionResponse{}, ctx.Err()
		case <-time.After(delay):
		}
	}
	return CompletionResponse{}, fmt.Errorf("llm: max retries exceeded: %w", lastErr)
}
