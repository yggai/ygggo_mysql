package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yggai/ygggo_mysql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	// Setup OpenTelemetry
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil { log.Fatalf("failed to create exporter: %v", err) }

	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)

	// Create pool with mock
	pool, mock, err := ygggo_mysql.NewPoolWithMock(ctx, ygggo_mysql.Config{}, true)
	if err != nil { log.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	// Enable telemetry
	pool.EnableTelemetry(true)

	// Setup mock expectations
	if mock != nil {
		rows := ygggo_mysql.NewRows([]string{"id", "name"})
		rows = ygggo_mysql.AddRow(rows, 1, "Alice")
		rows = ygggo_mysql.AddRow(rows, 2, "Bob")
		mock.ExpectQuery(`SELECT id, name FROM users`).WillReturnRows(rows)
	}

	// Use direct connection to avoid WithConn issues for now
	conn, err := pool.Acquire(ctx)
	if err != nil { log.Fatalf("Acquire: %v", err) }

	rs, err := conn.Query(ctx, "SELECT id, name FROM users")
	if err != nil { log.Fatalf("Query: %v", err) }
	defer rs.Close()

	count := 0
	for rs.Next() { count++ }
	fmt.Printf("Query returned %d rows\n", count)

	err = conn.Close()
	if err != nil { log.Fatalf("Close: %v", err) }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			log.Fatalf("mock expectations not met: %v", err)
		}
	}

	fmt.Println("ygggo_mysql example: telemetry integration")
	fmt.Println("Check the output above for OpenTelemetry spans!")
}
