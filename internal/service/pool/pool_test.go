package pool

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func BenchmarkPoolParallel(b *testing.B) {
	for _, cap := range []int{1, 2, 8, 64, 128} {
		b.Run(fmt.Sprintf("cap=%d", cap), func(b *testing.B) {
			p := New(cap)
			ctx := context.Background()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if err := p.Acquire(ctx); err != nil {
						b.Fatal(err)
					}
					_ = 42 + 1337
					p.Release()
				}
			})
		})
	}
}

var poolSizesTests = []struct {
	in  int
	out int
}{
	{in: -1, out: 1},
	{in: 0, out: 1},
	{in: -3, out: 1},
	{in: -129, out: 1},

	{in: 1, out: 1},
	{in: 3, out: 3},
	{in: 127, out: 127},
	{in: 128, out: 128},

	{in: 129, out: 128},
	{in: 1000, out: 128},
}

func TestNewPoolSize(t *testing.T) {
	t.Parallel()

	for _, tt := range poolSizesTests {
		tt := tt
		t.Run(fmt.Sprintf("size=%d", tt.in), func(t *testing.T) {
			t.Parallel()

			p := New(tt.in)
			if got := cap(p.sem); got != tt.out {
				t.Errorf("New(%d): got %d, want %d", tt.in, got, tt.out)
			}
		})
	}
}

var slotsTests = []struct {
	size int
}{
	{size: 0},
	{size: -1},
	{size: -3},
	{size: -129},
	{size: 1},
	{size: 2},
	{size: 8},
	{size: 64},
	{size: 128},
}

// A blocked Acquire must unblock exactly once after a single Release.
func TestPoolAcquireRelease(t *testing.T) {
	t.Parallel()

	for _, tt := range slotsTests {
		tt := tt
		t.Run(fmt.Sprintf("size=%d", tt.size), func(t *testing.T) {
			pool := New(tt.size)
			slots := cap(pool.sem)

			acquired := 0
			defer func() {
				for i := 0; i < acquired; i++ {
					pool.Release()
				}
			}()

			for i := 0; i < slots; i++ {
				if err := pool.Acquire(context.Background()); err != nil {
					t.Fatalf("prefill acquire #%d failed: %v", i+1, err)
				}
				acquired++
			}

			done := make(chan error, 1)
			started := make(chan struct{})

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			go func() {
				close(started)
				done <- pool.Acquire(ctx)
			}()

			<-started

			// Must not complete while pool is saturated.
			select {
			case err := <-done:
				t.Fatalf("expected extra acquire to block; got err=%v", err)
			default:
			}

			pool.Release()
			acquired--

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("unexpected extra acquire error: %v", err)
				}
				acquired++
			case <-time.After(200 * time.Millisecond):
				t.Fatal("expected blocked acquire to succeed after release")
			}
		})
	}
}

func TestPoolAcquireContextTimeout(t *testing.T) {
	t.Parallel()

	for _, tt := range slotsTests {
		tt := tt
		t.Run(fmt.Sprintf("size=%d", tt.size), func(t *testing.T) {
			pool := New(tt.size)
			slots := cap(pool.sem)

			acquired := 0
			defer func() {
				for i := 0; i < acquired; i++ {
					pool.Release()
				}
			}()

			for i := 0; i < slots; i++ {
				if err := pool.Acquire(context.Background()); err != nil {
					t.Fatalf("prefill acquire #%d failed: %v", i+1, err)
				}
				acquired++
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := pool.Acquire(ctx)
			if err == nil {
				t.Fatal("expected acquire to fail on context timeout")
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
			}
		})
	}
}
