package ygggo_mysql

import (
	"context"
	"testing"
)

func TestBulkInsert_Simple(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY, a INT, b TEXT)")
		if err != nil { return err }

		// Test bulk insert
		rows := [][]any{{1, "x"}, {2, "y"}}
		res, err := c.BulkInsert(context.Background(), "t", []string{"a", "b"}, rows)
		if err != nil { return err }
		aff, _ := res.RowsAffected()
		if aff != 2 { t.Fatalf("affected=%d", aff) }

		// Verify data was inserted
		rs, err := c.Query(context.Background(), "SELECT COUNT(*) FROM t")
		if err != nil { return err }
		defer rs.Close()

		var count int
		if rs.Next() {
			err = rs.Scan(&count)
			if err != nil { return err }
		}
		if count != 2 { t.Fatalf("expected 2 rows, got %d", count) }

		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
}

func TestInsertOnDuplicate_Simple(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table with unique constraint
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY, a INT UNIQUE, b TEXT)")
		if err != nil { return err }

		// Test INSERT ON DUPLICATE KEY UPDATE
		rows := [][]any{{1, "x"}, {1, "y"}} // Duplicate key
		_, err = c.InsertOnDuplicate(context.Background(), "t", []string{"a", "b"}, rows, []string{"b"})
		return err
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
}

func TestBulkInsert_EmptyRows_Error(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		_, err := c.BulkInsert(context.Background(), "t", []string{"a"}, nil)
		if err == nil { t.Fatalf("expected error for empty rows") }
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
}

func TestBulkInsert_ColumnMismatch_Error(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		rows := [][]any{{1, "x"}, {2}} // mismatch for second row
		_, err := c.BulkInsert(context.Background(), "t", []string{"a", "b"}, rows)
		if err == nil { t.Fatalf("expected error for mismatch columns") }
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
}

