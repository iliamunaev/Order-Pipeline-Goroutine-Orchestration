// courier_pool_test.go
package courier_pool

import (
	"fmt"
	"context"
	"testing"
	"time"
)

// BenchmarkCourierPoolParallel benchmarks the performance 
// of the courier pool in parallel.
func BenchmarkCourierPoolParallel(b *testing.B) {
	// test pool with different capacities
    for _, cap := range []int{1, 2, 8, 64} {
        b.Run(fmt.Sprintf("cap=%d", cap), func(b *testing.B) {
            p := New(cap)
            ctx := context.Background()
            b.ReportAllocs()
			// run parallel tests
            b.RunParallel(func(pb *testing.PB) {
                for pb.Next() {
					// acquire a courier
                    if err := p.Acquire(ctx); err != nil {
                        b.Fatal(err)
                    }
					// do some work for the courier
					_ = 42 + 1337
					// release the courier
                    p.Release()
                }
            })
        })
    }
}

// FuzzCourierPoolAcquireRelease is a minimal fuzz test for pool acquire/release.
func FuzzCourierPoolAcquireRelease(f *testing.F) {
	f.Add(1)

	f.Fuzz(func(t *testing.T, n int) {
		if n < 1 {
			n = 1
		}
		if n > 128 {
			n = 128
		}

		p := New(n)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		if err := p.Acquire(ctx); err != nil {
			t.Fatalf("acquire: %v (n=%d)", err, n)
		}
		p.Release()
	})
}


// TestNewCourierPoolSize tests the size of the courier pool.
func TestNewCourierPoolSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		size    int
		want int
	}{
		{name: "defaults to size 1 for zero", size: 0, want: 1},
		{name: "defaults to size 1 for negative", size: -3, want: 1},
		{name: "uses provided size", size: 3, want: 3},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := New(tt.size)
			if got := cap(pool.sem); got != tt.want {
				t.Fatalf("expected cap %d, got %d", tt.want, got)
			}
		})
	}
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
