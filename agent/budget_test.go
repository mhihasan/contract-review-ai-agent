package agent_test

import (
	"sync"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/agent"
)

func TestBudget_TokenCap_ExceededAfterRecord(t *testing.T) {
	b := agent.NewBudget(1000, 10.0, 100)
	if b.Exceeded() {
		t.Fatal("brand-new budget must not be exceeded")
	}
	b.Record("openai", "gpt-4o-mini", 600, 500)
	if !b.Exceeded() {
		t.Error("budget must be exceeded after 1100 tokens against cap of 1000")
	}
}

func TestBudget_CostCap_ExceededAfterRecord(t *testing.T) {
	b := agent.NewBudget(1_000_000, 0.001, 100)
	b.Record("openai", "gpt-4o", 0, 200)
	if !b.Exceeded() {
		t.Error("budget must be exceeded after cost crosses 0.001 USD cap")
	}
}

func TestBudget_StepCap_ExceededAfterRecord(t *testing.T) {
	b := agent.NewBudget(1_000_000, 100.0, 2)
	b.Record("openai", "gpt-4o-mini", 10, 10)
	b.Record("openai", "gpt-4o-mini", 10, 10)
	if !b.Exceeded() {
		t.Error("budget must be exceeded after 2 records against step cap of 2")
	}
}

func TestBudget_Snapshot_ReflectsAccumulatedUsage(t *testing.T) {
	b := agent.NewBudget(100_000, 10.0, 50)
	b.Record("openai", "gpt-4o-mini", 100, 50)
	b.Record("openai", "gpt-4o-mini", 200, 100)

	snap := b.Snapshot()
	if snap.UsedTokens != 450 {
		t.Errorf("UsedTokens = %d, want 450", snap.UsedTokens)
	}
	if snap.UsedSteps != 2 {
		t.Errorf("UsedSteps = %d, want 2", snap.UsedSteps)
	}
	if snap.UsedCostUSD <= 0 {
		t.Errorf("UsedCostUSD must be > 0, got %v", snap.UsedCostUSD)
	}
}

func TestBudget_UnlimitedCaps_NeverExceeded(t *testing.T) {
	b := agent.NewBudget(0, 0, 0)
	b.Record("openai", "gpt-4o", 10_000_000, 10_000_000)
	if b.Exceeded() {
		t.Error("zero cap means unlimited — Exceeded must return false")
	}
}

func TestBudget_Record_RaceFree(t *testing.T) {
	b := agent.NewBudget(0, 0, 0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Record("openai", "gpt-4o-mini", 10, 10)
		}()
	}
	wg.Wait()
	snap := b.Snapshot()
	if snap.UsedTokens != 2000 {
		t.Errorf("UsedTokens = %d after 100 concurrent records of 20 tokens each, want 2000", snap.UsedTokens)
	}
}
