package ygggo_mysql

import (
	"context"
	"testing"
)

func TestNewMockPool_CreatesPoolWithMockDB(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil { t.Fatalf("NewMockPool: %v", err) }
	defer pool.Close()

	mock.ExpectPing()
	if err := pool.Ping(ctx); err != nil { t.Fatalf("Ping: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestNewMockPool_WithConnAndQuery(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil { t.Fatalf("NewMockPool: %v", err) }
	defer pool.Close()

	rows := NewRows([]string{"c"})
	rows = AddRow(rows, 1)
	mock.ExpectQuery(`SELECT 1`).WillReturnRows(rows)
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT 1")
		if err != nil { return err }
		defer rs.Close()
		count := 0
		for rs.Next() { count++ }
		if count != 1 { t.Fatalf("count=%d", count) }
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestNewMockPool_WithinTx(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil { t.Fatalf("NewMockPool: %v", err) }
	defer pool.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(1).WillReturnResult(NewResult(1,1))
	mock.ExpectCommit()

	err = pool.WithinTx(ctx, func(tx DatabaseTx) error {
		_, err := tx.Exec(ctx, "INSERT INTO t(a) VALUES(?)", 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestNewMockPool_BulkInsert(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil { t.Fatalf("NewMockPool: %v", err) }
	defer pool.Close()

	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\)`).
		WithArgs(1, "x", 2, "y").
		WillReturnResult(NewResult(0, 2))

	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		rows := [][]any{{1, "x"}, {2, "y"}}
		_, err := c.BulkInsert(ctx, "t", []string{"a", "b"}, rows)
		return err
	})
	if err != nil { t.Fatalf("BulkInsert: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestNewMockPool_NamedExec(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil { t.Fatalf("NewMockPool: %v", err) }
	defer pool.Close()

	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\)`).WithArgs(1, "x").WillReturnResult(NewResult(1,1))

	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.NamedExec(ctx, "INSERT INTO t (a,b) VALUES (:a,:b)", map[string]any{"a": 1, "b": "x"})
		return err
	})
	if err != nil { t.Fatalf("NamedExec: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}
