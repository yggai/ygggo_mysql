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
		// BulkInsert: expect multi-values insert
		mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\)`).
			WithArgs(1, "x", 2, "y").
			WillReturnResult(ygggo_mysql.NewResult(0, 2))

		// Upsert: ON DUPLICATE KEY UPDATE b=VALUES(b)
		mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\) ON DUPLICATE KEY UPDATE b=VALUES\(b\)`).
			WithArgs(1, "x", 2, "y").
			WillReturnResult(ygggo_mysql.NewResult(0, 2))
	}

	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
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

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	}
	fmt.Println("ygggo_mysql example: bulk & upsert")
}

