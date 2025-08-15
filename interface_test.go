package ygggo_mysql

import (
	"context"
	"testing"
)

func TestPoolInterface_MockImplementation(t *testing.T) {
	ctx := context.Background()
	
	// Test that mock pool implements DatabasePool interface
	var pool DatabasePool
	mockPool, mock, err := NewPoolWithMock(ctx, Config{}, true)
	if err != nil { t.Fatalf("NewPoolWithMock: %v", err) }
	pool = mockPool // This should compile if interface is implemented
	defer pool.Close()

	if mock != nil {
		mock.ExpectPing()
	}
	if err := pool.Ping(ctx); err != nil { t.Fatalf("Ping: %v", err) }
	
	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("expectations: %v", err) }
	}
}

func TestPoolInterface_RealImplementation(t *testing.T) {
	ctx := context.Background()
	
	// Test that real pool implements DatabasePool interface
	var pool DatabasePool
	realPool, _, err := NewPoolWithMock(ctx, Config{
		Host: "localhost",
		Port: 3306,
		Username: "test",
		Password: "test",
		Database: "test",
	}, false)
	if err != nil {
		t.Skip("Skipping real DB test - no connection available")
	}
	pool = realPool // This should compile if interface is implemented
	defer pool.Close()
}

func TestConnInterface_MockImplementation(t *testing.T) {
	ctx := context.Background()
	
	pool, mock, err := NewPoolWithMock(ctx, Config{}, true)
	if err != nil { t.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	if mock != nil {
		rows := NewRows([]string{"c"})
		rows = AddRow(rows, 1)
		mock.ExpectQuery(`SELECT 1`).WillReturnRows(rows)
		mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(1).WillReturnResult(NewResult(1,1))
		// ExecCached uses prepare
		mock.ExpectPrepare(`INSERT INTO t\(a\) VALUES\(\?\)`).ExpectExec().WithArgs(1).WillReturnResult(NewResult(1,1))
		mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(1).WillReturnResult(NewResult(1,1))
		mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\)`).WithArgs(1, "x", 2, "y").WillReturnResult(NewResult(0,2))
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(2).WillReturnResult(NewResult(2,1))
		mock.ExpectCommit()
	}

	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		// Test that Conn implements DatabaseConn interface
		rs, err := c.Query(ctx, "SELECT 1")
		if err != nil { return err }
		defer rs.Close()
		
		_, err = c.Exec(ctx, "INSERT INTO t(a) VALUES(?)", 1)
		if err != nil { return err }
		
		c.EnableStmtCache(2)
		_, err = c.ExecCached(ctx, "INSERT INTO t(a) VALUES(?)", 1)
		if err != nil { return err }
		
		_, err = c.NamedExec(ctx, "INSERT INTO t(a) VALUES(:a)", map[string]any{"a": 1})
		if err != nil { return err }
		
		rows := [][]any{{1, "x"}, {2, "y"}}
		_, err = c.BulkInsert(ctx, "t", []string{"a", "b"}, rows)
		if err != nil { return err }
		
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }

	// Test transaction interface
	err = pool.WithinTx(ctx, func(tx DatabaseTx) error {
		_, err := tx.Exec(ctx, "INSERT INTO t(a) VALUES(?)", 2)
		return err
	})
	if err != nil { t.Fatalf("WithinTx: %v", err) }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("expectations: %v", err) }
	}
}

func TestInterface_AllMethodsAccessible(t *testing.T) {
	ctx := context.Background()
	
	pool, mock, err := NewPoolWithMock(ctx, Config{}, true)
	if err != nil { t.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	// Test all DatabasePool methods are accessible
	var dbPool DatabasePool = pool
	_ = dbPool.Ping(ctx)
	_ = dbPool.SelfCheck(ctx)
	_ = dbPool.Close()
	
	// Test that we can call methods through interface
	if mock != nil {
		mock.ExpectQuery(`SELECT 1`).WillReturnRows(NewRows([]string{"c"}))
	}
	
	err = dbPool.WithConn(ctx, func(conn DatabaseConn) error {
		// All DatabaseConn methods should be accessible
		_, _ = conn.Query(ctx, "SELECT 1")
		_, _ = conn.Exec(ctx, "SELECT 1")
		_ = conn.QueryRow(ctx, "SELECT 1")
		_ = conn.QueryStream(ctx, "SELECT 1", func([]any) error { return nil })
		conn.EnableStmtCache(1)
		_, _ = conn.ExecCached(ctx, "SELECT 1")
		_, _ = conn.QueryCached(ctx, "SELECT 1")
		_, _ = conn.NamedExec(ctx, "SELECT 1", map[string]any{})
		_, _ = conn.NamedQuery(ctx, "SELECT 1", map[string]any{})
		_, _ = conn.BulkInsert(ctx, "t", []string{"a"}, [][]any{{1}})
		_, _ = conn.InsertOnDuplicate(ctx, "t", []string{"a"}, [][]any{{1}}, []string{"a"})
		return nil
	})
	
	err = dbPool.WithinTx(ctx, func(tx DatabaseTx) error {
		// All DatabaseTx methods should be accessible
		_, _ = tx.Exec(ctx, "SELECT 1")
		return nil
	})

	if mock != nil {
		_ = mock.ExpectationsWereMet()
	}
}
