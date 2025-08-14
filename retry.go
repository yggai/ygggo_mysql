package ygggo_mysql

import (
	"context"
	"math/rand"
	"time"
)

// RetryPolicy controls retry strategy.
type RetryPolicy struct {
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	Jitter      bool
	MaxElapsed  time.Duration
}

// retryWithPolicy retries op according to policy. classify returns error class.
func retryWithPolicy(ctx context.Context, pol RetryPolicy, op func() error, classify func(error) ErrorClass) error {
	if pol.MaxAttempts <= 0 { pol.MaxAttempts = 1 }
	if pol.BaseBackoff <= 0 { pol.BaseBackoff = 10 * time.Millisecond }
	if pol.MaxBackoff <= 0 { pol.MaxBackoff = pol.BaseBackoff }
	start := time.Now()
	var lastErr error
	for attempt := 1; attempt <= pol.MaxAttempts; attempt++ {
		if ctx.Err() != nil { return ctx.Err() }
		err := op()
		if err == nil { return nil }
		cl := classify(err)
		if cl != ErrClassRetryable && cl != ErrClassReadonly {
			return err
		}
		lastErr = err
		if attempt == pol.MaxAttempts { break }
		if pol.MaxElapsed > 0 && time.Since(start) >= pol.MaxElapsed { break }
		// sleep
		d := pol.BaseBackoff * time.Duration(attempt)
		if d > pol.MaxBackoff { d = pol.MaxBackoff }
		if pol.Jitter {
			j := time.Duration(rand.Int63n(int64(d)))
			d = j
		}
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop(); return ctx.Err()
		case <-t.C:
		}
	}
	return lastErr
}


// TestingRetry exposes retryWithPolicy for examples; not part of the stable API yet.
func TestingRetry(ctx context.Context, pol RetryPolicy, op func() error, classify func(error) ErrorClass) error {
	return retryWithPolicy(ctx, pol, op, classify)
}
