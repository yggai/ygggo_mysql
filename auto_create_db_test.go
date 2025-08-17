package ygggo_mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Enhanced fake driver that can simulate database operations
type enhancedFakeDriver struct {
	databases map[string]bool // Track which databases "exist"
}

type enhancedFakeConn struct {
	driver *enhancedFakeDriver
}

func (d *enhancedFakeDriver) Open(name string) (driver.Conn, error) {
	return &enhancedFakeConn{driver: d}, nil
}

// Implement driver.Conn
func (c *enhancedFakeConn) Prepare(query string) (driver.Stmt, error) {
	return &enhancedFakeStmt{conn: c, query: query}, nil
}
func (c *enhancedFakeConn) Close() error              { return nil }
func (c *enhancedFakeConn) Begin() (driver.Tx, error) { return &enhancedFakeTx{}, nil }

// Implement driver.Pinger so db.Ping() succeeds
func (c *enhancedFakeConn) Ping(ctx context.Context) error { return nil }

// Implement driver.SessionResetter
func (c *enhancedFakeConn) ResetSession(ctx context.Context) error { return nil }

// Implement driver.Queryer for direct queries
func (c *enhancedFakeConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	return c.query(query, args)
}

// Implement driver.Execer for direct exec
func (c *enhancedFakeConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	return c.exec(query, args)
}

// Implement driver.QueryerContext
func (c *enhancedFakeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	// Convert NamedValue to Value
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.query(query, values)
}

// Implement driver.ExecerContext
func (c *enhancedFakeConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	// Convert NamedValue to Value
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return c.exec(query, values)
}

func (c *enhancedFakeConn) query(query string, args []driver.Value) (driver.Rows, error) {
	query = strings.TrimSpace(strings.ToUpper(query))

	if strings.Contains(query, "SHOW DATABASES") || strings.Contains(query, "INFORMATION_SCHEMA.SCHEMATA") {
		// Check if we're looking for a specific database
		if len(args) > 0 {
			targetDB := fmt.Sprintf("%v", args[0])
			if c.driver.databases[targetDB] {
				return &enhancedFakeRows{databases: []string{targetDB}, index: -1}, nil
			} else {
				return &enhancedFakeRows{databases: []string{}, index: -1}, nil
			}
		}

		// Return list of all existing databases
		var databases []string
		for db := range c.driver.databases {
			databases = append(databases, db)
		}
		return &enhancedFakeRows{databases: databases, index: -1}, nil
	}

	return &enhancedFakeRows{databases: []string{}, index: -1}, nil
}

func (c *enhancedFakeConn) exec(query string, args []driver.Value) (driver.Result, error) {
	originalQuery := query
	query = strings.TrimSpace(strings.ToUpper(query))

	if strings.HasPrefix(query, "CREATE DATABASE") {
		// Extract database name from original query (preserve case)
		parts := strings.Fields(originalQuery)
		if len(parts) >= 3 {
			dbName := strings.Trim(parts[2], "`'\"")
			if strings.Contains(strings.ToUpper(dbName), "IF") && len(parts) >= 6 {
				dbName = strings.Trim(parts[5], "`'\"")
			}

			c.driver.databases[dbName] = true
		}
		return &enhancedFakeResult{}, nil
	}

	return &enhancedFakeResult{}, nil
}

// Enhanced fake statement
type enhancedFakeStmt struct {
	conn  *enhancedFakeConn
	query string
}

func (s *enhancedFakeStmt) Close() error  { return nil }
func (s *enhancedFakeStmt) NumInput() int { return 0 }
func (s *enhancedFakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.conn.exec(s.query, args)
}
func (s *enhancedFakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.query(s.query, args)
}

// Implement driver.StmtQueryContext
func (s *enhancedFakeStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	// Convert NamedValue to Value
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return s.conn.query(s.query, values)
}

// Implement driver.StmtExecContext
func (s *enhancedFakeStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	// Convert NamedValue to Value
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return s.conn.exec(s.query, values)
}

// Enhanced fake rows
type enhancedFakeRows struct {
	databases []string
	index     int
}

