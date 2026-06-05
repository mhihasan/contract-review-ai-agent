package llm

import (
	"context"
	"fmt"
	"sync"
)

type Fake struct {
	Script []CompletionResponse
	Err    error

	mu    sync.Mutex
	Calls []CompletionRequest
	idx   int
}

var _ LLM = (*Fake)(nil)

func (f *Fake) Complete(_ context.Context, req CompletionRequest) (CompletionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, req)
	if f.Err != nil {
		return CompletionResponse{}, f.Err
	}
	if f.idx >= len(f.Script) {
		return CompletionResponse{}, fmt.Errorf("fake: script exhausted after %d calls", f.idx)
	}
	resp := f.Script[f.idx]
	f.idx++
	return resp, nil
}

func NewFake(response string) LLM {
	return &Fake{Script: []CompletionResponse{{Content: response}}}
}

type fakeWithHook struct {
	response string
	hook     func()
}

func NewFakeWithHook(response string, hook func()) LLM {
	return &fakeWithHook{response: response, hook: hook}
}

func (f *fakeWithHook) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	if f.hook != nil {
		f.hook()
	}
	return CompletionResponse{Content: f.response}, nil
}

type fakeCapture struct {
	response string
	capture  *string
}

func NewFakeCapture(capture *string) LLM {
	return &fakeCapture{response: "# Report\n\nSummary here.", capture: capture}
}

func (f *fakeCapture) Complete(_ context.Context, req CompletionRequest) (CompletionResponse, error) {
	if len(req.Messages) > 0 {
		*f.capture = req.Messages[len(req.Messages)-1].Content
	}
	return CompletionResponse{Content: f.response}, nil
}
