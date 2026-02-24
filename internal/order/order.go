// Package order provides a concurrent orchestrator for order processing.
//
// A Service coordinates a set of registered steps and executes them
// concurrently using context-aware cancellation semantics.
//
// Steps are executed in parallel. If any step returns a non-nil error,
// the shared context is canceled and remaining steps are expected to
// stop promptly. The first non-nil error is returned to the caller.
//
// The result slice always preserves step registration order.
package order

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
)

// Steps contract for the order pipeline.
type Step struct {
	Name string
	Run  func(ctx context.Context, req model.OrderRequest) error
}

// Service orchestrates the order workflow.
type Service struct {
	steps []Step
}

// New returns a Service that executes the provided steps concurrently.
//
// It panics if no steps are provided.
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

// Process executes all configured steps concurrently.
//
// Each step receives the same context. If any step returns a non-nil error,
// the shared context is canceled and remaining steps are expected to abort
// promptly. The first non-nil error is returned.
//
// The returned slice contains one StepResult per registered step,
// in registration order.
func (s *Service) Process(ctx context.Context, req model.OrderRequest) ([]model.StepResult, error) {
	g, ctx := errgroup.WithContext(ctx)

	out := make([]model.StepResult, len(s.steps))
	for i, step := range s.steps {
		out[i] = model.StepResult{Name: step.Name, Status: "canceled", Detail: "operation not completed"} // pre-fill with default value

		// Call the steps concurrently
		i, step := i, step
		g.Go(func() error {
			start := time.Now()
			err := step.Run(ctx, req) // execute the step function
			durationMS := time.Since(start).Milliseconds()

			status := "ok" // default value
			detail := ""
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					status = "canceled"
				} else {
					status = "error"
					var k kinder
					if errors.As(err, &k) {
						detail = k.Kind()
					}
				}
			}

			out[i] = model.StepResult{
				Name:       step.Name,
				Status:     status,
				DurationMS: durationMS,
				Detail:     detail,
			}
			return err
		})
	}

	err := g.Wait()
	return out, err
}
