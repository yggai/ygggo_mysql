package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

// TestHelper provides utilities for testing with SQLite
type TestHelper struct {
	pool *Pool
	t    *testing.T
}

// NewTestHelper creates a new test helper with SQLite backend
func NewTestHelper(t *testing.T) *TestHelper {
	ctx := context.Background()
	pool, err := NewSQLiteTestPool(ctx)
	if err != nil {
		t.Fatalf("Failed to create SQLite test pool: %v", err)
	}
	
	return &TestHelper{
		pool: pool,
		t:    t,
	}
}

// Pool returns the underlying pool
func (h *TestHelper) Pool() *Pool {
	return h.pool
}

// Close closes the test helper
func (h *TestHelper) Close() {
	if h.pool != nil {
		h.pool.Close()
	}
}

// CreateTable creates a table for testing using direct connection
func (h *TestHelper) CreateTable(tableName, schema string) {
	ctx := context.Background()
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.t.Fatalf("Failed to acquire connection: %v", err)
	}
	defer conn.Close()

	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE TABLE %s (%s)", tableName, schema))
	if err != nil {
		h.t.Fatalf("Failed to create table %s: %v", tableName, err)
	}
}

// InsertData inserts test data using direct connection
func (h *TestHelper) InsertData(query string, args ...any) sql.Result {
	ctx := context.Background()
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.t.Fatalf("Failed to acquire connection: %v", err)
	}
	defer conn.Close()

	result, err := conn.Exec(ctx, query, args...)
	if err != nil {
		h.t.Fatalf("Failed to insert data: %v", err)
	}
	return result
}

// QueryData queries test data using direct connection
func (h *TestHelper) QueryData(query string, args ...any) *sql.Rows {
	ctx := context.Background()
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.t.Fatalf("Failed to acquire connection: %v", err)
	}
	// Note: caller is responsible for closing both rows and connection

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		conn.Close()
		h.t.Fatalf("Failed to query data: %v", err)
	}
	return rows
}

// CountRows counts rows in a table
func (h *TestHelper) CountRows(tableName string) int {
	rows := h.QueryData(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName))
	defer rows.Close()
	
	var count int
	if rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			h.t.Fatalf("Failed to scan count: %v", err)
		}
	}
	return count
}

// ExpectExecSuccess expects an exec operation to succeed
func (h *TestHelper) ExpectExecSuccess(query string, args ...any) sql.Result {
	return h.InsertData(query, args...)
}

// ExpectExecError expects an exec operation to fail
func (h *TestHelper) ExpectExecError(query string, args ...any) {
	ctx := context.Background()
	err := h.pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, query, args...)
		return err
	})
	if err == nil {
		h.t.Fatalf("Expected exec to fail but it succeeded")
	}
}

// ExpectQuerySuccess expects a query to succeed and return rows
func (h *TestHelper) ExpectQuerySuccess(query string, args ...any) *sql.Rows {
	return h.QueryData(query, args...)
}

// ExpectQueryError expects a query to fail
func (h *TestHelper) ExpectQueryError(query string, args ...any) {
	ctx := context.Background()
	err := h.pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Query(ctx, query, args...)
		return err
	})
	if err == nil {
		h.t.Fatalf("Expected query to fail but it succeeded")
	}
}

// SetupBasicTable creates a basic test table with id, name, value columns
func (h *TestHelper) SetupBasicTable() {
	h.CreateTable("test_table", `
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		value TEXT
	`)
}

// SetupUsersTable creates a users table for testing
func (h *TestHelper) SetupUsersTable() {
	h.CreateTable("users", `
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	`)
}

// SetupAccountsTable creates an accounts table for transaction testing
func (h *TestHelper) SetupAccountsTable() {
	h.CreateTable("accounts", `
		id INTEGER PRIMARY KEY,
		balance INTEGER NOT NULL DEFAULT 0
	`)
}

// InsertUser inserts a user and returns the ID
func (h *TestHelper) InsertUser(name, email string) int64 {
	result := h.InsertData("INSERT INTO users (name, email) VALUES (?, ?)", name, email)
	id, err := result.LastInsertId()
	if err != nil {
		h.t.Fatalf("Failed to get last insert ID: %v", err)
	}
	return id
}

// InsertAccount inserts an account
func (h *TestHelper) InsertAccount(id int, balance int) {
	h.InsertData("INSERT INTO accounts (id, balance) VALUES (?, ?)", id, balance)
}

// GetAccountBalance gets account balance
func (h *TestHelper) GetAccountBalance(id int) int {
	rows := h.QueryData("SELECT balance FROM accounts WHERE id = ?", id)
	defer rows.Close()
	
	var balance int
	if rows.Next() {
		err := rows.Scan(&balance)
		if err != nil {
			h.t.Fatalf("Failed to scan balance: %v", err)
		}
	}
	return balance
}

// WithTransaction executes a function within a transaction
func (h *TestHelper) WithTransaction(fn func(tx DatabaseTx) error) error {
	ctx := context.Background()
	return h.pool.WithinTx(ctx, fn)
}

// ExpectTransactionSuccess expects a transaction to succeed
func (h *TestHelper) ExpectTransactionSuccess(fn func(tx DatabaseTx) error) {
	err := h.WithTransaction(fn)
	if err != nil {
		h.t.Fatalf("Expected transaction to succeed but got error: %v", err)
	}
}

// ExpectTransactionError expects a transaction to fail
func (h *TestHelper) ExpectTransactionError(fn func(tx DatabaseTx) error) {
	err := h.WithTransaction(fn)
	if err == nil {
		h.t.Fatalf("Expected transaction to fail but it succeeded")
	}
}

// EnableTelemetry enables telemetry for testing
func (h *TestHelper) EnableTelemetry() {
	h.pool.EnableTelemetry(true)
}

// EnableMetrics enables metrics for testing
func (h *TestHelper) EnableMetrics() {
	h.pool.EnableMetrics(true)
}

// EnableLogging enables logging for testing
func (h *TestHelper) EnableLogging() {
	h.pool.EnableLogging(true)
}

// SimulateDeadlock simulates a deadlock error for testing retry logic
func (h *TestHelper) SimulateDeadlock() error {
	// SQLite doesn't have traditional deadlocks like MySQL, but we can simulate
	// a retryable error by using a constraint violation that gets classified as retryable
	return &mysql.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock"}
}

// SimulateTimeout simulates a timeout error
func (h *TestHelper) SimulateTimeout() error {
	return &mysql.MySQLError{Number: 1205, Message: "Lock wait timeout exceeded"}
}

// SimulateReadOnly simulates a read-only error
func (h *TestHelper) SimulateReadOnly() error {
	return &mysql.MySQLError{Number: 1290, Message: "The MySQL server is running with the --read-only option"}
}

// WaitForCondition waits for a condition to be true with timeout
func (h *TestHelper) WaitForCondition(condition func() bool, timeout time.Duration, message string) {
	start := time.Now()
	for time.Since(start) < timeout {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.t.Fatalf("Condition not met within timeout: %s", message)
}

// AssertRowCount asserts the number of rows in a table
func (h *TestHelper) AssertRowCount(tableName string, expected int) {
	actual := h.CountRows(tableName)
	if actual != expected {
		h.t.Fatalf("Expected %d rows in %s, got %d", expected, tableName, actual)
	}
}

// AssertQueryResult asserts that a query returns expected results
func (h *TestHelper) AssertQueryResult(query string, expectedRows int, args ...any) {
	rows := h.QueryData(query, args...)
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		count++
	}
	
	if count != expectedRows {
		h.t.Fatalf("Expected %d rows from query, got %d", expectedRows, count)
	}
}
