package ygggo_mysql

import (
	"context"
	"log/slog"
	"os"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Enabled            bool
	SlowQueryThreshold time.Duration
	Level              slog.Level
}

var (
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
)

// EnableLogging enables or disables structured logging for this pool
func (p *Pool) EnableLogging(enabled bool) {
	if p == nil { return }
	p.loggingEnabled = enabled
	if enabled && p.logger == nil {
		p.logger = defaultLogger
	}
}

// SetLogger sets a custom logger for this pool
func (p *Pool) SetLogger(logger *slog.Logger) {
	if p == nil { return }
	p.logger = logger
}

// Note: SetSlowQueryThreshold is now defined in pool.go

// logQuery logs database query execution with structured fields
func (p *Pool) logQuery(ctx context.Context, operation, query string, args []any, duration time.Duration, err error) {
	if p == nil || !p.loggingEnabled || p.logger == nil { return }

	// Prepare log attributes
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("query", query),
		slog.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
	}

	// Add arguments if present (be careful with sensitive data)
	if len(args) > 0 {
		attrs = append(attrs, slog.Int("arg_count", len(args)))
	}

	// Add error information
	if err != nil {
		attrs = append(attrs, 
			slog.String("status", "error"),
			slog.String("error", err.Error()),
		)
		
		// Add MySQL-specific error code if available
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			attrs = append(attrs, slog.Int("error_code", int(mysqlErr.Number)))
		}
	} else {
		attrs = append(attrs, slog.String("status", "success"))
	}

	// Check for slow queries
	if p.slowQueryThreshold > 0 && duration > p.slowQueryThreshold {
		p.logger.LogAttrs(ctx, slog.LevelWarn, "slow query detected", attrs...)
	} else {
		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelError
		}
		p.logger.LogAttrs(ctx, level, "database query executed", attrs...)
	}

	// Record to slow query recorder if enabled
	if p.slowQueryRecorder != nil {
		p.slowQueryRecorder.Record(ctx, query, args, duration, err)
	}
}

// logConnection logs database connection events
func (p *Pool) logConnection(ctx context.Context, event string, duration time.Duration, err error) {
	if p == nil || !p.loggingEnabled || p.logger == nil { return }

	attrs := []slog.Attr{
		slog.String("event", event),
		slog.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
	}

	if err != nil {
		attrs = append(attrs, 
			slog.String("status", "error"),
			slog.String("error", err.Error()),
		)
		p.logger.LogAttrs(ctx, slog.LevelError, "database connection event", attrs...)
	} else {
		attrs = append(attrs, slog.String("status", "success"))
		p.logger.LogAttrs(ctx, slog.LevelDebug, "database connection event", attrs...)
	}
}

// logTransaction logs database transaction events
func (p *Pool) logTransaction(ctx context.Context, event string, duration time.Duration, err error) {
	if p == nil || !p.loggingEnabled || p.logger == nil { return }

	attrs := []slog.Attr{
		slog.String("event", event),
		slog.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
	}

	if err != nil {
		attrs = append(attrs, 
			slog.String("status", "error"),
			slog.String("error", err.Error()),
		)
		p.logger.LogAttrs(ctx, slog.LevelError, "database transaction event", attrs...)
	} else {
		attrs = append(attrs, slog.String("status", "success"))
		p.logger.LogAttrs(ctx, slog.LevelInfo, "database transaction event", attrs...)
	}
}

// logConnectionPool logs connection pool statistics
func (p *Pool) logConnectionPool(ctx context.Context, stats PoolStats) {
	if p == nil || !p.loggingEnabled || p.logger == nil { return }

	attrs := []slog.Attr{
		slog.Int("active_connections", stats.ActiveConnections),
		slog.Int("idle_connections", stats.IdleConnections),
		slog.Int("total_connections", stats.TotalConnections),
		slog.Int("max_open", stats.MaxOpen),
		slog.Int("max_idle", stats.MaxIdle),
	}

	p.logger.LogAttrs(ctx, slog.LevelDebug, "connection pool stats", attrs...)
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	ActiveConnections int
	IdleConnections   int
	TotalConnections  int
	MaxOpen          int
	MaxIdle          int
}

// GetPoolStats returns current pool statistics
func (p *Pool) GetPoolStats() PoolStats {
	if p == nil || p.db == nil {
		return PoolStats{}
	}

	stats := p.db.Stats()
	return PoolStats{
		ActiveConnections: stats.InUse,
		IdleConnections:   stats.Idle,
		TotalConnections:  stats.OpenConnections,
		MaxOpen:          stats.MaxOpenConnections,
		MaxIdle:          int(stats.MaxIdleClosed),
	}
}


