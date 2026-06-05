package agent

import (
	"sync"

	"github.com/mhihasan/contract-review-ai-agent/cost"
)

type BudgetSnapshot struct {
	UsedTokens  int
	UsedCostUSD float64
	UsedSteps   int
	MaxTokens   int
	MaxCostUSD  float64
	MaxSteps    int
}

func (s BudgetSnapshot) AsUsage() Usage {
	return Usage{
		InputTokens:  s.UsedTokens,
		OutputTokens: 0,
	}
}

type Budget struct {
	mu          sync.Mutex
	maxTokens   int
	maxCostUSD  float64
	maxSteps    int
	usedTokens  int
	usedCostUSD float64
	usedSteps   int
}

func NewBudget(maxTokens int, maxCostUSD float64, maxSteps int) *Budget {
	return &Budget{
		maxTokens:  maxTokens,
		maxCostUSD: maxCostUSD,
		maxSteps:   maxSteps,
	}
}

func (b *Budget) Record(provider, model string, in, out int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.usedTokens += in + out
	b.usedCostUSD += cost.Estimate(provider, model, in, out)
	b.usedSteps++
}

func (b *Budget) Exceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.maxTokens > 0 && b.usedTokens >= b.maxTokens {
		return true
	}
	if b.maxCostUSD > 0 && b.usedCostUSD >= b.maxCostUSD {
		return true
	}
	if b.maxSteps > 0 && b.usedSteps >= b.maxSteps {
		return true
	}
	return false
}

func (b *Budget) Snapshot() BudgetSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return BudgetSnapshot{
		UsedTokens:  b.usedTokens,
		UsedCostUSD: b.usedCostUSD,
		UsedSteps:   b.usedSteps,
		MaxTokens:   b.maxTokens,
		MaxCostUSD:  b.maxCostUSD,
		MaxSteps:    b.maxSteps,
	}
}
