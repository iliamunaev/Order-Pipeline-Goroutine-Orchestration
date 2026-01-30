package service

import "context"

type CourierPool struct {
	sem chan struct{}
}

func NewCourierPool(size int) *CourierPool {
	if size <= 0 {
		size = 1
	}
	return &CourierPool{sem: make(chan struct{}, size)}
}

func (p *CourierPool) Acquire(ctx context.Context) error {
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
