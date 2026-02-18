// Package courier implements the courier assignment step.
package courier

import (
	"context"
	"fmt"
	"time"

	"order-pipeline/internal/model"
	"order-pipeline/internal/service/shared"
	"order-pipeline/internal/service/tracker"
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
