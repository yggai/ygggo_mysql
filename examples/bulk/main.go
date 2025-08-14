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

	dsn := "example_sqlmock_bulk"
	db, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil { log.Fatalf("sqlmock.NewWithDSN: %v", err) }
	defer db.Close()
	mock.ExpectPing()

	// BulkInsert: expect multi-values insert
	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\)`).
		WithArgs(1, "x", 2, "y").
		WillReturnResult(sqlmock.NewResult(0, 2))

	// Upsert: ON DUPLICATE KEY UPDATE b=VALUES(b)
	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\) ON DUPLICATE KEY UPDATE b=VALUES\(b\)`).
		WithArgs(1, "x", 2, "y").
		WillReturnResult(sqlmock.NewResult(0, 2))

	pool, err := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{Driver: "sqlmock", DSN: dsn})
	if err != nil { log.Fatalf("NewPool: %v", err) }
	defer pool.Close()

	err = pool.WithConn(ctx, func(c *ygggo_mysql.Conn) error {
		rows := [][]any{{1, "x"}, {2, "y"}}
		res, err := c.BulkInsert(ctx, "t", []string{"a", "b"}, rows)
		if err != nil { return err }
		aff, _ := res.RowsAffected()
		fmt.Println("bulk affected:", aff)

		res, err = c.InsertOnDuplicate(ctx, "t", []string{"a", "b"}, rows, []string{"b"})
		if err != nil { return err }
		aff, _ = res.RowsAffected()
		fmt.Println("upsert affected:", aff)
		return nil
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	fmt.Println("ygggo_mysql example: bulk & upsert")
}

