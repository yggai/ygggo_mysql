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

	// Setup in-memory (sqlmock) database and expectations
	dsn := "example_sqlmock_basic_query"
	db, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil { log.Fatalf("sqlmock.NewWithDSN: %v", err) }
	defer db.Close()
	mock.ExpectPing()
	mock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))

	pool, err := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{Driver: "sqlmock", DSN: dsn})
	if err != nil { log.Fatalf("NewPool: %v", err) }
	defer pool.Close()

	err = pool.WithConn(ctx, func(c *ygggo_mysql.Conn) error {
		rs, err := c.Query(ctx, "SELECT 1")
		if err != nil { return err }
		defer rs.Close()
		var count int
		for rs.Next() { count++ }
		fmt.Println("rows:", count)
		return nil
	})
	if err != nil { log.Fatalf("WithConn err: %v", err) }

	if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	fmt.Println("ygggo_mysql example: basic_query")
}

