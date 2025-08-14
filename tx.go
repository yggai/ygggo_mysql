package ygggo_mysql

import "context"

// Tx represents a placeholder transaction.
type Tx struct{}

// WithinTx executes a function within a transaction (placeholder).
func (p *Pool) WithinTx(ctx context.Context, fn func(*Tx) error, opts ...any) error {
	return fn(&Tx{})
}

