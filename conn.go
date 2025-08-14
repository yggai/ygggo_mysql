package ygggo_mysql

import "context"

// Conn is a placeholder for a borrowed connection.
type Conn struct{}

// WithConn runs a function with a connection (placeholder).
func (p *Pool) WithConn(ctx context.Context, fn func(*Conn) error) error {
	return fn(&Conn{})
}

// Acquire acquires a connection (placeholder).
func (p *Pool) Acquire(ctx context.Context) (*Conn, error) { return &Conn{}, nil }

// Close releases the connection (placeholder).
func (c *Conn) Close() error { return nil }

