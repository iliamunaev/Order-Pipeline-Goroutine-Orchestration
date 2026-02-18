// Package order orchestrates concurrent order processing.
package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
	"order-pipeline/internal/service/courier"
	"order-pipeline/internal/service/payment"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
	"order-pipeline/internal/service/vendor"
)

// Service orchestrates the order workflow.
type Service struct {
	pool *pool.Pool
	tr   *tracker.Tracker
}

// New creates a Service with the given pool and tracker.
func New(p *pool.Pool, tr *tracker.Tracker) *Service {
	if p == nil {
		panic("order.New: nil pool")
	}
	if tr == nil {
		tr = &tracker.Tracker{}
	}
	return &Service{
		pool: p,
		tr:   tr,
	}
}

// Process runs payment, vendor, and courier steps concurrently
// and returns per-step results in deterministic order.
func (s *Service) Process(ctx context.Context, req model.OrderRequest) ([]model.StepResult, error) {
	g, ctx := errgroup.WithContext(ctx)

	results := make(map[string]model.StepResult, 3)
	var mu sync.Mutex

	record := func(name string, fn func() error) func() error {
		return func() error {
			start := time.Now()
			err := fn()
			durMS := time.Since(start).Milliseconds()

			st := "ok"
			detail := ""
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					st = "canceled"
				} else {
					st = "error"
					detail = apperr.Kind(err)
				}
			}

			mu.Lock()
			results[name] = model.StepResult{
				Name:       name,
				Status:     st,
				DurationMS: durMS,
				Detail:     detail,
			}
			mu.Unlock()

			return err
		}
	}

	g.Go(record("payment", func() error { return payment.Process(ctx, req, s.tr) }))
	g.Go(record("vendor", func() error { return vendor.Notify(ctx, req, s.tr) }))
	g.Go(record("courier", func() error { return courier.Assign(ctx, req, s.pool, s.tr) }))

	err := g.Wait()
	return flattenResults(results), err
}

// flattenResults extracts steps in a fixed order so the response is deterministic
// regardless of which goroutine finished first.
func flattenResults(m map[string]model.StepResult) []model.StepResult {
	out := make([]model.StepResult, 0, 3)
	if r, ok := m["payment"]; ok {
		out = append(out, r)
	}
	if r, ok := m["vendor"]; ok {
		out = append(out, r)
	}
	if r, ok := m["courier"]; ok {
		out = append(out, r)
	}
	return out
}
