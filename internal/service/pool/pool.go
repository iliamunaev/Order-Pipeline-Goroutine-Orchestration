// Package pool provides a bounded concurrency semaphore.
package pool

import "context"

// Pool limits concurrent resource assignments.
type Pool struct {
	sem chan struct{}
}

// New creates a pool with at least one slot
// and at most 128 slots.
// Slots are used to limit the number of concurrent requests to the order processor.
func New(size int) *Pool {
	if size <= 0 {
		size = 1
	}
	if size > 128 {
		size = 128
	}
	return &Pool{sem: make(chan struct{}, size)}
}

// Acquire reserves one slot in the pool.
// If the pool is full, it blocks until a slot becomes available
// or the context is canceled.
// It returns ctx.Err() if acquisition is aborted due to cancellation.
func (p *Pool) Acquire(ctx context.Context) error {
	select {
	case p.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release frees a previously acquired slot.
func (p *Pool) Release() {
	<-p.sem
}
