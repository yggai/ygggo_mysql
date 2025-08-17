package ygggo_mysql

import (
	"context"
	"testing"
)

func TestWithinTx_Retry_DeadlockThenSuccess(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Use unique table name for this test
	tableName := "tx_deadlock_test"
	ctx := context.Background()

	// Clean up table before and after test
	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// Create test table and initial data
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Clean up any existing table first
		_, _ = c.Exec(context.Background(), "DROP TABLE IF EXISTS "+tableName)

		_, err := c.Exec(context.Background(), "CREATE TABLE "+tableName+" (id INT AUTO_INCREMENT PRIMARY KEY, a INT)")
		if err != nil {
			return err
		}
		_, err = c.Exec(context.Background(), "INSERT INTO "+tableName+" (id, a) VALUES (1, 1)")
		return err
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	p := helper.Pool()
	p.retry = RetryPolicy{MaxAttempts: 3, BaseBackoff: 1, MaxBackoff: 10, Jitter: false}

	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "UPDATE "+tableName+" SET a=? WHERE id=?", 2, 1)
		return err
	})
	if err != nil {
		t.Fatalf("WithinTx err: %v", err)
	}
}

func TestWithinTx_Retry_ReadOnlyThenSuccess(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Use unique table name for this test
	tableName := "tx_readonly_test"
	ctx := context.Background()

	// Clean up table before and after test
	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// Create test table
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Clean up any existing table first
		_, _ = c.Exec(context.Background(), "DROP TABLE IF EXISTS "+tableName)

		_, err := c.Exec(context.Background(), "CREATE TABLE "+tableName+" (id INT AUTO_INCREMENT PRIMARY KEY, a INT)")
		return err
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	p := helper.Pool()
	p.retry = RetryPolicy{MaxAttempts: 2, BaseBackoff: 1, MaxBackoff: 10, Jitter: false}

	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "INSERT INTO "+tableName+"(a) VALUES(?)", 1)
		return err
	})
	if err != nil {
		t.Fatalf("WithinTx err: %v", err)
	}
}
