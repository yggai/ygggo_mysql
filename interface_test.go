package ygggo_mysql

import (
	"context"
	"testing"
)

func TestPoolInterface_DockerImplementation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	ctx := context.Background()

	// Test that Docker pool implements DatabasePool interface
	var pool DatabasePool
	helper, err := NewDockerTestHelper(ctx)
	if err != nil { t.Fatalf("NewDockerTestHelper: %v", err) }
	defer helper.Close()

	pool = helper.Pool() // This should compile if interface is implemented

	// Test basic interface methods
	if err := pool.Ping(ctx); err != nil { t.Fatalf("Ping: %v", err) }
}

func TestConnInterface_DockerImplementation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	ctx := context.Background()

	helper, err := NewDockerTestHelper(ctx)
	if err != nil { t.Fatalf("NewDockerTestHelper: %v", err) }
	defer helper.Close()

	pool := helper.Pool()

	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		// Create test table
		_, err := c.Exec(ctx, "CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY, a INT, b TEXT)")
		if err != nil { return err }

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
}