func (r *enhancedFakeRows) Columns() []string { return []string{"Database"} }
func (r *enhancedFakeRows) Close() error      { return nil }
func (r *enhancedFakeRows) Next(dest []driver.Value) error {
	r.index++
	if r.index >= len(r.databases) {
		return io.EOF
	}
	if len(dest) > 0 {
		dest[0] = r.databases[r.index]
	}
	return nil
}

// Enhanced fake result
type enhancedFakeResult struct{}

func (r *enhancedFakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r *enhancedFakeResult) RowsAffected() (int64, error) { return 1, nil }

// Enhanced fake transaction
type enhancedFakeTx struct{}

func (tx *enhancedFakeTx) Commit() error   { return nil }
func (tx *enhancedFakeTx) Rollback() error { return nil }

// Global enhanced fake driver instance
var enhancedFakeDriverInstance = &enhancedFakeDriver{
	databases: make(map[string]bool),
}

// Register the enhanced fake driver
func init() {
	sql.Register("enhanced_fake", enhancedFakeDriverInstance)
}

func TestAutoCreateDatabase_DatabaseNotExists(t *testing.T) {
	ctx := context.Background()

	// Reset driver state
	enhancedFakeDriverInstance.databases = make(map[string]bool)

	// Set environment variables for a database that doesn't exist
	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "localhost")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "password")
	t.Setenv("YGGGO_MYSQL_DATABASE", "testdb")

	// Verify database doesn't exist initially
	if enhancedFakeDriverInstance.databases["testdb"] {
		t.Fatal("testdb should not exist initially")
	}

	pool, err := NewPoolEnv(ctx)
	if err != nil {
		t.Fatalf("NewPoolEnv should succeed and auto-create database: %v", err)
	}
	defer pool.Close()

	// Verify database was created
	if !enhancedFakeDriverInstance.databases["testdb"] {
		t.Fatal("testdb should have been auto-created")
	}
}

func TestAutoCreateDatabase_DatabaseExists(t *testing.T) {
	ctx := context.Background()

	// Reset driver state and pre-create database
	enhancedFakeDriverInstance.databases = map[string]bool{
		"existingdb": true,
	}

	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "localhost")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "password")
	t.Setenv("YGGGO_MYSQL_DATABASE", "existingdb")

	pool, err := NewPoolEnv(ctx)
	if err != nil {
		t.Fatalf("NewPoolEnv should succeed when database exists: %v", err)
	}
	defer pool.Close()

	// Database should still exist (not recreated)
	if !enhancedFakeDriverInstance.databases["existingdb"] {
		t.Fatal("existingdb should still exist")
	}
}

func TestAutoCreateDatabase_NoDatabaseSpecified(t *testing.T) {
	ctx := context.Background()

	// Reset driver state
	enhancedFakeDriverInstance.databases = make(map[string]bool)

	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "localhost")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "password")
	// No database specified

	pool, err := NewPoolEnv(ctx)
	if err != nil {
		t.Fatalf("NewPoolEnv should succeed when no database specified: %v", err)
	}
	defer pool.Close()

	// No databases should be created
	if len(enhancedFakeDriverInstance.databases) > 0 {
		t.Fatal("No databases should be created when none specified")
	}
}

func TestAutoCreateDatabase_WithDSNFields(t *testing.T) {
	ctx := context.Background()

	// Reset driver state
	enhancedFakeDriverInstance.databases = make(map[string]bool)

	// Set environment variables for field-based DSN construction with database
	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "p@ss:word/!")
	t.Setenv("YGGGO_MYSQL_DATABASE", "auto_created_db")
	t.Setenv("YGGGO_MYSQL_PARAMS", "parseTime=true&loc=Local")

	// Verify database doesn't exist initially
	if enhancedFakeDriverInstance.databases["auto_created_db"] {
		t.Fatal("auto_created_db should not exist initially")
	}

	pool, err := NewPoolEnv(ctx)
	if err != nil {
		t.Fatalf("NewPoolEnv should succeed and auto-create database: %v", err)
	}
	defer pool.Close()

	// Verify database was created
	if !enhancedFakeDriverInstance.databases["auto_created_db"] {
		t.Fatal("auto_created_db should have been auto-created")
	}

	// Verify DSN contains the database name
	dsn := GetDSN()
	if !strings.Contains(dsn, "auto_created_db") {
		t.Fatalf("DSN should contain database name, got: %s", dsn)
	}
}
