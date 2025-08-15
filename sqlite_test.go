package ygggo_mysql

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLite_BasicConnection(t *testing.T) {
	pool, err := NewSQLitePool(context.Background())
	if err != nil {
		t.Fatalf("NewSQLitePool failed: %v", err)
	}
	defer pool.Close()

	// Test basic connectivity
	ctx := context.Background()
	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestSQLite_WithConn_Query(t *testing.T) {
	pool, err := NewSQLitePool(context.Background())
	if err != nil {
		t.Fatalf("NewSQLitePool failed: %v", err)
	}
	defer pool.Close()

	// Create test table
	ctx := context.Background()
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)`)
		return err
	})
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	// Insert test data
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "Alice", "alice@example.com")
		if err != nil {
			return err
		}
		_, err = c.Exec(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "Bob", "bob@example.com")
		return err
	})
	if err != nil {
		t.Fatalf("Insert data failed: %v", err)
	}

	// Query data
	var users []struct {
		ID    int
		Name  string
		Email string
	}

	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, `SELECT id, name, email FROM users ORDER BY id`)
		if err != nil {
			return err
		}
		defer rs.Close()

		for rs.Next() {
			var user struct {
				ID    int
				Name  string
				Email string
			}
			err := rs.Scan(&user.ID, &user.Name, &user.Email)
			if err != nil {
				return err
			}
			users = append(users, user)
		}
		return rs.Err()
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Verify results
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "Alice" || users[0].Email != "alice@example.com" {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if users[1].Name != "Bob" || users[1].Email != "bob@example.com" {
		t.Fatalf("unexpected second user: %+v", users[1])
	}
}

func TestSQLite_WithinTx_CommitOnSuccess(t *testing.T) {
	pool, err := NewSQLitePool(context.Background())
	if err != nil {
		t.Fatalf("NewSQLitePool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Create test table
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER)`)
		return err
	})
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	// Insert initial data
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `INSERT INTO accounts (id, balance) VALUES (1, 1000), (2, 500)`)
		return err
	})
	if err != nil {
		t.Fatalf("Insert initial data failed: %v", err)
	}

	// Perform transaction
	err = pool.WithinTx(ctx, func(tx DatabaseTx) error {
		// Transfer 200 from account 1 to account 2
		_, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance - 200 WHERE id = 1`)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance + 200 WHERE id = 2`)
		return err
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify balances
	var balance1, balance2 int
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, `SELECT balance FROM accounts WHERE id = 1`)
		if err != nil {
			return err
		}
		defer rs.Close()
		if rs.Next() {
			err = rs.Scan(&balance1)
			if err != nil {
				return err
			}
		}

		rs2, err := c.Query(ctx, `SELECT balance FROM accounts WHERE id = 2`)
		if err != nil {
			return err
		}
		defer rs2.Close()
		if rs2.Next() {
			err = rs2.Scan(&balance2)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Query balances failed: %v", err)
	}

	if balance1 != 800 {
		t.Fatalf("expected balance1 = 800, got %d", balance1)
	}
	if balance2 != 700 {
		t.Fatalf("expected balance2 = 700, got %d", balance2)
	}
}

func TestSQLite_WithinTx_RollbackOnError(t *testing.T) {
	pool, err := NewSQLitePool(context.Background())
	if err != nil {
		t.Fatalf("NewSQLitePool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Create test table
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER)`)
		return err
	})
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	// Insert initial data
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `INSERT INTO accounts (id, balance) VALUES (1, 1000), (2, 500)`)
		return err
	})
	if err != nil {
		t.Fatalf("Insert initial data failed: %v", err)
	}

	// Perform failing transaction
	err = pool.WithinTx(ctx, func(tx DatabaseTx) error {
		// Transfer 200 from account 1 to account 2
		_, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance - 200 WHERE id = 1`)
		if err != nil {
			return err
		}
		// Simulate an error
		return sql.ErrTxDone
	})
	if err == nil {
		t.Fatalf("Expected transaction to fail")
	}

	// Verify balances are unchanged
	var balance1, balance2 int
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, `SELECT balance FROM accounts WHERE id = 1`)
		if err != nil {
			return err
		}
		defer rs.Close()
		if rs.Next() {
			err = rs.Scan(&balance1)
			if err != nil {
				return err
			}
		}

		rs2, err := c.Query(ctx, `SELECT balance FROM accounts WHERE id = 2`)
		if err != nil {
			return err
		}
		defer rs2.Close()
		if rs2.Next() {
			err = rs2.Scan(&balance2)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Query balances failed: %v", err)
	}

	// Balances should be unchanged due to rollback
	if balance1 != 1000 {
		t.Fatalf("expected balance1 = 1000 (unchanged), got %d", balance1)
	}
	if balance2 != 500 {
		t.Fatalf("expected balance2 = 500 (unchanged), got %d", balance2)
	}
}

func TestSQLite_TelemetryAndMetrics(t *testing.T) {
	pool, err := NewSQLitePool(context.Background())
	if err != nil {
		t.Fatalf("NewSQLitePool failed: %v", err)
	}
	defer pool.Close()

	// Enable telemetry and metrics
	pool.EnableTelemetry(true)
	pool.EnableMetrics(true)

	ctx := context.Background()

	// Create test table and perform operations
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, `CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)`)
		if err != nil {
			return err
		}
		_, err = c.Exec(ctx, `INSERT INTO test (value) VALUES (?)`, "test_value")
		if err != nil {
			return err
		}
		rs, err := c.Query(ctx, `SELECT id, value FROM test`)
		if err != nil {
			return err
		}
		defer rs.Close()
		
		for rs.Next() {
			var id int
			var value string
			err := rs.Scan(&id, &value)
			if err != nil {
				return err
			}
		}
		return rs.Err()
	})
	if err != nil {
		t.Fatalf("Operations failed: %v", err)
	}

	// Test should pass without deadlocks
	t.Log("Telemetry and metrics integration working with SQLite")
}
