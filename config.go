package ygggo_mysql

import "time"

// PoolConfig holds pool-related settings (placeholders).
type PoolConfig struct {
	MaxOpen          int
	MaxIdle          int
	ConnMaxLifetime  time.Duration
	ConnMaxIdleTime  time.Duration
}

// Config holds library configuration (placeholders).
type Config struct {
	DSN                string
	Pool               PoolConfig
	Retry              RetryPolicy
	Telemetry          TelemetryConfig
	SlowQueryThreshold time.Duration
}
