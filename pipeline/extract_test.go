package pipeline

import (
	"context"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/domain"
	"github.com/mhihasan/contract-review-ai-agent/store"
)

type fakeStore struct {
	store.Store
	contracts map[string]domain.Contract
	statusLog []domain.ContractStatus
}

func newFakeStore() *fakeStore {
	return &fakeStore{contracts: map[string]domain.Contract{}}
}

func (f *fakeStore) CreateContract(_ context.Context, filename, rawText string) (domain.Contract, error) {
	c := domain.Contract{ID: "c1", Filename: filename, RawText: rawText, Status: domain.StatusUploaded}
	f.contracts[c.ID] = c
	return c, nil
}

func (f *fakeStore) GetContract(_ context.Context, id string) (domain.Contract, error) {
	return f.contracts[id], nil
}

func (f *fakeStore) UpdateContractStatus(_ context.Context, id string, s domain.ContractStatus) error {
	c := f.contracts[id]
	c.Status = s
	f.contracts[id] = c
	f.statusLog = append(f.statusLog, s)
	return nil
}

func (f *fakeStore) UpdateContractText(_ context.Context, id, rawText string) error {
	c := f.contracts[id]
	c.RawText = rawText
	f.contracts[id] = c
	return nil
}

func TestRunExtract_HappyPath(t *testing.T) {
	f := newFakeStore()
	id, err := RunExtract(context.Background(), f, extractFunc(func(context.Context, string) (string, error) {
		return "extracted body", nil
	}), "contract.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := f.contracts[id]
	if c.Status != domain.StatusExtracted {
		t.Fatalf("want status extracted, got %s", c.Status)
	}
	if c.RawText != "extracted body" {
		t.Fatalf("want persisted raw text, got %q", c.RawText)
	}
	want := []domain.ContractStatus{domain.StatusExtracting, domain.StatusExtracted}
	if len(f.statusLog) != len(want) || f.statusLog[0] != want[0] || f.statusLog[1] != want[1] {
		t.Fatalf("want status transitions %v, got %v", want, f.statusLog)
	}
}

func TestRunExtract_IdempotentNoOp(t *testing.T) {
	f := newFakeStore()
	f.contracts["c1"] = domain.Contract{ID: "c1", Status: domain.StatusExtracted, RawText: "already done"}

	id, err := RunExtractContract(context.Background(), f, extractFunc(func(context.Context, string) (string, error) {
		t.Fatal("extractor must not run for already-extracted contract")
		return "", nil
	}), "c1", "contract.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "c1" {
		t.Fatalf("want id c1, got %s", id)
	}
	if len(f.statusLog) != 0 {
		t.Fatalf("want no status writes on no-op, got %v", f.statusLog)
	}
}
