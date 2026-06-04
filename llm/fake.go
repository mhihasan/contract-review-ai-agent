package llm

import (
	"context"
	"fmt"
)

type Fake struct {
	Script []CompletionResponse
	Calls  []CompletionRequest
	Err    error
	idx    int
}

var _ LLM = (*Fake)(nil)

func (f *Fake) Complete(_ context.Context, req CompletionRequest) (CompletionResponse, error) {
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
