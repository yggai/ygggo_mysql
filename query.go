package ygggo_mysql

import (
	"context"
)

// Exec executes a statement (placeholder).
func (c *Conn) Exec(ctx context.Context, query string, args ...any) (any, error) { return nil, nil }

// Query runs a query and returns rows (placeholder).
func (c *Conn) Query(ctx context.Context, query string, args ...any) (any, error) { return nil, nil }

// QueryRow runs a query and returns a single row (placeholder).
func (c *Conn) QueryRow(ctx context.Context, query string, args ...any) any { return nil }

// QueryStream streams rows via callback (placeholder).
func (c *Conn) QueryStream(ctx context.Context, query string, cb func(any) error, args ...any) error { return nil }

