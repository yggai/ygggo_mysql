package ygggo_mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStmtCache_PerConn_CachesPrepare(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	prep := mock.ExpectPrepare(`INSERT INTO t\(a\) VALUES\(\?\)`) // prepare once, reuse
	prep.ExpectExec().WithArgs(1).WillReturnResult(sqlmock.NewResult(1,1))
	prep.ExpectExec().WithArgs(2).WillReturnResult(sqlmock.NewResult(1,1))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		c.EnableStmtCache(2)
		if _, err := c.ExecCached(context.Background(), "INSERT INTO t(a) VALUES(?)", 1); err != nil { return err }
		if _, err := c.ExecCached(context.Background(), "INSERT INTO t(a) VALUES(?)", 2); err != nil { return err }
		return nil
	})
	if err != nil { t.Fatalf("WithConn err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestStmtCache_LRUEvicts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	// capacity 1: Q1 -> Q2 -> Q1 (evicted) => 2 prepares for Q1, 1 for Q2
	mock.ExpectPrepare(`SELECT 1`).ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectPrepare(`SELECT 2`).ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(2))
	mock.ExpectPrepare(`SELECT 1`).ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		c.EnableStmtCache(1)
		rs, err := c.QueryCached(context.Background(), "SELECT 1")
		if err != nil { return err }
		rs.Close()
		rs, err = c.QueryCached(context.Background(), "SELECT 2")
		if err != nil { return err }
		rs.Close()
		rs, err = c.QueryCached(context.Background(), "SELECT 1")
		if err != nil { return err }
		rs.Close()
		return nil
	})
	if err != nil { t.Fatalf("WithConn err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestStmtCache_PerConn_Isolated(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	// Two connections -> expect two prepares for same SQL
	mock.ExpectPrepare(`UPDATE t SET a=\?`).ExpectExec().WithArgs(1).WillReturnResult(sqlmock.NewResult(0,1))
	mock.ExpectPrepare(`UPDATE t SET a=\?`).ExpectExec().WithArgs(2).WillReturnResult(sqlmock.NewResult(0,1))

	ctx := context.Background()
	err = p.WithConn(ctx, func(c1 *Conn) error {
		c1.EnableStmtCache(2)
		_, err := c1.ExecCached(ctx, "UPDATE t SET a=?", 1)
		return err
	})
	if err != nil { t.Fatalf("c1 err: %v", err) }
	err = p.WithConn(ctx, func(c2 *Conn) error {
		c2.EnableStmtCache(2)
		_, err := c2.ExecCached(ctx, "UPDATE t SET a=?", 2)
		return err
	})
	if err != nil { t.Fatalf("c2 err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

