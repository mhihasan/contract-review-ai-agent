package logfmt

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

type Handler struct {
	mu       sync.Mutex
	w        io.Writer
	level    slog.Level
	preAttrs []slog.Attr
}

func NewHandler(w io.Writer, level slog.Level) *Handler {
	return &Handler{w: w, level: level}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder

	switch r.Level {
	case slog.LevelDebug:
		b.WriteString("    ")
		b.WriteString(r.Message)
	case slog.LevelInfo:
		b.WriteString(r.Message)
	case slog.LevelWarn:
		b.WriteString("WARN  ")
		b.WriteString(r.Message)
	case slog.LevelError:
		b.WriteString("ERROR ")
		b.WriteString(r.Message)
	default:
		b.WriteString("ERROR ")
		b.WriteString(r.Message)
	}

	for _, a := range h.preAttrs {
		fmt.Fprintf(&b, "   %s=%v", a.Key, a.Value)
	}
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&b, "   %s=%v", a.Key, a.Value)
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := &Handler{w: h.w, level: h.level}
	next.preAttrs = append(next.preAttrs, h.preAttrs...)
	next.preAttrs = append(next.preAttrs, attrs...)
	return next
}

func (h *Handler) WithGroup(_ string) slog.Handler {
	return h
}
