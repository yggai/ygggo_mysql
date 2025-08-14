package ygggo_mysql

import "context"

// Get scans one row into dest (placeholder).
func Get[T any](ctx context.Context, c *Conn, dest *T, query string, args ...any) error { return nil }

// Select scans rows into dest slice (placeholder).
func Select[T any](ctx context.Context, c *Conn, dest *[]T, query string, args ...any) error { return nil }

