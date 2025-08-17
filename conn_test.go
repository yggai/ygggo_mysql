package ygggo_mysql

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithConn_AutoReturnAndFnError(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// Create a pool with MaxOpen=1 to test timeout behavior
	config := helper.Config()
	config.Pool.MaxOpen = 1
	config.Pool.MaxIdle = 1

	ctx := context.Background()
	p, err := NewPool(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// First goroutine holds the only connection until released
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		err = p.WithConn(context.Background(), func(c DatabaseConn) error {
			<-release // hold connection
			return nil
		})
		close(done)
	}()

	// Give time to acquire
	time.Sleep(20 * time.Millisecond)

	// Second acquire should timeout due to pool exhaustion
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := p.Acquire(ctx); err == nil {
		t.Fatalf("expected timeout acquiring conn, got nil")
	}

	// Release first and ensure next acquire succeeds
	close(release)
	<-done
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	conn, err := p.Acquire(ctx2)
	if err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}

	// WithConn should propagate fn error
	sent := errors.New("sentinel")
	err = p.WithConn(context.Background(), func(c DatabaseConn) error { return sent })
	if !errors.Is(err, sent) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}
