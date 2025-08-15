package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteConfig holds SQLite-specific configuration
type SQLiteConfig struct {
	// Database file path, use ":memory:" for in-memory database
	Path string
	
	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	
	// SQLite-specific settings
	BusyTimeout time.Duration
	JournalMode string // WAL, DELETE, TRUNCATE, PERSIST, MEMORY, OFF
	Synchronous string // FULL, NORMAL, OFF
	CacheSize   int    // Number of pages in cache
}

// DefaultSQLiteConfig returns a default SQLite configuration
func DefaultSQLiteConfig() SQLiteConfig {
	return SQLiteConfig{
		Path:            ":memory:",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
		BusyTimeout:     5 * time.Second,
		JournalMode:     "WAL",
		Synchronous:     "NORMAL",
		CacheSize:       2000,
	}
}

// NewSQLitePool creates a new database pool using SQLite
func NewSQLitePool(ctx context.Context, configs ...SQLiteConfig) (*Pool, error) {
	var config SQLiteConfig
	if len(configs) > 0 {
		config = configs[0]
	} else {
		config = DefaultSQLiteConfig()
	}
	
	// Build DSN
	dsn := buildSQLiteDSN(config)
	
	// Open database
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	
	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}
	
	// Create pool
	pool := &Pool{
		db: db,
		retry: RetryPolicy{
			MaxAttempts: 3,
			BaseBackoff: 100 * time.Millisecond,
			MaxBackoff:  time.Second,
			MaxElapsed:  30 * time.Second,
			Jitter:      true,
		},
	}
	
	return pool, nil
}

// buildSQLiteDSN builds a SQLite DSN string from config
func buildSQLiteDSN(config SQLiteConfig) string {
	dsn := config.Path
	
	// Add query parameters
	params := make(map[string]string)
	
	if config.BusyTimeout > 0 {
		params["_busy_timeout"] = fmt.Sprintf("%d", config.BusyTimeout.Milliseconds())
	}
	
	if config.JournalMode != "" {
		params["_journal_mode"] = config.JournalMode
	}
	
	if config.Synchronous != "" {
		params["_synchronous"] = config.Synchronous
	}
	
	if config.CacheSize > 0 {
		params["_cache_size"] = fmt.Sprintf("%d", config.CacheSize)
	}
	
	// Enable foreign keys by default
	params["_foreign_keys"] = "on"
	
	// Build query string
	if len(params) > 0 {
		dsn += "?"
		first := true
		for key, value := range params {
			if !first {
				dsn += "&"
			}
			dsn += key + "=" + value
			first = false
		}
	}
	
	return dsn
}

// NewSQLiteTestPool creates a new in-memory SQLite pool for testing
func NewSQLiteTestPool(ctx context.Context) (*Pool, error) {
	config := SQLiteConfig{
		Path:            ":memory:",
		MaxOpenConns:    1, // Use single connection for in-memory to maintain state
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
		BusyTimeout:     time.Second,
		JournalMode:     "MEMORY",
		Synchronous:     "OFF", // Faster for testing
		CacheSize:       1000,
	}
	
	return NewSQLitePool(ctx, config)
}

// SQLiteTestHelper provides utilities for SQLite testing
type SQLiteTestHelper struct {
	pool *Pool
}

// NewSQLiteTestHelper creates a new test helper
func NewSQLiteTestHelper(ctx context.Context) (*SQLiteTestHelper, error) {
	pool, err := NewSQLiteTestPool(ctx)
	if err != nil {
		return nil, err
	}
	
	return &SQLiteTestHelper{pool: pool}, nil
}

// Pool returns the underlying pool
func (h *SQLiteTestHelper) Pool() *Pool {
	return h.pool
}

// Close closes the test helper and pool
func (h *SQLiteTestHelper) Close() error {
	if h.pool != nil {
		return h.pool.Close()
	}
	return nil
}

// CreateTable creates a table with the given schema
func (h *SQLiteTestHelper) CreateTable(ctx context.Context, tableName, schema string) error {
	return h.pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, fmt.Sprintf("CREATE TABLE %s (%s)", tableName, schema))
		return err
	})
}

// InsertData inserts data into a table
func (h *SQLiteTestHelper) InsertData(ctx context.Context, query string, args ...any) error {
	return h.pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.Exec(ctx, query, args...)
		return err
	})
}

// QueryData queries data from the database
func (h *SQLiteTestHelper) QueryData(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	var rows *sql.Rows
	err := h.pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		rows = rs
		return nil
	})
	return rows, err
}

// CountRows counts rows in a table
func (h *SQLiteTestHelper) CountRows(ctx context.Context, tableName string) (int, error) {
	var count int
	err := h.pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName))
		if err != nil {
			return err
		}
		defer rs.Close()
		
		if rs.Next() {
			err = rs.Scan(&count)
			if err != nil {
				return err
			}
		}
		return rs.Err()
	})
	return count, err
}

// SetupUsersTable creates a standard users table for testing
func (h *SQLiteTestHelper) SetupUsersTable(ctx context.Context) error {
	return h.CreateTable(ctx, "users", `
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	`)
}

// InsertUser inserts a user and returns the ID
func (h *SQLiteTestHelper) InsertUser(ctx context.Context, name, email string) (int64, error) {
	var id int64
	err := h.pool.WithConn(ctx, func(c DatabaseConn) error {
		result, err := c.Exec(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", name, email)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	return id, err
}

// GetUser retrieves a user by ID
func (h *SQLiteTestHelper) GetUser(ctx context.Context, id int64) (name, email string, err error) {
	err = h.pool.WithConn(ctx, func(c DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT name, email FROM users WHERE id = ?", id)
		if err != nil {
			return err
		}
		defer rs.Close()
		
		if rs.Next() {
			err = rs.Scan(&name, &email)
			if err != nil {
				return err
			}
		}
		return rs.Err()
	})
	return name, email, err
}
