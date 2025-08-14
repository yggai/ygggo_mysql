package ygggo_mysql

import (
	"context"
	"database/sql"
)

// Exec executes a statement using the underlying connection.
func (c *Conn) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if c == nil || c.inner == nil { return nil, sql.ErrConnDone }
	return c.inner.ExecContext(ctx, query, args...)
}

// Query runs a query and returns rows.
func (c *Conn) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if c == nil || c.inner == nil { return nil, sql.ErrConnDone }
	return c.inner.QueryContext(ctx, query, args...)
}

// QueryRow runs a query and returns a single row.
func (c *Conn) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if c == nil || c.inner == nil { return &sql.Row{} }
	return c.inner.QueryRowContext(ctx, query, args...)
}

// QueryStream streams rows via callback; cb receives []any per row.
func (c *Conn) QueryStream(ctx context.Context, query string, cb func([]any) error, args ...any) error {
	rs, err := c.Query(ctx, query, args...)
	if err != nil { return err }
	defer rs.Close()
	cols, err := rs.Columns()
	if err != nil { return err }
	buf := make([]any, len(cols))
	scan := make([]any, len(cols))
	for i := range buf { scan[i] = &buf[i] }
	for rs.Next() {
		if err := rs.Scan(scan...); err != nil { return err }
		if err := cb(buf); err != nil { return err }
	}
	return rs.Err()
}
