package ygggo_mysql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMetrics_NoMock_EnableDisable(t *testing.T) {
	// Create a pool without mock to avoid potential deadlock issues
	pool := &Pool{
		metricsEnabled: false,
	}

	// Test enabling metrics
	pool.EnableMetrics(true)
	if !pool.metricsEnabled {
		t.Fatalf("metrics should be enabled")
	}

	// Test disabling metrics
	pool.EnableMetrics(false)
	if pool.metricsEnabled {
		t.Fatalf("metrics should be disabled")
	}
}

func TestMetrics_NoMock_InitMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	
	pool := &Pool{
		metricsEnabled: false,
	}

	pool.EnableMetrics(true)
	pool.SetMeterProvider(provider)

	// Check that metrics were initialized
	if pool.metrics == nil {
		t.Fatalf("metrics should be initialized")
	}

	// Check that all expected metric instruments exist
	if pool.metrics.connectionsActive == nil {
		t.Fatalf("connectionsActive metric should be initialized")
	}
	if pool.metrics.connectionsTotal == nil {
		t.Fatalf("connectionsTotal metric should be initialized")
	}
	if pool.metrics.queriesTotal == nil {
		t.Fatalf("queriesTotal metric should be initialized")
	}
	if pool.metrics.queryDuration == nil {
		t.Fatalf("queryDuration metric should be initialized")
	}
}

func TestMetrics_NoMock_RecordOperations(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	
	pool := &Pool{
		metricsEnabled: true,
	}
	pool.SetMeterProvider(provider)

	ctx := context.Background()

	// Record some operations
	pool.recordConnectionAcquired(ctx)
	pool.recordQuery(ctx, "query", 100*time.Microsecond, nil)
	pool.recordQuery(ctx, "exec", 50*time.Microsecond, nil)

	// Collect metrics
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, &rm)
	if err != nil { t.Fatalf("Collect: %v", err) }

	// Verify we have metrics
	if len(rm.ScopeMetrics) == 0 {
		t.Fatalf("no metrics collected")
	}

	// Check for expected metrics
	foundMetrics := make(map[string]bool)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			foundMetrics[m.Name] = true
			t.Logf("Found metric: %s", m.Name)
		}
	}

	expectedMetrics := []string{
		"ygggo_mysql_connections_active",
		"ygggo_mysql_connections_total",
		"ygggo_mysql_queries_total",
		"ygggo_mysql_query_duration_seconds",
	}

	for _, expected := range expectedMetrics {
		if !foundMetrics[expected] {
			t.Fatalf("missing metric: %s", expected)
		}
	}
}

func TestMetrics_NoMock_ErrorRecording(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	
	pool := &Pool{
		metricsEnabled: true,
	}
	pool.SetMeterProvider(provider)

	ctx := context.Background()

	// Record an error operation
	pool.recordQuery(ctx, "query", 100*time.Microsecond, sqlmock.ErrCancelled)

	// Collect metrics
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, &rm)
	if err != nil { t.Fatalf("Collect: %v", err) }

	// Look for error status in metrics
	foundError := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ygggo_mysql_queries_total" {
				if data, ok := m.Data.(metricdata.Sum[int64]); ok {
					for _, dp := range data.DataPoints {
						for _, attr := range dp.Attributes.ToSlice() {
							if attr.Key == "status" && attr.Value.AsString() == "error" {
								foundError = true
								break
							}
						}
					}
				}
			}
		}
	}

	if !foundError {
		t.Fatalf("error status not recorded in metrics")
	}
}
