// Package order contains application orchestration for order processing.
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
	shared "order-pipeline/internal/service/shared"
	"order-pipeline/internal/service/tracker"
	"order-pipeline/internal/service/vendor"
)

// Service orchestrates the order workflow.
type Service struct {
	pool *pool.Pool
	tr   *tracker.Tracker
	vs   vendor.Helper
}

type defaultVendorService struct{}

func (defaultVendorService) DelayForStep(delayMS map[string]int64, step string, defaultMS time.Duration) time.Duration {
	return shared.DelayForStep(delayMS, step, defaultMS)
}

func (defaultVendorService) SleepOrDone(ctx context.Context, delay time.Duration) error {
	return shared.SleepOrDone(ctx, delay)
}

// New creates an orchestration service with dependencies.
func New(pool *pool.Pool, tr *tracker.Tracker) *Service {
	if tr == nil {
		tr = &tracker.Tracker{}
	}
	return &Service{
		pool: pool,
		tr:   tr,
		vs:   defaultVendorService{},
	}
}

// Process runs payment, vendor, and courier steps concurrently and returns step results.
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
	g.Go(record("vendor", func() error { return vendor.Notify(ctx, req, s.tr, s.vs) }))
	g.Go(record("courier", func() error { return courier.Assign(ctx, req, s.pool, s.tr) }))

	err := g.Wait()
	return flattenResults(results), err
}

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
