// Package courier implements the courier assignment step.
package courier

import (
	"context"
	"fmt"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/shared"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

type noCourierError struct{}

func (noCourierError) Error() string { return "no courier available" }
func (noCourierError) Kind() string  { return "no_courier" }

// ErrNoCourierAvailable is returned when no courier can be assigned.
var ErrNoCourierAvailable = noCourierError{}

// limiter abstracts bounded-concurrency resource acquisition.
type limiter interface {
	Acquire(context.Context) error
	Release()
}

// Assign runs the courier assignment step for an order.
func Assign(ctx context.Context, req model.OrderRequest, l limiter, tr *tracker.Tracker) error {
	// Track the running step
	tr.Inc()
	defer tr.Dec()

	delay := shared.DelayForStep(req.DelayMS, "courier", 100*time.Millisecond)

	if err := l.Acquire(ctx); err != nil {
		return err
	}
	defer l.Release()

	if err := shared.SleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "courier" {
		return fmt.Errorf("courier assign: %w", ErrNoCourierAvailable)
	}

	return nil
}
