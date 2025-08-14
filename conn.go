package ygggo_mysql

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"
)

// BorrowLeak carries info about a long-held connection.
type BorrowLeak struct {
	HeldFor time.Duration
}

// Conn wraps a single connection obtained from *sql.DB
// It must be closed to return the connection back to the pool.
type Conn struct{
	inner *sql.Conn
	p     *Pool
	acqNS int64 // monotonic acquisition time (ns)
	// per-connection stmt cache (optional)
	cache *stmtCache
}

// WithConn acquires a connection, calls fn, and always returns the connection.
func (p *Pool) WithConn(ctx context.Context, fn func(*Conn) error) error {
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
func (p *Pool) Acquire(ctx context.Context) (*Conn, error) {
	if p == nil || p.db == nil { return nil, errors.New("nil pool") }
	c, err := p.db.Conn(ctx)
	if err != nil { return nil, err }
	conn := &Conn{inner: c, p: p}
	conn.markAcquired()
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
	c.p.onReturn()
	if c.cache != nil { c.cache.closeAll() }
	return c.inner.Close()
}

