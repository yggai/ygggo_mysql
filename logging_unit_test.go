package ygggo_mysql

import (
	"log/slog"
	"testing"
)

// These tests avoid hitting TestMain's Docker dependency by not creating pools.

func TestYgggoBridge_BuildLogger(t *testing.T) {
	lg := newYgggoSlogLoggerFromEnv()
	if lg == nil {
		t.Fatal("expected non-nil slog logger from ygggo_log bridge")
	}
}

func TestPool_UseYgggoLoggerFromEnv_NoPanic(t *testing.T) {
	var p Pool
	p.UseYgggoLoggerFromEnv()
	// Verify fields set
	if p.logger == nil {
		t.Fatal("expected logger to be set")
	}
	if !p.loggingEnabled {
		t.Fatal("expected loggingEnabled to be true")
	}
}

func TestSetLoggerOverrides_Default(t *testing.T) {
	var p Pool
	custom := slog.New(slog.NewTextHandler(&bufSink{}, nil))
	p.SetLogger(custom)
	p.EnableLogging(true)
	if p.logger != custom {
		t.Fatal("expected custom logger to be used")
	}
}

type bufSink struct{}

func (*bufSink) Write(p []byte) (int, error) { return len(p), nil }
