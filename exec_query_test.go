package ygggo_mysql

import (
	"context"
	"testing"
)

func TestConn_Exec_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Do everything in a single WithConn to avoid nesting
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create table (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE test_table (id INT AUTO_INCREMENT PRIMARY KEY, a INT, b TEXT)")
		if err != nil { return err }

		// Insert data
		res, err := c.Exec(context.Background(), "INSERT INTO test_table(a,b) VALUES(?,?)", 1, "x")
		if err != nil { return err }
		affected, _ := res.RowsAffected()
		if affected != 1 { t.Fatalf("affected=%d", affected) }

		// Verify data was inserted
		rs, err := c.Query(context.Background(), "SELECT COUNT(*) FROM test_table")
		if err != nil { return err }
		defer rs.Close()

		var count int
		if rs.Next() {
			err = rs.Scan(&count)
			if err != nil { return err }
		}

		if count != 1 {
			t.Fatalf("expected 1 row, got %d", count)
		}

		return nil
	})
	if err != nil { t.Fatalf("WithConn err: %v", err) }
}

func TestConn_Query_And_Stream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Do everything in a single WithConn to avoid nesting
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE test_table (id INT AUTO_INCREMENT PRIMARY KEY, name TEXT)")
		if err != nil { return err }

		// Insert test data
		_, err = c.Exec(context.Background(), "INSERT INTO test_table(id, name) VALUES(1, 'a')")
		if err != nil { return err }
		_, err = c.Exec(context.Background(), "INSERT INTO test_table(id, name) VALUES(2, 'b')")
		if err != nil { return err }

		// Query and read all
		rs, err := c.Query(context.Background(), "SELECT id,name FROM test_table ORDER BY id")
		if err != nil { return err }
		defer rs.Close()
		cnt := 0
		for rs.Next() { cnt++ }
		if cnt != 2 { t.Fatalf("rows cnt=%d", cnt) }

		return nil
	})
	if err != nil { t.Fatalf("WithConn err: %v", err) }

	// Test streaming in a separate WithConn
	callbacks := 0
	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		return c.QueryStream(context.Background(), "SELECT id,name FROM test_table ORDER BY id", func(_ []any) error {
			callbacks++
			return nil
		})
	})
	if err != nil { t.Fatalf("stream err: %v", err) }
	if callbacks != 2 { t.Fatalf("callbacks=%d", callbacks) }
}

