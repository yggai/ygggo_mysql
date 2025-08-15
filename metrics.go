package ygggo_mysql

import (
	"context"
	"database/sql"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	metricsInstrumentationName = "github.com/yggai/ygggo_mysql"
)

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool
}

// Metrics holds all the metric instruments
type Metrics struct {
	// Connection metrics
	connectionsActive   metric.Int64UpDownCounter
	connectionsTotal    metric.Int64Counter
	connectionDuration  metric.Float64Histogram

	// Query metrics
	queriesTotal        metric.Int64Counter
	queryDuration       metric.Float64Histogram
	
	// Transaction metrics
	transactionsTotal   metric.Int64Counter
	transactionDuration metric.Float64Histogram
}

var (
	defaultMeter = otel.Meter(metricsInstrumentationName)
)

// EnableMetrics enables or disables metrics collection for this pool
func (p *Pool) EnableMetrics(enabled bool) {
	if p == nil { return }
	p.metricsEnabled = enabled
	if enabled && p.metrics == nil {
		p.initMetrics()
	}
}

// SetMeterProvider sets a custom meter provider for metrics
func (p *Pool) SetMeterProvider(provider metric.MeterProvider) {
	if p == nil { return }
	p.meterProvider = provider
	if p.metricsEnabled {
		p.initMetrics()
	}
}

// initMetrics initializes all metric instruments
func (p *Pool) initMetrics() {
	if p == nil { return }
	
	var meter metric.Meter
	if p.meterProvider != nil {
		meter = p.meterProvider.Meter(metricsInstrumentationName)
	} else {
		meter = defaultMeter
	}
	
	p.metrics = &Metrics{}
	
	// Connection metrics
	p.metrics.connectionsActive, _ = meter.Int64UpDownCounter(
		"ygggo_mysql_connections_active",
		metric.WithDescription("Number of active database connections"),
	)
	
	p.metrics.connectionsTotal, _ = meter.Int64Counter(
		"ygggo_mysql_connections_total",
		metric.WithDescription("Total number of database connections created"),
	)
	
	p.metrics.connectionDuration, _ = meter.Float64Histogram(
		"ygggo_mysql_connection_duration_seconds",
		metric.WithDescription("Duration of database connections"),
		metric.WithUnit("s"),
	)
	
	// Query metrics
	p.metrics.queriesTotal, _ = meter.Int64Counter(
		"ygggo_mysql_queries_total",
		metric.WithDescription("Total number of database queries"),
	)
	
	p.metrics.queryDuration, _ = meter.Float64Histogram(
		"ygggo_mysql_query_duration_seconds",
		metric.WithDescription("Duration of database queries"),
		metric.WithUnit("s"),
	)
	
	// Transaction metrics
	p.metrics.transactionsTotal, _ = meter.Int64Counter(
		"ygggo_mysql_transactions_total",
		metric.WithDescription("Total number of database transactions"),
	)
	
	p.metrics.transactionDuration, _ = meter.Float64Histogram(
		"ygggo_mysql_transaction_duration_seconds",
		metric.WithDescription("Duration of database transactions"),
		metric.WithUnit("s"),
	)
}

// recordConnectionAcquired records when a connection is acquired
func (p *Pool) recordConnectionAcquired(ctx context.Context) {
	if p == nil || !p.metricsEnabled || p.metrics == nil { return }
	
	p.metrics.connectionsActive.Add(ctx, 1)
	p.metrics.connectionsTotal.Add(ctx, 1)
}

// recordConnectionReleased records when a connection is released
func (p *Pool) recordConnectionReleased(ctx context.Context, duration time.Duration) {
	if p == nil || !p.metricsEnabled || p.metrics == nil { return }
	
	p.metrics.connectionsActive.Add(ctx, -1)
	p.metrics.connectionDuration.Record(ctx, duration.Seconds())
}

// recordQuery records query execution metrics
func (p *Pool) recordQuery(ctx context.Context, operation string, duration time.Duration, err error) {
	if p == nil || !p.metricsEnabled || p.metrics == nil { return }
	
	status := "success"
	if err != nil {
		status = "error"
	}
	
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("status", status),
	}
	
	p.metrics.queriesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	p.metrics.queryDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// recordTransaction records transaction execution metrics
func (p *Pool) recordTransaction(ctx context.Context, duration time.Duration, err error) {
	if p == nil || !p.metricsEnabled || p.metrics == nil { return }
	
	status := "success"
	if err != nil {
		status = "error"
	}
	
	attrs := []attribute.KeyValue{
		attribute.String("status", status),
	}
	
	p.metrics.transactionsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	p.metrics.transactionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// instrumentedQueryWithMetrics wraps query execution with tracing, metrics and logging
func (p *Pool) instrumentedQueryWithMetrics(ctx context.Context, conn *sql.Conn, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()

	// Start tracing span but don't modify the context passed to database operations
	var span interface{}
	if p.telemetryEnabled {
		_, span = p.startSpan(ctx, "query", query)
	}

	// Execute query with original context to avoid deadlock
	rs, err := conn.QueryContext(ctx, query, args...)

	// Record duration
	duration := time.Since(start)

	// Log the query
	if p.loggingEnabled {
		p.logQuery(ctx, "query", query, args, duration, err)
	}

	// Record metrics
	if p.metricsEnabled {
		p.recordQuery(ctx, "query", duration, err)
	}

	// Finish tracing span
	if p.telemetryEnabled && span != nil {
		p.finishSpan(span, err)
	}

	return rs, err
}

// instrumentedExecWithMetrics wraps exec execution with tracing, metrics and logging
func (p *Pool) instrumentedExecWithMetrics(ctx context.Context, conn *sql.Conn, query string, args ...any) (sql.Result, error) {
	start := time.Now()

	// Start tracing span but don't modify the context passed to database operations
	var span interface{}
	if p.telemetryEnabled {
		_, span = p.startSpan(ctx, "exec", query)
	}

	// Execute with original context to avoid deadlock
	result, err := conn.ExecContext(ctx, query, args...)

	// Record duration
	duration := time.Since(start)

	// Log the execution
	if p.loggingEnabled {
		p.logQuery(ctx, "exec", query, args, duration, err)
	}

	// Record metrics
	if p.metricsEnabled {
		p.recordQuery(ctx, "exec", duration, err)
	}

	// Finish tracing span
	if p.telemetryEnabled && span != nil {
		p.finishSpan(span, err)
	}

	return result, err
}
