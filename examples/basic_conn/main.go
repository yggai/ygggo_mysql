package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yggai/ygggo_mysql"
)

func main() {
	ctx := context.Background()

	// Use sqlmock as in-memory DB, register a DSN and expect a Ping
	dsn := "example_sqlmock_basic_conn"
	_, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil { log.Fatalf("sqlmock.NewWithDSN: %v", err) }
	mock.ExpectPing()

	pool, err := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{Driver: "sqlmock", DSN: dsn})
	if err != nil { log.Fatalf("NewPool: %v", err) }
	defer pool.Close()

	if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	fmt.Println("ygggo_mysql example: basic_conn", ygggo_mysql.Version())
}

