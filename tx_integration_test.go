package ygggo_mysql

import (
	"context"
	"testing"
)

func TestWithinTx_Retry_DeadlockThenSuccess(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()

	// Create test table and initial data
	err := helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INTEGER PRIMARY KEY, a INTEGER)")
		if err != nil { return err }
		_, err = c.Exec(context.Background(), "INSERT INTO t (id, a) VALUES (1, 1)")
		return err
	})
	if err != nil { t.Fatalf("Setup failed: %v", err) }

	p := helper.Pool()
	p.retry = RetryPolicy{MaxAttempts: 3, BaseBackoff: 1, MaxBackoff: 10, Jitter: false}

	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "UPDATE t SET a=? WHERE id=?", 2, 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx err: %v", err) }
}

func TestWithinTx_Retry_ReadOnlyThenSuccess(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()

	// Create test table
	err := helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INTEGER PRIMARY KEY, a INTEGER)")
		return err
	})
	if err != nil { t.Fatalf("Setup failed: %v", err) }

	p := helper.Pool()
	p.retry = RetryPolicy{MaxAttempts: 2, BaseBackoff: 1, MaxBackoff: 10, Jitter: false}

	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "INSERT INTO t(a) VALUES(?)", 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx err: %v", err) }
}

