// Package pool provides concurrency limiters for service steps.
package pool

import "context"

// CourierPool limits concurrent courier assignments.
type CourierPool struct {
	sem chan struct{}
}

// NewCourierPool creates a pool with at least one slot.
func NewCourierPool(size int) *CourierPool {
	if size <= 0 {
		size = 1
	}
	return &CourierPool{sem: make(chan struct{}, size)}
}

// Acquire blocks until a slot is available or the context is canceled.
func (p *CourierPool) Acquire(ctx context.Context) error {
	select {
	case p.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release frees a previously acquired slot.
func (p *CourierPool) Release() {
	<-p.sem
}
