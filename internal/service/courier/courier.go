// Package courier contains the courier assignment step for an order workflow.
// It enforces concurrency limits and simulates assignment delays and failures.
package courier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"order-pipeline/internal/model"
	shared "order-pipeline/internal/service/shared"
	"order-pipeline/internal/service/tracker"
)

var ErrNoCourierAvailable = errors.New("no courier available")

type Limiter interface {
	Acquire(context.Context) error
	Release()
}

// Assign runs the courier assignment step for an order.
func Assign(ctx context.Context, req model.OrderRequest, l Limiter, tr *tracker.Tracker) error {
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
