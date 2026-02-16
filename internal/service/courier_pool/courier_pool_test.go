// courier_pool_test.go
package courier_pool

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkCourierPoolParallel benchmarks the performance
// of the courier pool in parallel.
func BenchmarkCourierPoolParallel(b *testing.B) {
	// test pool with different capacities
	for _, cap := range []int{1, 2, 8, 64, 128} {
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

	// fuzz the pool size
	f.Fuzz(func(t *testing.T, n int) {
		if n <= 0 {
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

// poolSizesTests is a table of test cases 
// for the NewCourierPoolSize test.
var poolSizesTests = []struct {
	in  int
	out int
}{
	// edge cases
	{in: -1, out: 1},
	{in: 0, out: 1},
	{in: 128, out: 128},
	{in: 129, out: 128},

	// negative cases
	{in: -3, out: 1},
	{in: -129, out: 1},

	// positive cases
	{in: 1, out: 1},
	{in: 3, out: 3},
	{in: 127, out: 127},
}

// TestNewCourierPoolSize tests the size of the courier pool.
func TestNewCourierPoolSize(t *testing.T) {
	for _, tt := range poolSizesTests {
		tt := tt
		t.Run(fmt.Sprintf("size=%d", tt.in), func(t *testing.T) {
			pool := New(tt.in)
			got := cap(pool.sem)
			if got != tt.out {
				t.Errorf("got %d, want %d", got, tt.out)
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
