package ygggo_mysql

import "time"

// RetryPolicy is a placeholder for retry config.
type RetryPolicy struct {
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	Jitter      bool
}

