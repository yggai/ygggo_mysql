package ygggo_mysql

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"
)

// Pool is a placeholder wrapper over *sql.DB based pool.
type Pool struct {
	db *sql.DB
	// leak detection
	borrowWarnNS int64        // threshold in ns; 0 means disabled
	borrowed     int64        // current borrowed count
	leakHandler  atomic.Value // func(BorrowLeak)
}

// SetBorrowWarnThreshold sets the warn threshold for held connections.
func (p *Pool) SetBorrowWarnThreshold(d time.Duration) { atomic.StoreInt64(&p.borrowWarnNS, d.Nanoseconds()) }

// SetLeakHandler registers a callback invoked when a borrow exceeds threshold.
func (p *Pool) SetLeakHandler(h func(BorrowLeak)) { p.leakHandler.Store(h) }

func (p *Pool) onBorrow(acqNS int64) {
	atomic.AddInt64(&p.borrowed, 1)
	thr := atomic.LoadInt64(&p.borrowWarnNS)
	if thr <= 0 { return }
	if h, _ := p.leakHandler.Load().(func(BorrowLeak)); h != nil {
		// schedule async watchdog
		go func(start int64) {
			t := time.NewTimer(time.Duration(thr))
			defer t.Stop()
			<-t.C
			// If still borrowed (best-effort), signal
			if atomic.LoadInt64(&p.borrowed) > 0 {
				h(BorrowLeak{HeldFor: time.Duration(time.Now().UnixNano() - start)})
			}
		}(acqNS)
	}
}

func (p *Pool) onReturn() {
	atomic.AddInt64(&p.borrowed, -1)
}

// NewPool creates a new Pool, opening a DB and applying basic pool settings.
func NewPool(ctx context.Context, cfg Config) (*Pool, error) {
	// Apply env overrides first (convention over configuration)
	applyEnv(&cfg)
	if cfg.Driver == "" {
		cfg.Driver = "mysql"
	}
	// Build DSN from config (supports raw DSN or field-based build)
	dsn, err := dsnFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	// Open DB
	db, err := sql.Open(cfg.Driver, dsn)
	if err != nil {
		return nil, err
	}
	p := &Pool{db: db}
	// Apply pool settings (placeholders)
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
	// Try ping to validate connectivity
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return p, nil
}

// Close closes the pool (placeholder).
func (p *Pool) Close() error {
	if p == nil || p.db == nil {
		return nil
	}
	return p.db.Close()
}

// Ping checks connectivity (placeholder).
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.db == nil {
		return errors.New("nil pool")
	}
	return p.db.PingContext(ctx)
}

// SelfCheck performs a basic health check (placeholder).
func (p *Pool) SelfCheck(ctx context.Context) error {
	return p.Ping(ctx)
}
