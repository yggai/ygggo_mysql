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

	// Use sqlmock as an in-memory MySQL to demonstrate StmtCache behavior
	dsn := "example_sqlmock_stmt_cache"
	db, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil { log.Fatalf("sqlmock.NewWithDSN: %v", err) }
	defer db.Close()
	mock.ExpectPing()

	// Expect one Prepare for the INSERT statement, then two Execs reusing it
	prep := mock.ExpectPrepare(`INSERT INTO t\(a\) VALUES\(\?\)`)
	prep.ExpectExec().WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().WithArgs(2).WillReturnResult(sqlmock.NewResult(2, 1))

	// Also show QueryCached reuse (one prepare, one query)
	mock.ExpectPrepare(`SELECT id,name FROM t`).
		ExpectQuery().
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "a").AddRow(2, "b"))

	pool, err := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{Driver: "sqlmock", DSN: dsn})
	if err != nil { log.Fatalf("NewPool: %v", err) }
	defer pool.Close()

	// Use a single connection and enable LRU stmt cache (capacity 2)
	err = pool.WithConn(ctx, func(c *ygggo_mysql.Conn) error {
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

	if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	fmt.Println("ygggo_mysql example: stmt_cache (prepared once, reused)")
}

