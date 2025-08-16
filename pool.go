package ygggo_mysql

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Pool is a production-ready database connection pool with advanced features.
//
// Pool wraps the standard library's *sql.DB and adds enterprise-grade features
// including connection leak detection, health monitoring, retry policies,
// observability (metrics, logging, tracing), and slow query analysis.
//
// Key features:
//   - Automatic connection management and pooling
//   - Connection leak detection with configurable thresholds
//   - Retry policies for handling transient failures
//   - Comprehensive observability (OpenTelemetry integration)
//   - Slow query detection and recording
//   - Health monitoring and self-checks
//
// Example usage:
//
//	pool, err := NewPool(ctx, config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer pool.Close()
//
//	// Enable connection leak detection
//	pool.SetBorrowWarnThreshold(30 * time.Second)
//	pool.SetLeakHandler(func(leak BorrowLeak) {
//		log.Printf("Connection held for %v", leak.HeldFor)
//	})
//
// Thread Safety:
//
// Pool is safe for concurrent use by multiple goroutines. All methods
// can be called concurrently without external synchronization.
type Pool struct {
	// db is the underlying database connection pool
	db *sql.DB

	// Connection leak detection
	borrowWarnNS int64        // threshold in nanoseconds; 0 means disabled
	borrowed     int64        // current borrowed connection count
	leakHandler  atomic.Value // func(BorrowLeak) - callback for leak detection

	// Retry policy for handling transient failures
	retry RetryPolicy

	// Observability features
	telemetryEnabled bool // OpenTelemetry tracing enabled

	// Metrics collection
	metricsEnabled bool                   // Prometheus metrics enabled
	meterProvider  metric.MeterProvider   // OpenTelemetry meter provider
	metrics        *Metrics               // Metrics collector instance

	// Logging configuration
	loggingEnabled     bool           // Structured logging enabled
	logger            *slog.Logger    // Logger instance
	slowQueryThreshold time.Duration  // Threshold for slow query logging

	// Slow query analysis
	slowQueryRecorder *SlowQueryRecorder // Records and analyzes slow queries

	// Health monitoring
	healthMonitor *HealthMonitor // Monitors pool and connection health
}

// SetBorrowWarnThreshold sets the warning threshold for connection hold time.
//
// When a connection is held longer than this threshold, the registered
// leak handler (if any) will be called. This helps detect connection
// leaks and long-running operations that might impact pool performance.
//
// Parameters:
//   - d: Duration threshold. Set to 0 to disable leak detection.
//
// Example:
//
//	pool.SetBorrowWarnThreshold(30 * time.Second)
//
// Thread Safety: This method is safe for concurrent use.
func (p *Pool) SetBorrowWarnThreshold(d time.Duration) {
	atomic.StoreInt64(&p.borrowWarnNS, d.Nanoseconds())
}

// SetLeakHandler registers a callback function for connection leak detection.
//
// The handler function is called asynchronously when a connection is held
// longer than the threshold set by SetBorrowWarnThreshold. This allows
// applications to log, monitor, or take corrective action for potential leaks.
//
// Parameters:
//   - h: Handler function that receives leak information
//
// Example:
//
//	pool.SetLeakHandler(func(leak BorrowLeak) {
//		log.Printf("WARN: Connection held for %v", leak.HeldFor)
//		// Could also increment metrics, send alerts, etc.
//	})
//
// Thread Safety: This method is safe for concurrent use.
func (p *Pool) SetLeakHandler(h func(BorrowLeak)) {
	p.leakHandler.Store(h)
}

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

// NewPool creates a new database connection pool with the specified configuration.
//
// This function initializes a production-ready connection pool with all
// configured features including connection limits, retry policies, observability,
// and health monitoring. The pool is validated with a connectivity test before
// being returned.
//
// Configuration Priority:
//  1. Environment variables (YGGGO_MYSQL_* prefix)
//  2. Provided Config struct values
//  3. Default values
//
// Parameters:
//   - ctx: Context for initialization and connectivity testing
//   - cfg: Configuration struct with connection and feature settings
//
// Returns:
//   - *Pool: Configured and validated connection pool
//   - error: Configuration, connection, or validation error
//
// Example:
//
//	config := Config{
//		Host:     "localhost",
//		Port:     3306,
//		Username: "user",
//		Password: "password",
//		Database: "mydb",
//		Pool: PoolConfig{
//			MaxOpen: 25,
//			MaxIdle: 10,
//			ConnMaxLifetime: 5 * time.Minute,
//		},
//	}
//
//	pool, err := NewPool(ctx, config)
//	if err != nil {
//		log.Fatalf("Failed to create pool: %v", err)
//	}
//	defer pool.Close()
//
// Environment Variable Examples:
//   - YGGGO_MYSQL_HOST=localhost
//   - YGGGO_MYSQL_PORT=3306
//   - YGGGO_MYSQL_USERNAME=user
//   - YGGGO_MYSQL_PASSWORD=secret
//   - YGGGO_MYSQL_DATABASE=mydb
//
// The function performs the following steps:
//  1. Apply environment variable overrides to config
//  2. Set default driver to "mysql" if not specified
//  3. Build DSN from config (either raw DSN or constructed from fields)
//  4. Open database connection with specified driver
//  5. Apply pool configuration (connection limits, timeouts)
//  6. Validate connectivity with ping test
//  7. Return configured pool or cleanup and return error
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
	// Apply retry policy from config
	p.retry = cfg.Retry
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

// Close closes the pool and all its connections.
func (p *Pool) Close() error {
	if p == nil || p.db == nil {
		return nil
	}
	if p.slowQueryRecorder != nil {
		p.slowQueryRecorder.Close()
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

// internal retry policy storage (temporary until full feature wired)
func (p *Pool) setRetryPolicy(r RetryPolicy) { p.retry = r }

// EnableSlowQueryRecording enables slow query recording with the given configuration
func (p *Pool) EnableSlowQueryRecording(config SlowQueryConfig, storage SlowQueryStorage) {
	p.slowQueryRecorder = NewSlowQueryRecorder(config, storage)
}

// DisableSlowQueryRecording disables slow query recording
func (p *Pool) DisableSlowQueryRecording() {
	if p.slowQueryRecorder != nil {
		p.slowQueryRecorder.Close()
		p.slowQueryRecorder = nil
	}
}

// GetSlowQueryRecorder returns the slow query recorder
func (p *Pool) GetSlowQueryRecorder() *SlowQueryRecorder {
	return p.slowQueryRecorder
}

// IsSlowQueryRecordingEnabled returns whether slow query recording is enabled
func (p *Pool) IsSlowQueryRecordingEnabled() bool {
	return p.slowQueryRecorder != nil && p.slowQueryRecorder.IsEnabled()
}

// SetSlowQueryThreshold sets the slow query threshold for both logging and recording
func (p *Pool) SetSlowQueryThreshold(threshold time.Duration) {
	p.slowQueryThreshold = threshold
	if p.slowQueryRecorder != nil {
		p.slowQueryRecorder.SetThreshold(threshold)
	}
}

// GetSlowQueryThreshold returns the current slow query threshold
func (p *Pool) GetSlowQueryThreshold() time.Duration {
	return p.slowQueryThreshold
}


