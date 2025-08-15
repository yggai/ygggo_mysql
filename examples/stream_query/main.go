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
		rows := ygggo_mysql.NewRows([]string{"id","name"})
		rows = ygggo_mysql.AddRow(rows, 1, "a")
		rows = ygggo_mysql.AddRow(rows, 2, "b")
		mock.ExpectQuery(`SELECT id,name FROM t`).WillReturnRows(rows)
	}

	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		return c.QueryStream(ctx, "SELECT id,name FROM t", func(_ []any) error {
			return nil
		})
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	}
	fmt.Println("ygggo_mysql example: stream_query")
}

