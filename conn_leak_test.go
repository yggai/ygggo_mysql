package ygggo_mysql

import (
	"context"
	"testing"
	"time"
)

func TestLeakDetection_ReportsWhenHeldBeyondThreshold(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	p := helper.Pool()

	// Configure leak threshold and handler
	p.SetBorrowWarnThreshold(20 * time.Millisecond)
	ch := make(chan BorrowLeak, 1)
	p.SetLeakHandler(func(info BorrowLeak){ ch <- info })

	// Acquire but don't close within threshold
	ctx := context.Background()
	c, err := p.Acquire(ctx)
	if err != nil { t.Fatalf("acquire: %v", err) }
	defer c.Close()

	// Wait beyond threshold to allow handler to fire
	time.Sleep(40 * time.Millisecond)
	select {
	case info := <-ch:
		if info.HeldFor <= 0 { t.Fatalf("expected positive HeldFor, got %v", info.HeldFor) }
	default:
		t.Fatalf("expected leak notification, got none")
	}
}

func TestLeakDetection_NoReportWhenWithinThreshold(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	p := helper.Pool()

	p.SetBorrowWarnThreshold(50 * time.Millisecond)
	ch := make(chan BorrowLeak, 1)
	p.SetLeakHandler(func(info BorrowLeak){ ch <- info })

	err = p.WithConn(context.Background(), func(c DatabaseConn) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil { t.Fatalf("WithConn err: %v", err) }

	// Give some time to ensure no late notification
	time.Sleep(60 * time.Millisecond)
	select {
	case info := <-ch:
		t.Fatalf("unexpected leak notification: %+v", info)
	default:
		// ok
	}
}

