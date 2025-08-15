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
		rows := ygggo_mysql.NewRows([]string{"c"})
		rows = ygggo_mysql.AddRow(rows, 1)
		mock.ExpectQuery(`SELECT 1`).WillReturnRows(rows)
	}

	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT 1")
		if err != nil { return err }
		defer rs.Close()
		var count int
		for rs.Next() { count++ }
		fmt.Println("rows:", count)
		return nil
	})
	if err != nil { log.Fatalf("WithConn err: %v", err) }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	}
	fmt.Println("ygggo_mysql example: basic_query")
}

