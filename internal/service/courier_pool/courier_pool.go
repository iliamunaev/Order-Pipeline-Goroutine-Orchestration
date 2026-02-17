package courier_pool

import "context"

// CourierPool limits concurrent courier assignments.

type CourierPool struct {
	sem chan struct{}
}

// New creates a pool with at least one slot
// and at most 128 slots.
func New(size int) *CourierPool {
	if size <= 0 {
		size = 1
	}
	if size > 128 {
		size = 128
	}
	return &CourierPool{sem: make(chan struct{}, size)}
}

// Acquire books a slot in the pool.
// It blocks until a slot is available or 
// the context is canceled, whichever happens first.
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
