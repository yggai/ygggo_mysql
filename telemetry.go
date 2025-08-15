package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/yggai/ygggo_mysql"
	instrumentationVersion = "v0.1.0"
)

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	Enabled bool
	ServiceName string
	ServiceVersion string
}

var (
	tracer = otel.Tracer(instrumentationName, trace.WithInstrumentationVersion(instrumentationVersion))
)

// EnableTelemetry enables or disables OpenTelemetry tracing for this pool
func (p *Pool) EnableTelemetry(enabled bool) {
	if p == nil { return }
	p.telemetryEnabled = enabled
}

// startSpan creates a new span with common database attributes
func (p *Pool) startSpan(ctx context.Context, operation string, query string) (context.Context, trace.Span) {
	if p == nil || !p.telemetryEnabled {
		return ctx, trace.SpanFromContext(ctx)
	}

	spanName := fmt.Sprintf("ygggo_mysql.%s", operation)
	ctx, span := tracer.Start(ctx, spanName)

	// Set common attributes
	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", operation),
	)

	if query != "" {
		span.SetAttributes(attribute.String("db.statement", query))
	}

	return ctx, span
}

// finishSpan completes a span with error handling
func (p *Pool) finishSpan(span trace.Span, err error) {
	if p == nil || !p.telemetryEnabled {
		return
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
}

// instrumentedQuery wraps query execution with tracing
func (p *Pool) instrumentedQuery(ctx context.Context, conn *sql.Conn, query string, args ...any) (*sql.Rows, error) {
	spanCtx, span := p.startSpan(ctx, "query", query)
	rs, err := conn.QueryContext(spanCtx, query, args...)
	p.finishSpan(span, err)
	return rs, err
}

// instrumentedExec wraps exec execution with tracing
func (p *Pool) instrumentedExec(ctx context.Context, conn *sql.Conn, query string, args ...any) (sql.Result, error) {
	spanCtx, span := p.startSpan(ctx, "exec", query)
	result, err := conn.ExecContext(spanCtx, query, args...)
	p.finishSpan(span, err)
	return result, err
}


