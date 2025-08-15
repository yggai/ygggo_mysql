package ygggo_mysql

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

func TestLogging_EnableDisable(t *testing.T) {
	pool := &Pool{
		loggingEnabled: false,
	}

	// Test enabling logging
	pool.EnableLogging(true)
	if !pool.loggingEnabled {
		t.Fatalf("logging should be enabled")
	}

	// Test disabling logging
	pool.EnableLogging(false)
	if pool.loggingEnabled {
		t.Fatalf("logging should be disabled")
	}
}

func TestLogging_SetLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool := &Pool{}
	pool.SetLogger(logger)

	if pool.logger != logger {
		t.Fatalf("logger should be set")
	}
}

func TestLogging_QueryLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool := &Pool{
		loggingEnabled: true,
		logger:         logger,
	}

	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"
	args := []any{123}
	duration := 50 * time.Millisecond
	
	pool.logQuery(ctx, "query", query, args, duration, nil)

	// Parse the logged JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	// Verify log fields
	if logEntry["msg"] != "database query executed" {
		t.Fatalf("expected msg 'database query executed', got %v", logEntry["msg"])
	}
	if logEntry["operation"] != "query" {
		t.Fatalf("expected operation 'query', got %v", logEntry["operation"])
	}
	if logEntry["query"] != query {
		t.Fatalf("expected query %s, got %v", query, logEntry["query"])
	}
	if logEntry["duration_ms"] != float64(50) {
		t.Fatalf("expected duration_ms 50, got %v", logEntry["duration_ms"])
	}
	if logEntry["status"] != "success" {
		t.Fatalf("expected status 'success', got %v", logEntry["status"])
	}
}

func TestLogging_QueryLoggingWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool := &Pool{
		loggingEnabled: true,
		logger:         logger,
	}

	ctx := context.Background()
	query := "SELECT * FROM nonexistent"
	duration := 10 * time.Millisecond
	err := &mysql.MySQLError{Number: 1146, Message: "Table doesn't exist"}
	
	pool.logQuery(ctx, "query", query, nil, duration, err)

	// Parse the logged JSON
	var logEntry map[string]interface{}
	if parseErr := json.Unmarshal(buf.Bytes(), &logEntry); parseErr != nil {
		t.Fatalf("failed to parse log JSON: %v", parseErr)
	}

	// Verify error logging
	if logEntry["status"] != "error" {
		t.Fatalf("expected status 'error', got %v", logEntry["status"])
	}
	errorMsg := logEntry["error"].(string)
	if !strings.Contains(errorMsg, "Table doesn't exist") {
		t.Fatalf("expected error message to contain 'Table doesn't exist', got %v", errorMsg)
	}
	if logEntry["error_code"] != float64(1146) {
		t.Fatalf("expected error_code 1146, got %v", logEntry["error_code"])
	}
}

func TestLogging_ConnectionLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool := &Pool{
		loggingEnabled: true,
		logger:         logger,
	}

	ctx := context.Background()
	duration := 100 * time.Millisecond
	
	pool.logConnection(ctx, "acquired", duration, nil)

	// Parse the logged JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	// Verify connection log fields
	if logEntry["msg"] != "database connection event" {
		t.Fatalf("expected msg 'database connection event', got %v", logEntry["msg"])
	}
	if logEntry["event"] != "acquired" {
		t.Fatalf("expected event 'acquired', got %v", logEntry["event"])
	}
	if logEntry["duration_ms"] != float64(100) {
		t.Fatalf("expected duration_ms 100, got %v", logEntry["duration_ms"])
	}
}

func TestLogging_TransactionLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool := &Pool{
		loggingEnabled: true,
		logger:         logger,
	}

	ctx := context.Background()
	duration := 200 * time.Millisecond
	
	pool.logTransaction(ctx, "commit", duration, nil)

	// Parse the logged JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	// Verify transaction log fields
	if logEntry["msg"] != "database transaction event" {
		t.Fatalf("expected msg 'database transaction event', got %v", logEntry["msg"])
	}
	if logEntry["event"] != "commit" {
		t.Fatalf("expected event 'commit', got %v", logEntry["event"])
	}
	if logEntry["duration_ms"] != float64(200) {
		t.Fatalf("expected duration_ms 200, got %v", logEntry["duration_ms"])
	}
}

func TestLogging_SlowQueryLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	pool := &Pool{
		loggingEnabled:    true,
		logger:           logger,
		slowQueryThreshold: 100 * time.Millisecond,
	}

	ctx := context.Background()
	query := "SELECT * FROM large_table"
	duration := 150 * time.Millisecond // Exceeds threshold
	
	pool.logQuery(ctx, "query", query, nil, duration, nil)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "slow query detected") {
		t.Fatalf("expected slow query warning, got: %s", logOutput)
	}
}

func TestLogging_DisabledLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool := &Pool{
		loggingEnabled: false, // Disabled
		logger:         logger,
	}

	ctx := context.Background()
	pool.logQuery(ctx, "query", "SELECT 1", nil, 10*time.Millisecond, nil)

	// Should not log anything
	if buf.Len() > 0 {
		t.Fatalf("expected no logging when disabled, got: %s", buf.String())
	}
}
