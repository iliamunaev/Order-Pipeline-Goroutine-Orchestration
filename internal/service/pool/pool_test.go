// pool_test.go
package pool

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// BenchmarkCourierParallel benchmarks the performance
// of the pool in parallel.
func BenchmarkPoolParallel(b *testing.B) {
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

// FuzzPoolAcquireRelease is a minimal fuzz test for pool acquire/release.
func FuzzPoolAcquireRelease(f *testing.F) {
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
// for the NewPoolSize test.
var poolSizesTests = []struct {
	in  int
	out int
}{
	// below minimum
	{in: -1, out: 1},
	{in: 0, out: 1},
	{in: -3, out: 1},
	{in: -129, out: 1},

	// within range
	{in: 1, out: 1},
	{in: 3, out: 3},
	{in: 127, out: 127},
	{in: 128, out: 128},

	// above maximum
	{in: 129, out: 128},
	{in: 1000, out: 128},
}

// TestNewPoolSize tests the size of the pool.
func TestNewPoolSize(t *testing.T) {
	for _, tt := range poolSizesTests {
		tt := tt
		t.Run(fmt.Sprintf("size=%d", tt.in), func(t *testing.T) {
			pool := New(tt.in)
			if pool == nil {
				t.Fatalf("New(%d) returned pool == nil", tt.in)
			}
			if got := cap(pool.sem); got != tt.out {
				t.Errorf("New(%d): got %d, want %d", tt.in, got, tt.out)
			}
		})
	}
}

// slotsTests is a table of test cases
// for the TestPoolAcquireRelease test and
// TestPoolAcquireContextTimeout test.
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

// TestPoolAcquireRelease tests the acquire/release of slots in the pool.
// It tests the behavior of the pool when the number of slots is
// 0, -1, -3, -129, 1, 2, 8, 64, and 128.
func TestPoolAcquireRelease(t *testing.T) {
	for _, tt := range slotsTests {
		tt := tt
		t.Run(fmt.Sprintf("size=%d", tt.size), func(t *testing.T) {
			pool := New(tt.size)
			if pool == nil {
				t.Fatalf("New(%d) returned pool == nil", tt.size)
			}

			slots := cap(pool.sem)

			// Track the number of slots acquired.
			acquired := 0
			defer func() {
				for i := 0; i < acquired; i++ {
					pool.Release()
				}
			}()

			// Prefill: take all slots.
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

			// Must not complete before a release.
			select {
			case err := <-done:
				t.Fatalf("expected extra acquire to block; got err=%v", err)
			default:
			}

			// Release one slot.
			pool.Release()
			acquired--

			// After one release, blocked acquire should succeed.
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("unexpected extra acquire error: %v", err)
				}
				acquired++ // extra acquire succeeded
			case <-time.After(200 * time.Millisecond):
				t.Fatal("expected blocked acquire to succeed after release")
			}
		})
	}
}

// TestPoolAcquireContextTimeout tests the acquire of slots in the pool with a context timeout.
// It tests the behavior of the pool when the number of slots is
// 0, -1, -3, -129, 1, 2, 8, 64, and 128.
func TestPoolAcquireContextTimeout(t *testing.T) {
	for _, tt := range slotsTests {
		tt := tt
		t.Run(fmt.Sprintf("size=%d", tt.size), func(t *testing.T) {
			pool := New(tt.size)
			if pool == nil {
				t.Fatalf("New(%d) returned pool == nil", tt.size)
			}

			slots := cap(pool.sem)

			// Track the number of slots acquired.
			acquired := 0
			defer func() {
				for i := 0; i < acquired; i++ {
					pool.Release()
				}
			}()

			// Prefill: take all slots.
			for i := 0; i < slots; i++ {
				if err := pool.Acquire(context.Background()); err != nil {
					t.Fatalf("prefill acquire #%d failed: %v", i+1, err)
				}
				acquired++
			}

			// Pool is full; next acquire must time out.
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
