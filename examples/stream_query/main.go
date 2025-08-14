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

	dsn := "example_sqlmock_stream"
	db, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil { log.Fatalf("sqlmock.NewWithDSN: %v", err) }
	defer db.Close()
	mock.ExpectPing()
	mock.ExpectQuery(`SELECT id,name FROM t`).
		WillReturnRows(sqlmock.NewRows([]string{"id","name"}).AddRow(1, "a").AddRow(2, "b"))

	pool, err := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{Driver: "sqlmock", DSN: dsn})
	if err != nil { log.Fatalf("NewPool: %v", err) }
	defer pool.Close()

	err = pool.WithConn(ctx, func(c *ygggo_mysql.Conn) error {
		return c.QueryStream(ctx, "SELECT id,name FROM t", func(_ []any) error {
			return nil
		})
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	fmt.Println("ygggo_mysql example: stream_query")
}

