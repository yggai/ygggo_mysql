package ygggo_mysql

import "context"

// Pool is a placeholder wrapper over a future *sql.DB based pool.
type Pool struct{}

// NewPool creates a new Pool (placeholder).
func NewPool(ctx context.Context, cfg Config) (*Pool, error) {
	return &Pool{}, nil
}

// Close closes the pool (placeholder).
func (p *Pool) Close() error { return nil }

