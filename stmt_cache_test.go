package ygggo_mysql

import (
	"context"
	"testing"
)

func TestStmtCache_PerConn_CachesPrepare(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Use unique table name for this test
	tableName := "stmt_cache_prepare_test"
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
		if err != nil {
			return err
		}

		// Enable statement cache
		c.EnableStmtCache(2)

		// Execute same statement multiple times - should use cached prepared statement
		if _, err := c.ExecCached(context.Background(), "INSERT INTO "+tableName+"(a) VALUES(?)", 1); err != nil {
			return err
		}
		if _, err := c.ExecCached(context.Background(), "INSERT INTO "+tableName+"(a) VALUES(?)", 2); err != nil {
			return err
		}

		// Verify data was inserted
		rs, err := c.Query(context.Background(), "SELECT COUNT(*) FROM "+tableName)
		if err != nil {
			return err
		}
		defer rs.Close()

		var count int
		if rs.Next() {
			err = rs.Scan(&count)
			if err != nil {
				return err
			}
		}
		if count != 2 {
			t.Fatalf("expected 2 rows, got %d", count)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WithConn err: %v", err)
	}
}

func TestStmtCache_LRUEvicts(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Enable statement cache with capacity 1 to test LRU eviction
		c.EnableStmtCache(1)

		// Execute different queries to test cache eviction
		rs1, err := c.QueryCached(context.Background(), "SELECT 1")
		if err != nil {
			return err
		}
		rs1.Close()

		rs2, err := c.QueryCached(context.Background(), "SELECT 2") // should evict SELECT 1
		if err != nil {
			return err
		}
		rs2.Close()

		rs3, err := c.QueryCached(context.Background(), "SELECT 1") // should re-prepare
		if err != nil {
			return err
		}
		rs3.Close()

		return nil
	})
	if err != nil {
		t.Fatalf("WithConn err: %v", err)
	}
}

func TestStmtCache_PerConn_Isolated(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Use unique table name for this test
	tableName := "stmt_cache_isolated_test"
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
		if err != nil {
			return err
		}
		_, err = c.Exec(context.Background(), "INSERT INTO "+tableName+" (id, a) VALUES (1, 0)")
		return err
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// First connection
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		c.EnableStmtCache(2)
		_, err := c.ExecCached(context.Background(), "UPDATE "+tableName+" SET a=? WHERE id=1", 1)
		return err
	})
	if err != nil {
		t.Fatalf("WithConn 1 err: %v", err)
	}

	// Second connection (should have its own cache)
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		c.EnableStmtCache(2)
		_, err := c.ExecCached(context.Background(), "UPDATE "+tableName+" SET a=? WHERE id=1", 2)
		return err
	})
	if err != nil {
		t.Fatalf("WithConn 2 err: %v", err)
	}

	// Verify final value
	conn, err := helper.Pool().Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer conn.Close()

	rs, err := conn.Query(context.Background(), "SELECT a FROM t WHERE id=1")
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
