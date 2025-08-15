package ygggo_mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTelemetry_QuerySpan(t *testing.T) {
	// Setup tracing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}
	p.EnableTelemetry(true)

	mock.ExpectQuery(`SELECT \* FROM users WHERE id=\?`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Alice"))

	ctx := context.Background()
	err = p.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT * FROM users WHERE id=?", 1)
		if err != nil { return err }
		defer rs.Close()
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }

	// Verify spans
	spans := exporter.GetSpans()
	if len(spans) != 1 { t.Fatalf("expected 1 span, got %d", len(spans)) }
	
	span := spans[0]
	if span.Name != "ygggo_mysql.query" { t.Fatalf("span name=%s", span.Name) }
	if span.Status.Code != codes.Ok { t.Fatalf("span status=%v", span.Status) }
	
	// Check attributes
	attrs := span.Attributes
	expectedAttrs := map[string]string{
		"db.system":     "mysql",
		"db.operation":  "query",
		"db.statement":  "SELECT * FROM users WHERE id=?",
	}
	for key, expected := range expectedAttrs {
		found := false
		for _, attr := range attrs {
			if string(attr.Key) == key && attr.Value.AsString() == expected {
				found = true
				break
			}
		}
		if !found { t.Fatalf("missing attribute %s=%v", key, expected) }
	}
}

func TestTelemetry_TransactionSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}
	p.EnableTelemetry(true)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO users \(name\) VALUES \(\?\)`).WithArgs("Bob").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = p.WithinTx(ctx, func(tx DatabaseTx) error {
		_, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Bob")
		return err
	})
	if err != nil { t.Fatalf("WithinTx: %v", err) }

	spans := exporter.GetSpans()
	if len(spans) != 2 { t.Fatalf("expected 2 spans, got %d", len(spans)) }

	// Should have tx span and exec span
	var hasTxSpan, hasExecSpan bool
	for _, span := range spans {
		if span.Name == "ygggo_mysql.transaction" {
			hasTxSpan = true
		} else if span.Name == "ygggo_mysql.exec" {
			hasExecSpan = true
		}
	}

	if !hasTxSpan { t.Fatalf("missing transaction span") }
	if !hasExecSpan { t.Fatalf("missing exec span") }
}

func TestTelemetry_ErrorSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}
	p.EnableTelemetry(true)

	mock.ExpectQuery(`SELECT \* FROM nonexistent`).
		WillReturnError(sqlmock.ErrCancelled)

	ctx := context.Background()
	err = p.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Query(ctx, "SELECT * FROM nonexistent")
		return err
	})
	if err == nil { t.Fatalf("expected error") }

	spans := exporter.GetSpans()
	if len(spans) != 1 { t.Fatalf("expected 1 span, got %d", len(spans)) }
	
	span := spans[0]
	if span.Status.Code != codes.Error { t.Fatalf("expected error status, got %v", span.Status) }
}

func TestTelemetry_Disabled(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}
	// Don't enable telemetry

	mock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))

	ctx := context.Background()
	err = p.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Query(ctx, "SELECT 1")
		return err
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }

	spans := exporter.GetSpans()
	if len(spans) != 0 { t.Fatalf("expected 0 spans when disabled, got %d", len(spans)) }
}
