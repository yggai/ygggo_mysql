package ygggo_mysql

import (
	"context"
	"testing"
)

// Test basic SQLite mock functionality - this should fail initially
func TestSQLiteMock_BasicQuery(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil {
		t.Fatalf("NewMockPool: %v", err)
	}
	defer pool.Close()

	// Set up expectation
	rows := NewRows([]string{"id", "name"})
	rows = AddRow(rows, 1, "Alice")
	rows = AddRow(rows, 2, "Bob")
	
	mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)

	// Execute query
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT id, name FROM users")
		if err != nil {
			return err
		}
		defer rs.Close()

		count := 0
		for rs.Next() {
			var id int
			var name string
			if err := rs.Scan(&id, &name); err != nil {
				return err
			}
			count++
			
			// Verify data matches expectations
			if count == 1 && (id != 1 || name != "Alice") {
				t.Errorf("Expected first row: id=1, name=Alice, got: id=%d, name=%s", id, name)
			}
			if count == 2 && (id != 2 || name != "Bob") {
				t.Errorf("Expected second row: id=2, name=Bob, got: id=%d, name=%s", id, name)
			}
		}
		
		if count != 2 {
			t.Errorf("Expected 2 rows, got %d", count)
		}
		
		return rs.Err()
	})
	
	if err != nil {
		t.Fatalf("Query execution: %v", err)
	}

	// Verify expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Expectations not met: %v", err)
	}
}

// Test SQLite mock with arguments - this should fail initially
func TestSQLiteMock_QueryWithArgs(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil {
		t.Fatalf("NewMockPool: %v", err)
	}
	defer pool.Close()

	// Set up expectation
	rows := NewRows([]string{"name"})
	rows = AddRow(rows, "Alice")
	
	mock.ExpectQuery("SELECT name FROM users WHERE id = ?").
		WithArgs(1).
		WillReturnRows(rows)

	// Execute query
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT name FROM users WHERE id = ?", 1)
		if err != nil {
			return err
		}
		defer rs.Close()

		if rs.Next() {
			var name string
			if err := rs.Scan(&name); err != nil {
				return err
			}
			if name != "Alice" {
				t.Errorf("Expected name=Alice, got name=%s", name)
			}
		} else {
			t.Error("Expected one row, got none")
		}
		
		return rs.Err()
	})
	
	if err != nil {
		t.Fatalf("Query execution: %v", err)
	}

	// Verify expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Expectations not met: %v", err)
	}
}

// Test SQLite mock exec - this should fail initially
func TestSQLiteMock_Exec(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil {
		t.Fatalf("NewMockPool: %v", err)
	}
	defer pool.Close()

	// Set up expectation
	mock.ExpectExec("INSERT INTO users (name) VALUES (?)").
		WithArgs("Charlie").
		WillReturnResult(NewResult(3, 1))

	// Execute statement
	err = pool.WithConn(ctx, func(c DatabaseConn) error {
		result, err := c.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Charlie")
		if err != nil {
			return err
		}
		
		lastId, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if lastId != 3 {
			t.Errorf("Expected LastInsertId=3, got %d", lastId)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			t.Errorf("Expected RowsAffected=1, got %d", rowsAffected)
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Exec execution: %v", err)
	}

	// Verify expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Expectations not met: %v", err)
	}
}

// Test SQLite mock ping - this should fail initially
func TestSQLiteMock_Ping(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil {
		t.Fatalf("NewMockPool: %v", err)
	}
	defer pool.Close()

	mock.ExpectPing()
	
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Expectations not met: %v", err)
	}
}

// Test SQLite mock transaction - this should work
func TestSQLiteMock_Transaction(t *testing.T) {
	ctx := context.Background()
	pool, mock, err := NewMockPool(ctx, Config{})
	if err != nil {
		t.Fatalf("NewMockPool: %v", err)
	}
	defer pool.Close()

	// Set up transaction expectations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users (name) VALUES (?)").
		WithArgs("Dave").
		WillReturnResult(NewResult(4, 1))
	mock.ExpectCommit()

	// Execute transaction
	err = pool.WithinTx(ctx, func(tx DatabaseTx) error {
		result, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Dave")
		if err != nil {
			return err
		}

		lastId, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if lastId != 4 {
			t.Errorf("Expected LastInsertId=4, got %d", lastId)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Transaction execution: %v", err)
	}

	// Verify expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Expectations not met: %v", err)
	}
}
