package ygggo_mysql

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"
)

// BorrowLeak contains information about a connection that has been held too long.
//
// This structure is passed to leak handler callbacks when a connection
// exceeds the configured warning threshold. It provides diagnostic
// information to help identify potential connection leaks.
type BorrowLeak struct {
	// HeldFor is the duration the connection has been held.
	//
	// This represents the time elapsed since the connection was
	// acquired from the pool until the leak detection triggered.
	HeldFor time.Duration
}

// Conn represents a single database connection obtained from the connection pool.
//
// Conn wraps the standard library's *sql.Conn and adds additional features
// like prepared statement caching, leak detection, and observability integration.
// Each connection must be properly closed to return it to the pool and prevent
// connection leaks.
//
// Key features:
//   - Automatic connection lifecycle management
//   - Optional prepared statement caching for performance
//   - Integration with pool-level observability features
//   - Connection leak detection and monitoring
//
// Example usage:
//
//	conn, err := pool.Acquire(ctx)
//	if err != nil {
//		return err
//	}
//	defer conn.Close() // Always close to return to pool
//
//	result, err := conn.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
//
// Preferred usage with automatic management:
//
//	err := pool.WithConn(ctx, func(conn DatabaseConn) error {
//		return conn.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
//	}) // Connection automatically returned to pool
//
// Thread Safety:
//
// Conn is NOT safe for concurrent use. Each connection should be used
// by only one goroutine at a time. Use separate connections for concurrent
// operations.
type Conn struct{
	// inner is the underlying database connection
	inner *sql.Conn

	// p is a reference to the parent pool for observability features
	p *Pool

	// acqNS is the monotonic acquisition time in nanoseconds for leak detection
	acqNS int64

	// cache is an optional per-connection prepared statement cache
	cache *stmtCache
}

// WithConn executes a function with an automatically managed database connection.
//
// This is the recommended way to perform database operations as it handles
// connection acquisition and cleanup automatically. The connection is acquired
// from the pool, passed to the provided function, and always returned to the
// pool regardless of whether the function succeeds or fails.
//
// Benefits of using WithConn:
//   - Automatic connection management (no risk of leaks)
//   - Consistent error handling patterns
//   - Integration with pool-level observability features
//   - Simplified code with proper resource cleanup
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and tracing
//   - fn: Function to execute with the connection
//
// Returns:
//   - error: Connection acquisition error or error returned by fn
//
// Example:
//
//	err := pool.WithConn(ctx, func(conn DatabaseConn) error {
//		// Perform database operations
//		result, err := conn.Exec(ctx, "INSERT INTO users (name, email) VALUES (?, ?)",
//			"Alice", "alice@example.com")
//		if err != nil {
//			return err
//		}
//
//		id, err := result.LastInsertId()
//		if err != nil {
//			return err
//		}
//
//		log.Printf("Created user with ID: %d", id)
//		return nil
//	})
//	if err != nil {
//		log.Printf("Database operation failed: %v", err)
//	}
//
// Error Handling:
//
// If fn returns an error, that error is returned by WithConn. The connection
// is still properly returned to the pool. If connection acquisition fails,
// fn is not called and the acquisition error is returned.
//
// Thread Safety:
//
// This method is safe for concurrent use. Multiple goroutines can call
// WithConn simultaneously, and each will receive its own connection.
func (p *Pool) WithConn(ctx context.Context, fn func(DatabaseConn) error) error {
	conn, err := p.Acquire(ctx)
	if err != nil { return err }
	defer conn.Close()
	return fn(conn)
}

// EnableStmtCache enables per-connection LRU stmt cache with the given capacity.
func (c *Conn) EnableStmtCache(capacity int) { c.cache = newStmtCache(capacity) }

// ExecCached executes using a cached prepared statement when enabled.
func (c *Conn) ExecCached(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if c == nil || c.inner == nil { return nil, sql.ErrConnDone }
	if c.cache == nil { return c.Exec(ctx, query, args...) }
	st, _, err := c.cache.getOrPrepare(ctx, c.inner, query)
	if err != nil { return nil, err }
	return st.ExecContext(ctx, args...)
}

// QueryCached runs a query using stmt cache when enabled.
func (c *Conn) QueryCached(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if c == nil || c.inner == nil { return nil, sql.ErrConnDone }
	if c.cache == nil { return c.Query(ctx, query, args...) }
	st, _, err := c.cache.getOrPrepare(ctx, c.inner, query)
	if err != nil { return nil, err }
	return st.QueryContext(ctx, args...)
}

// Acquire gets a connection from the underlying *sql.DB honoring context.
func (p *Pool) Acquire(ctx context.Context) (DatabaseConn, error) {
	if p == nil || p.db == nil { return nil, errors.New("nil pool") }
	c, err := p.db.Conn(ctx)
	if err != nil { return nil, err }
	conn := &Conn{inner: c, p: p}
	conn.markAcquired()
	// Record connection acquisition for metrics
	if p.metricsEnabled {
		p.recordConnectionAcquired(ctx)
	}
	return conn, nil
}

func (c *Conn) markAcquired() {
	if c == nil || c.p == nil { return }
	ns := time.Now().UnixNano()
	atomic.StoreInt64(&c.acqNS, ns)
	c.p.onBorrow(ns)
}

// Close returns the connection to the pool.
func (c *Conn) Close() error {
	if c == nil || c.inner == nil { return nil }

	// TODO: Re-enable connection metrics after fixing deadlock issues
	// if c.p != nil && c.p.metricsEnabled && c.acqNS > 0 {
	//     duration := time.Duration(time.Now().UnixNano() - c.acqNS)
	//     c.p.recordConnectionReleased(context.Background(), duration)
	// }

	c.p.onReturn()
	if c.cache != nil { c.cache.closeAll() }
	return c.inner.Close()
}

