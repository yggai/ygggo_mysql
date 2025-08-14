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

	dsn := "example_sqlmock_tx_retry"
	db, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil { log.Fatalf("sqlmock.NewWithDSN: %v", err) }
	defer db.Close()
	mock.ExpectPing()
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(1).WillReturnResult(sqlmock.NewResult(1,1))
	mock.ExpectCommit()

	pool, err := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{Driver: "sqlmock", DSN: dsn})
	if err != nil { log.Fatalf("NewPool: %v", err) }
	defer pool.Close()

	err = pool.WithinTx(ctx, func(tx *ygggo_mysql.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO t(a) VALUES(?)", 1)
		return err
	})
	if err != nil { log.Fatalf("WithinTx err: %v", err) }

	if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	fmt.Println("ygggo_mysql example: tx_retry")
}

