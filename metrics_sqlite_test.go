package ygggo_mysql

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMetrics_SQLite_BasicFunctionality(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	pool, err := NewSQLiteTestPool(context.Background())
	if err != nil { t.Fatalf("NewSQLiteTestPool: %v", err) }
	defer pool.Close()

	pool.EnableMetrics(true)
	pool.SetMeterProvider(provider)

	// Create test table and perform operations
	ctx := context.Background()
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)`)
		if err != nil { return err }
		_, err = c.Exec(ctx, `INSERT INTO test (value) VALUES (?)`, "test_value")
		if err != nil { return err }
		rs, err := c.Query(ctx, `SELECT id, value FROM test`)
		if err != nil { return err }
		defer rs.Close()
		
		for rs.Next() {
			var id int
			var value string
			err := rs.Scan(&id, &value)
			if err != nil { return err }
		}
		return rs.Err()
	})
	if err != nil { t.Fatalf("Operations failed: %v", err) }

	// Collect metrics
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(ctx, &rm)
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

func TestMetrics_SQLite_ErrorRecording(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	pool, err := NewSQLiteTestPool(context.Background())
	if err != nil { t.Fatalf("NewSQLiteTestPool: %v", err) }
	defer pool.Close()

	pool.EnableMetrics(true)
	pool.SetMeterProvider(provider)

	ctx := context.Background()
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		// This should fail - table doesn't exist
		_, err := c.Exec(ctx, "INSERT INTO nonexistent_table VALUES (1)")
		return err
	})
	if err == nil { t.Fatalf("expected error") }

	// Collect metrics
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(ctx, &rm)
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

func TestMetrics_SQLite_TransactionMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	pool, err := NewSQLiteTestPool(context.Background())
	if err != nil { t.Fatalf("NewSQLiteTestPool: %v", err) }
	defer pool.Close()

	pool.EnableMetrics(true)
	pool.SetMeterProvider(provider)

	// Create test table
	ctx := context.Background()
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER)`)
		if err != nil { return err }
		_, err = c.Exec(ctx, `INSERT INTO accounts (id, balance) VALUES (1, 1000)`)
		return err
	})
	if err != nil { t.Fatalf("Setup failed: %v", err) }

	// Execute transaction
	err = pool.WithinTx(ctx, func(tx DatabaseTx) error {
		_, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - 100 WHERE id = 1")
		return err
	})
	if err != nil { t.Fatalf("Transaction failed: %v", err) }

	// Collect metrics
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(ctx, &rm)
	if err != nil { t.Fatalf("Collect: %v", err) }

	// Look for transaction metrics
	foundTxMetric := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ygggo_mysql_transactions_total" {
				foundTxMetric = true
				break
			}
		}
	}

	if !foundTxMetric {
		t.Fatalf("transaction metrics not found")
	}
}
