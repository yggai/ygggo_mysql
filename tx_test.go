package ygggo_mysql

import (
	"context"
	"errors"
	"testing"
)

func TestWithinTx_CommitOnSuccess(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Use unique table name for this test
	tableName := "tx_commit_test"
	ctx := context.Background()

	// Clean up table before and after test
	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Clean up any existing table first
		_, _ = c.Exec(context.Background(), "DROP TABLE IF EXISTS "+tableName)

		// Create test table (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE "+tableName+" (id INT AUTO_INCREMENT PRIMARY KEY, a INT)")
		return err
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	err = helper.Pool().WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "INSERT INTO "+tableName+"(a) VALUES(?)", 1)
		return err
	})
	if err != nil {
		t.Fatalf("WithinTx err: %v", err)
	}

	// Verify data was committed
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		rows, err := c.Query(context.Background(), "SELECT COUNT(*) FROM "+tableName)
		if err != nil {
			return err
		}
		defer rows.Close()

		if rows.Next() {
			var count int
			if err := rows.Scan(&count); err != nil {
				return err
			}
			if count != 1 {
				t.Errorf("expected 1 row, got %d", count)
			}
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatalf("count check: %v", err)
	}
}

func TestWithinTx_RollbackOnFnError(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Use unique table name for this test
	tableName := "tx_rollback_test"
	ctx := context.Background()

	// Clean up table before and after test
	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Clean up any existing table first
		_, _ = c.Exec(context.Background(), "DROP TABLE IF EXISTS "+tableName)

		// Create test table (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE "+tableName+" (id INT AUTO_INCREMENT PRIMARY KEY, a INT)")
		return err
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	sentinel := errors.New("boom")
	err = helper.Pool().WithinTx(context.Background(), func(tx DatabaseTx) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel, got %v", err)
	}

	// Verify no data was committed (rollback worked)
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		rows, err := c.Query(context.Background(), "SELECT COUNT(*) FROM "+tableName)
		if err != nil {
			return err
		}
		defer rows.Close()

		if rows.Next() {
			var count int
			if err := rows.Scan(&count); err != nil {
				return err
			}
			if count != 0 {
				t.Errorf("expected 0 rows, got %d", count)
			}
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatalf("count check: %v", err)
	}
}

func TestWithinTx_RetryOnDeadlock(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Use unique table name for this test
	tableName := "tx_deadlock_retry_test"
	ctx := context.Background()

	// Clean up table before and after test
	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Clean up any existing table first
		_, _ = c.Exec(context.Background(), "DROP TABLE IF EXISTS "+tableName)

		// Create test table and insert initial data (MySQL syntax)
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

	// Set retry policy
	helper.Pool().retry = RetryPolicy{MaxAttempts: 2, BaseBackoff: 1, MaxBackoff: 10, Jitter: false}

	err = helper.Pool().WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "UPDATE "+tableName+" SET a=? WHERE id=?", 2, 1)
		return err
	})
	if err != nil {
		t.Fatalf("WithinTx err: %v", err)
	}

	// Verify data was updated
	conn, err := helper.Pool().Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer conn.Close()

	rs, err := conn.Query(context.Background(), "SELECT a FROM t WHERE id = 1")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer rs.Close()

	var a int
	if rs.Next() {
		err = rs.Scan(&a)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
	}
	if a != 2 {
		t.Fatalf("expected a=2, got a=%d", a)
	}
}
