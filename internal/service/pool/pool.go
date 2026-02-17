// pool.go
package pool

import "context"

// Pool limits concurrent resource assignments.
// Mimics real-world resource pools like availability, capacity, etc.
type Pool struct {
	sem chan struct{}
}

// New creates a pool with at least one slot
// and at most 128 slots.
func New(size int) *Pool {
	if size <= 0 {
		size = 1
	}
	if size > 128 {
		size = 128
	}
	return &Pool{sem: make(chan struct{}, size)}
}

// Acquire books a slot in the pool.
// It blocks until a slot is available or
// the context is canceled, whichever happens first.
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
