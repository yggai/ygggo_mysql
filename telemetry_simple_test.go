package ygggo_mysql

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTelemetry_EnableDisable(t *testing.T) {
	pool, mock, err := NewPoolWithMock(context.Background(), Config{}, true)
	if err != nil { t.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	// Test enabling telemetry
	pool.EnableTelemetry(true)
	if !pool.telemetryEnabled {
		t.Fatalf("telemetry should be enabled")
	}

	// Test disabling telemetry
	pool.EnableTelemetry(false)
	if pool.telemetryEnabled {
		t.Fatalf("telemetry should be disabled")
	}

	// Ensure mock expectations are met
	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock expectations not met: %v", err)
		}
	}
}

func TestTelemetry_BasicSpan(t *testing.T) {
	// Setup tracing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	pool, mock, err := NewPoolWithMock(context.Background(), Config{}, true)
	if err != nil { t.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	pool.EnableTelemetry(true)

	if mock != nil {
		rows := NewRows([]string{"c"})
		rows = AddRow(rows, 1)
		mock.ExpectQuery(`SELECT 1`).WillReturnRows(rows)
	}

	ctx := context.Background()
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Query(ctx, "SELECT 1")
		return err
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }

	// Verify spans were created
	spans := exporter.GetSpans()
	if len(spans) == 0 { t.Fatalf("expected at least 1 span, got 0") }

	// Find query span
	var querySpan tracetest.SpanStub
	for _, span := range spans {
		if span.Name == "ygggo_mysql.query" {
			querySpan = span
			break
		}
	}

	if querySpan.Name == "" { t.Fatalf("missing query span") }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock expectations not met: %v", err)
		}
	}
}
