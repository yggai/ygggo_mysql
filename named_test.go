package ygggo_mysql

import (
	"context"
	"reflect"
	"testing"
)

type row struct {
	A int    `db:"a"`
	B string `db:"b"`
}

func TestNamedExec_WithStruct(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY, a INT, b TEXT)")
		if err != nil { return err }

		// Test named exec with struct
		_, err = c.NamedExec(context.Background(), "INSERT INTO t (a,b) VALUES (:a,:b)", row{A:1, B:"x"})
		if err != nil { return err }

		// Verify data was inserted
		rs, err := c.Query(context.Background(), "SELECT a, b FROM t WHERE a = 1")
		if err != nil { return err }
		defer rs.Close()

		if !rs.Next() { t.Fatalf("no rows found") }
		var a int
		var b string
		err = rs.Scan(&a, &b)
		if err != nil { return err }

		if a != 1 || b != "x" { t.Fatalf("expected a=1, b='x', got a=%d, b='%s'", a, b) }

		return nil
	})
	if err != nil { t.Fatalf("NamedExec err: %v", err) }
}

func TestNamedExec_WithSliceStructs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY, a INT, b TEXT)")
		if err != nil { return err }

		// Test named exec with slice of structs
		rows := []row{{1,"x"},{2,"y"}}
		_, err = c.NamedExec(context.Background(), "INSERT INTO t (a,b) VALUES (:a,:b)", rows)
		if err != nil { return err }

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
	if err != nil { t.Fatalf("NamedExec slice err: %v", err) }
}

func TestNamedQuery_WithMap(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table and insert data (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY)")
		if err != nil { return err }
		_, err = c.Exec(context.Background(), "INSERT INTO t (id) VALUES (42)")
		if err != nil { return err }

		// Test named query with map
		rs, err := c.NamedQuery(context.Background(), "SELECT * FROM t WHERE id=:id", map[string]any{"id": 42})
		if err != nil { return err }
		defer rs.Close()
		var ids []int
		for rs.Next() {
			var id int
			if err := rs.Scan(&id); err != nil { return err }
			ids = append(ids, id)
		}
		if !reflect.DeepEqual(ids, []int{42}) { t.Fatalf("ids=%v", ids) }
		return nil
	})
	if err != nil { t.Fatalf("NamedQuery err: %v", err) }
}

func TestIn_Helper_ExpandsSlice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	err = helper.Pool().WithConn(context.Background(), func(c DatabaseConn) error {
		// Create test table and insert data (MySQL syntax)
		_, err := c.Exec(context.Background(), "CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY, kind TEXT)")
		if err != nil { return err }
		_, err = c.Exec(context.Background(), "INSERT INTO t (id, kind) VALUES (1, 'a'), (2, 'a'), (3, 'a')")
		if err != nil { return err }

		// Test BuildIn helper
		q, args, err := BuildIn("SELECT * FROM t WHERE id IN (?) AND kind=?", []int{1,2,3}, "a")
		if err != nil { return err }
		rs, err := c.Query(context.Background(), q, args...)
		if err != nil { return err }
		defer rs.Close()

		// Count results
		count := 0
		for rs.Next() { count++ }
		if count != 3 { t.Fatalf("expected 3 rows, got %d", count) }

		return nil
	})
	if err != nil { t.Fatalf("BuildIn err: %v", err) }
}
