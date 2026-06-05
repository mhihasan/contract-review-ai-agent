package logfmt_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/logfmt"
)

func TestHandler_DebugLineIsIndented(t *testing.T) {
	var buf bytes.Buffer
	h := logfmt.NewHandler(&buf, slog.LevelDebug)
	l := slog.New(h)
	l.Debug("step 1 tool call", "tool", "get_contract_section", "args", `{"reference":"7.2"}`)
	out := buf.String()

	if !strings.HasPrefix(out, "    ") {
		t.Fatalf("DEBUG line should be indented with 4 spaces, got: %q", out)
	}
	if !strings.Contains(out, "step 1 tool call") {
		t.Fatalf("DEBUG line should contain the message, got: %q", out)
	}
	if !strings.Contains(out, "get_contract_section") {
		t.Fatalf("DEBUG line should contain attr value, got: %q", out)
	}
	if strings.Contains(out, "level=") {
		t.Fatalf("DEBUG line must not contain 'level=', got: %q", out)
	}
	if strings.Contains(out, "time=") {
		t.Fatalf("DEBUG line must not contain 'time=', got: %q", out)
	}
}

func TestHandler_InfoLineIsCompact(t *testing.T) {
	var buf bytes.Buffer
	h := logfmt.NewHandler(&buf, slog.LevelDebug)
	l := slog.New(h)
	l.Info("clause done", "clause_id", "c-001", "stop", "submitted", "steps", 3)
	out := buf.String()

	if strings.HasPrefix(out, "    ") {
		t.Fatalf("INFO line should NOT be indented, got: %q", out)
	}
	if !strings.Contains(out, "clause done") {
		t.Fatalf("INFO line should contain the message, got: %q", out)
	}
	if !strings.Contains(out, "c-001") {
		t.Fatalf("INFO line should contain attr value, got: %q", out)
	}
	if strings.Contains(out, "level=") {
		t.Fatalf("INFO line must not contain 'level=', got: %q", out)
	}
}

func TestHandler_WarnAndErrorHavePrefix(t *testing.T) {
	var buf bytes.Buffer
	h := logfmt.NewHandler(&buf, slog.LevelDebug)
	l := slog.New(h)
	l.Warn("something odd", "key", "val")
	l.Error("something failed", "err", "oops")
	out := buf.String()

	if !strings.Contains(out, "WARN") {
		t.Fatalf("WARN line should contain 'WARN', got: %q", out)
	}
	if !strings.Contains(out, "ERROR") {
		t.Fatalf("ERROR line should contain 'ERROR', got: %q", out)
	}
}

func TestHandler_DebugSuppressedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	h := logfmt.NewHandler(&buf, slog.LevelInfo)
	l := slog.New(h)
	l.Debug("this should not appear")
	l.Info("this should appear")
	out := buf.String()

	if strings.Contains(out, "this should not appear") {
		t.Fatalf("DEBUG line must be suppressed at INFO level, got: %q", out)
	}
	if !strings.Contains(out, "this should appear") {
		t.Fatalf("INFO line should appear at INFO level, got: %q", out)
	}
}

func TestHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := logfmt.NewHandler(&buf, slog.LevelDebug)
	l := slog.New(h).With("contract_id", "abc123")
	l.Info("stage start")
	out := buf.String()

	if !strings.Contains(out, "abc123") {
		t.Fatalf("pre-set attr 'abc123' should appear in output, got: %q", out)
	}
}

// Ensure Handle can be called from multiple goroutines without a race.
func TestHandler_Concurrent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logfmt.NewHandler(&buf, slog.LevelDebug)
	l := slog.New(h)
	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func(n int) {
			l.Debug("concurrent", "n", n)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}
