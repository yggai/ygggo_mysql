package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yggai/ygggo_mysql"
)

func main() {
	ctx := context.Background()

	pool, mock, err := ygggo_mysql.NewPoolWithMock(ctx, ygggo_mysql.Config{}, true)
	if err != nil { log.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	if mock != nil {
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(1).WillReturnResult(ygggo_mysql.NewResult(1,1))
		mock.ExpectCommit()
	}

	err = pool.WithinTx(ctx, func(tx *ygggo_mysql.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO t(a) VALUES(?)", 1)
		return err
	})
	if err != nil { log.Fatalf("WithinTx err: %v", err) }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	}
	fmt.Println("ygggo_mysql example: tx_retry")
}

