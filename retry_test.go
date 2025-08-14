package ygggo_mysql

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errRetry = errors.New("retryable")
var errNonRetry = errors.New("non-retryable")

func classifyForTest(err error) ErrorClass {
	if errors.Is(err, errRetry) { return ErrClassRetryable }
	return ErrClassUnknown
}

func TestRetry_SucceedsAfterRetries(t *testing.T) {
	ctx := context.Background()
	pol := RetryPolicy{MaxAttempts: 3, BaseBackoff: time.Millisecond, MaxBackoff: 2*time.Millisecond, Jitter: false, MaxElapsed: 50*time.Millisecond}
	calls := 0
	op := func() error {
		calls++
		if calls < 3 { return errRetry }
		return nil
	}
	if err := retryWithPolicy(ctx, pol, op, classifyForTest); err != nil {
		t.Fatalf("retryWithPolicy err: %v", err)
	}
	if calls != 3 { t.Fatalf("calls=%d want 3", calls) }
}

func TestRetry_StopsOnNonRetryable(t *testing.T) {
	ctx := context.Background()
	pol := RetryPolicy{MaxAttempts: 5, BaseBackoff: time.Millisecond, MaxBackoff: 2*time.Millisecond, Jitter: false, MaxElapsed: 50*time.Millisecond}
	calls := 0
	op := func() error { calls++; return errNonRetry }
	if err := retryWithPolicy(ctx, pol, op, classifyForTest); !errors.Is(err, errNonRetry) {
		t.Fatalf("expected non-retryable returned, got %v", err)
	}
	if calls != 1 { t.Fatalf("calls=%d want 1", calls) }
}

func TestRetry_RespectsMaxElapsed(t *testing.T) {
	ctx := context.Background()
	pol := RetryPolicy{MaxAttempts: 100, BaseBackoff: 2*time.Millisecond, MaxBackoff: 5*time.Millisecond, Jitter: false, MaxElapsed: 5*time.Millisecond}
	calls := 0
	op := func() error { calls++; return errRetry }
	start := time.Now()
	err := retryWithPolicy(ctx, pol, op, classifyForTest)
	if err == nil { t.Fatalf("expected error due to elapsed") }
	if calls < 1 || calls >= pol.MaxAttempts { t.Fatalf("calls=%d out of expected range", calls) }
	_ = start
}

