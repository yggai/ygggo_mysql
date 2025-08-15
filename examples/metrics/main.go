package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yggai/ygggo_mysql"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func main() {
	ctx := context.Background()

	// Setup OpenTelemetry Metrics
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	// Create pool with mock
	pool, mock, err := ygggo_mysql.NewPoolWithMock(ctx, ygggo_mysql.Config{}, true)
	if err != nil { log.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	// Enable metrics
	pool.EnableMetrics(true)
	pool.SetMeterProvider(provider)

	// Setup mock expectations
	if mock != nil {
		rows := ygggo_mysql.NewRows([]string{"id", "name"})
		rows = ygggo_mysql.AddRow(rows, 1, "Alice")
		rows = ygggo_mysql.AddRow(rows, 2, "Bob")
		mock.ExpectQuery(`SELECT id, name FROM users`).WillReturnRows(rows)
		
		mock.ExpectExec(`INSERT INTO users \(name\) VALUES \(\?\)`).WithArgs("Charlie").
			WillReturnResult(ygggo_mysql.NewResult(3, 1))
	}

	// Use direct connection to avoid WithConn issues for now
	conn, err := pool.Acquire(ctx)
	if err != nil { log.Fatalf("Acquire: %v", err) }

	// Execute query
	rs, err := conn.Query(ctx, "SELECT id, name FROM users")
	if err != nil { log.Fatalf("Query: %v", err) }
	defer rs.Close()

	count := 0
	for rs.Next() { count++ }
	fmt.Printf("Query returned %d rows\n", count)

	// Execute insert
	result, err := conn.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Charlie")
	if err != nil { log.Fatalf("Exec: %v", err) }
	
	affected, _ := result.RowsAffected()
	fmt.Printf("Insert affected %d rows\n", affected)

	err = conn.Close()
	if err != nil { log.Fatalf("Close: %v", err) }

	// Collect and display metrics
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(ctx, &rm)
	if err != nil { log.Fatalf("Collect: %v", err) }

	fmt.Println("\n=== Metrics ===")
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			fmt.Printf("Metric: %s\n", m.Name)
			fmt.Printf("  Description: %s\n", m.Description)
			
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					fmt.Printf("  Value: %d, Attributes: %v\n", dp.Value, dp.Attributes)
				}
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					fmt.Printf("  Count: %d, Sum: %f, Attributes: %v\n", dp.Count, dp.Sum, dp.Attributes)
				}
			}
			fmt.Println()
		}
	}

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			log.Fatalf("mock expectations not met: %v", err)
		}
	}

	fmt.Println("ygggo_mysql example: metrics integration")
}
