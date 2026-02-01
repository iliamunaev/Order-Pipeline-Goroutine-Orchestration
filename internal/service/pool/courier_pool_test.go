package pool

import (
	"context"
	"testing"
	"time"
)

func TestNewCourierPoolSize(t *testing.T) {
	t.Parallel()

	t.Run("defaults to size 1", func(t *testing.T) {
		t.Parallel()

		pool := New(0)
		if got := cap(pool.sem); got != 1 {
			t.Fatalf("expected cap 1, got %d", got)
		}
	})

	t.Run("uses provided size", func(t *testing.T) {
		t.Parallel()

		pool := New(3)
		if got := cap(pool.sem); got != 3 {
			t.Fatalf("expected cap 3, got %d", got)
		}
	})
}

func TestCourierPoolAcquireRelease(t *testing.T) {
	pool := New(1)

	if err := pool.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- pool.Acquire(context.Background())
	}()

	select {
	case <-done:
		t.Fatal("expected second acquire to block before release")
	case <-time.After(25 * time.Millisecond):
	}

	pool.Release()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected second acquire error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected second acquire to succeed after release")
	}

	pool.Release()
}

func TestCourierPoolAcquireContextCancel(t *testing.T) {
	pool := New(1)

	if err := pool.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := pool.Acquire(ctx)
	if err == nil {
		t.Fatal("expected acquire to fail on context timeout")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}

	pool.Release()
}
