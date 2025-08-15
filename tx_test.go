package ygggo_mysql

import (
	"context"
	"errors"
	"testing"
)

func TestWithinTx_CommitOnSuccess(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()

	err := helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INTEGER PRIMARY KEY, a INTEGER)")
		return err
	})
	if err != nil { t.Fatalf("Setup failed: %v", err) }

	err = helper.Pool().WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "INSERT INTO t(a) VALUES(?)", 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx err: %v", err) }

	// Verify data was committed
	helper.AssertRowCount("t", 1)
}

func TestWithinTx_RollbackOnFnError(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()

	err := helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INTEGER PRIMARY KEY, a INTEGER)")
		return err
	})
	if err != nil { t.Fatalf("Setup failed: %v", err) }

	sentinel := errors.New("boom")
	err = helper.Pool().WithinTx(context.Background(), func(tx DatabaseTx) error { return sentinel })
	if !errors.Is(err, sentinel) { t.Fatalf("expected sentinel, got %v", err) }

	// Verify no data was committed (rollback worked)
	helper.AssertRowCount("t", 0)
}

func TestWithinTx_RetryOnDeadlock(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()

	err := helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table and insert initial data
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INTEGER PRIMARY KEY, a INTEGER)")
		if err != nil { return err }
		_, err = c.Exec(context.Background(), "INSERT INTO t (id, a) VALUES (1, 1)")
		return err
	})
	if err != nil { t.Fatalf("Setup failed: %v", err) }

	// Set retry policy
	helper.Pool().retry = RetryPolicy{MaxAttempts: 2, BaseBackoff: 1, MaxBackoff: 10, Jitter: false}

	err = helper.Pool().WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "UPDATE t SET a=? WHERE id=?", 2, 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx err: %v", err) }

	// Verify data was updated
	conn, err := helper.Pool().Acquire(context.Background())
	if err != nil { t.Fatalf("Acquire: %v", err) }
	defer conn.Close()

	rs, err := conn.Query(context.Background(), "SELECT a FROM t WHERE id = 1")
	if err != nil { t.Fatalf("Query: %v", err) }
	defer rs.Close()

	var a int
	if rs.Next() {
		err = rs.Scan(&a)
		if err != nil { t.Fatalf("Scan: %v", err) }
	}
	if a != 2 { t.Fatalf("expected a=2, got a=%d", a) }
}

