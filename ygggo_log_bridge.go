package ygggo_mysql

import (
	"context"
	"log/slog"

	ggl "github.com/yggai/ygggo_log"
)

// ygggoHandler bridges slog records to ygggo_log package-level logging.
// It respects slog levels and forwards attributes as key-value pairs.
type ygggoHandler struct {
	group string
	attrs []slog.Attr
}

func newYgggoHandler() slog.Handler {
	// Ensure env-driven logging is initialized. It is safe to call multiple times.
	ggl.InitLogEnv()
	return &ygggoHandler{}
}

func (h *ygggoHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *ygggoHandler) Handle(_ context.Context, r slog.Record) error {
	// Collect attributes: handler-level attrs + record attrs
	kvs := make([]any, 0, len(h.attrs)*2+8)
	appendAttr := func(a slog.Attr) {
		// Best-effort: forward key and value as-is
		kvs = append(kvs, a.Key, a.Value.Any())
	}
	for _, a := range h.attrs {
		appendAttr(a)
	}
	r.Attrs(func(a slog.Attr) bool { appendAttr(a); return true })
	if h.group != "" {
		kvs = append(kvs, "group", h.group)
	}

	// Map slog level to ygggo_log level
	switch {
	case r.Level >= slog.LevelError:
		ggl.Error(r.Message, kvs...)
	case r.Level >= slog.LevelWarn:
		ggl.Warning(r.Message, kvs...)
	case r.Level >= slog.LevelInfo:
		ggl.Info(r.Message, kvs...)
	default:
		ggl.Debug(r.Message, kvs...)
	}
	return nil
}

func (h *ygggoHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &nh
}

func (h *ygggoHandler) WithGroup(name string) slog.Handler {
	nh := *h
	if nh.group == "" {
		nh.group = name
	} else if name != "" {
		nh.group = nh.group + "." + name
	}
	return &nh
}

// newYgggoSlogLoggerFromEnv builds a slog.Logger backed by ygggo_log, using env configuration.
func newYgggoSlogLoggerFromEnv() *slog.Logger {
	// Initialize env-based logger once
	ggl.InitLogEnv()
	return slog.New(newYgggoHandler())
}

// UseYgggoLoggerFromEnv forces this pool to use ygggo_log as slog backend and enables logging.
func (p *Pool) UseYgggoLoggerFromEnv() {
	if p == nil {
		return
	}
	p.logger = newYgggoSlogLoggerFromEnv()
	p.loggingEnabled = true
}
