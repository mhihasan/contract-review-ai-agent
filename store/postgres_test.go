package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/store"
)

func TestNewPool_ConnectsAndPings(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}

	ctx := context.Background()
	pool, err := store.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("expected pool to connect, got error: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping after connect failed: %v", err)
	}
}

func TestNewPool_BadURL(t *testing.T) {
	ctx := context.Background()
	_, err := store.NewPool(ctx, "postgres://bad-host:9999/nope?sslmode=disable")
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
}
