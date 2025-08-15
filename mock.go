package ygggo_mysql

import (
	"context"

	"github.com/DATA-DOG/go-sqlmock"
)

// NewMockPool creates a Pool backed by sqlmock for testing.
// Returns the Pool, the sqlmock.Sqlmock for setting expectations, and any error.
func NewMockPool(ctx context.Context, cfg Config) (*Pool, sqlmock.Sqlmock, error) {
	// Apply env overrides first (convention over configuration)
	applyEnv(&cfg)
	
	// Create sqlmock DB
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		return nil, nil, err
	}
	
	p := &Pool{db: db}
	// Apply retry policy from config
	p.retry = cfg.Retry
	
	// Apply pool settings (though they may not be meaningful for mock)
	if cfg.Pool.MaxOpen > 0 {
		db.SetMaxOpenConns(cfg.Pool.MaxOpen)
	}
	if cfg.Pool.MaxIdle > 0 {
		db.SetMaxIdleConns(cfg.Pool.MaxIdle)
	}
	if cfg.Pool.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.Pool.ConnMaxLifetime)
	}
	if cfg.Pool.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.Pool.ConnMaxIdleTime)
	}
	
	return p, mock, nil
}

// NewPoolWithMock creates either a real Pool or mock Pool based on isMock flag.
func NewPoolWithMock(ctx context.Context, cfg Config, isMock bool) (*Pool, sqlmock.Sqlmock, error) {
	if isMock {
		return NewMockPool(ctx, cfg)
	}
	pool, err := NewPool(ctx, cfg)
	return pool, nil, err
}
