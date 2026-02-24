// Package order orchestrates concurrent order processing.
package order

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
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

	out := make([]model.StepResult, len(s.steps))
	for i, step := range s.steps {
		out[i] = model.StepResult{Name: step.Name, Status: "canceled"}

		i, step := i, step
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

			out[i] = model.StepResult{
				Name:       step.Name,
				Status:     st,
				DurationMS: durMS,
				Detail:     detail,
			}
			return err
		})
	}

	err := g.Wait()
	return out, err
}
