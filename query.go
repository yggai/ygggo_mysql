package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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


// BulkInsert inserts multiple rows using a single multi-values INSERT.
// table: table name; columns: column names; rows: len(rows) > 0 and each len == len(columns)
func (c *Conn) BulkInsert(ctx context.Context, table string, columns []string, rows [][]any) (sql.Result, error) {
	if c == nil || c.inner == nil { return nil, sql.ErrConnDone }
	if len(rows) == 0 { return nil, fmt.Errorf("no rows to insert") }
	colN := len(columns)
	for i, r := range rows {
		if len(r) != colN { return nil, fmt.Errorf("row %d has %d values, want %d", i, len(r), colN) }
	}
	placeOne := "(" + strings.TrimRight(strings.Repeat("?,", colN), ",") + ")"
	var b strings.Builder
	b.Grow(64 + len(rows)*len(placeOne))
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	b.WriteString(strings.Join(columns, ","))
	b.WriteString(") VALUES ")
	args := make([]any, 0, len(rows)*colN)
	for i, r := range rows {
		if i > 0 { b.WriteString(",") }
		b.WriteString(placeOne)
		args = append(args, r...)
	}
	return c.Exec(ctx, b.String(), args...)
}

// InsertOnDuplicate is BulkInsert with ON DUPLICATE KEY UPDATE for the given updateCols.
func (c *Conn) InsertOnDuplicate(ctx context.Context, table string, columns []string, rows [][]any, updateCols []string) (sql.Result, error) {
	if len(updateCols) == 0 { return c.BulkInsert(ctx, table, columns, rows) }
	if c == nil || c.inner == nil { return nil, sql.ErrConnDone }
	if len(rows) == 0 { return nil, fmt.Errorf("no rows to insert") }
	colN := len(columns)
	for i, r := range rows {
		if len(r) != colN { return nil, fmt.Errorf("row %d has %d values, want %d", i, len(r), colN) }
	}
	placeOne := "(" + strings.TrimRight(strings.Repeat("?,", colN), ",") + ")"
	var b strings.Builder
	b.Grow(64 + len(rows)*len(placeOne))
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	b.WriteString(strings.Join(columns, ","))
	b.WriteString(") VALUES ")
	args := make([]any, 0, len(rows)*colN)
	for i, r := range rows {
		if i > 0 { b.WriteString(",") }
		b.WriteString(placeOne)
		args = append(args, r...)
	}
	b.WriteString(" ON DUPLICATE KEY UPDATE ")
	for i, col := range updateCols {
		if i > 0 { b.WriteString(",") }
		b.WriteString(col)
		b.WriteString("=VALUES(")
		b.WriteString(col)
		b.WriteString(")")
	}
	return c.Exec(ctx, b.String(), args...)
}
