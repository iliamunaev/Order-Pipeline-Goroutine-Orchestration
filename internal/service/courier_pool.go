package service

import "context"

// https://dave.cheney.net/2014/03/25/the-empty-struct
type CourierPool struct {
	sem chan struct{}
}

func NewCourierPool(size int) *CourierPool { // add test
	if size <= 0 {
		size = 1
	}
	return &CourierPool{sem: make(chan struct{}, size)}
}

func (p *CourierPool) Acquire(ctx context.Context) error { // add test
	select {
	case p.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *CourierPool) Release() {
	<-p.sem
}
