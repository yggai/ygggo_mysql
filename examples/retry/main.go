package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/yggai/ygggo_mysql"
)

// This example demonstrates the retryWithPolicy helper with a simulated operation
// that fails twice with retryable error and then succeeds.
func main() {
	ctx := context.Background()

	pol := ygggo_mysql.RetryPolicy{
		MaxAttempts: 3,
		BaseBackoff: 50 * time.Millisecond,
		MaxBackoff:  100 * time.Millisecond,
		Jitter:      false,
		MaxElapsed:  500 * time.Millisecond,
	}

	retryable := errors.New("temporary failure")
	classify := func(err error) ygggo_mysql.ErrorClass {
		if errors.Is(err, retryable) { return ygggo_mysql.ErrClassRetryable }
		return ygggo_mysql.ErrClassUnknown
	}

	calls := 0
	op := func() error {
		calls++
		if calls < 3 { return retryable }
		return nil
	}

	if err := ygggo_mysql.TestingRetry(ctx, pol, op, classify); err != nil {
		log.Fatalf("retry failed: %v", err)
	}
	fmt.Println("retry success after calls:", calls)
}

