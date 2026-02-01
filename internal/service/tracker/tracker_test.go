package tracker

import (
	"sync"
	"testing"
)

func TestTrackerIncDec(t *testing.T) {
	t.Parallel()

	tr := &Tracker{}
	tr.Inc()
	if got := tr.Running(); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	tr.Dec()
	if got := tr.Running(); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestTrackerConcurrent(t *testing.T) {
	t.Parallel()

	tr := &Tracker{}
	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				tr.Inc()
				tr.Dec()
			}
		}()
	}
	wg.Wait()

	if got := tr.Running(); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}
