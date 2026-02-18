// Package order orchestrates concurrent order processing.
package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"order-pipeline/internal/model"
)

// Step is a named unit of work in the order pipeline.
type Step struct {
	Name string
	Run  func(ctx context.Context, req model.OrderRequest) error
}

// Service orchestrates the order workflow.
type Service struct {
	steps []Step
}

// New creates a Service that runs the given steps concurrently.
func New(steps []Step) *Service {
	if len(steps) == 0 {
		panic("order.New: no steps")
	}
	return &Service{steps: steps}
}

// kinder is satisfied by errors that carry a classification kind.
type kinder interface {
	Kind() string
}

// Process runs all steps concurrently and returns per-step results
// in registration order.
func (s *Service) Process(ctx context.Context, req model.OrderRequest) ([]model.StepResult, error) {
	g, ctx := errgroup.WithContext(ctx)

	results := make(map[string]model.StepResult, len(s.steps))
	var mu sync.Mutex

	for _, step := range s.steps {
		step := step
		g.Go(func() error {
			start := time.Now()
			err := step.Run(ctx, req)
			durMS := time.Since(start).Milliseconds()

			st := "ok"
			detail := ""
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					st = "canceled"
				} else {
					st = "error"
					var k kinder
					if errors.As(err, &k) {
						detail = k.Kind()
					}
				}
			}

			mu.Lock()
			results[step.Name] = model.StepResult{
				Name:       step.Name,
				Status:     st,
				DurationMS: durMS,
				Detail:     detail,
			}
			mu.Unlock()

			return err
		})
	}

	err := g.Wait()

	out := make([]model.StepResult, 0, len(s.steps))
	for _, step := range s.steps {
		if r, ok := results[step.Name]; ok {
			out = append(out, r)
		}
	}

	return out, err
}
