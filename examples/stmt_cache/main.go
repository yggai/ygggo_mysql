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
		// Expect one Prepare for the INSERT statement, then two Execs reusing it
		prep := mock.ExpectPrepare(`INSERT INTO t\(a\) VALUES\(\?\)`)
		prep.ExpectExec().WithArgs(1).WillReturnResult(ygggo_mysql.NewResult(1, 1))
		prep.ExpectExec().WithArgs(2).WillReturnResult(ygggo_mysql.NewResult(2, 1))

		// Also show QueryCached reuse (one prepare, one query)
		rows := ygggo_mysql.NewRows([]string{"id", "name"})
		rows = ygggo_mysql.AddRow(rows, 1, "a")
		rows = ygggo_mysql.AddRow(rows, 2, "b")
		mock.ExpectPrepare(`SELECT id,name FROM t`).ExpectQuery().WillReturnRows(rows)
	}

	// Use a single connection and enable LRU stmt cache (capacity 2)
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		c.EnableStmtCache(2)
		if _, err := c.ExecCached(ctx, "INSERT INTO t(a) VALUES(?)", 1); err != nil { return err }
		if _, err := c.ExecCached(ctx, "INSERT INTO t(a) VALUES(?)", 2); err != nil { return err }

		rs, err := c.QueryCached(ctx, "SELECT id,name FROM t")
		if err != nil { return err }
		defer rs.Close()
		count := 0
		for rs.Next() { count++ }
		fmt.Println("query rows:", count)
		return rs.Err()
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	}
	fmt.Println("ygggo_mysql example: stmt_cache (prepared once, reused)")
}

