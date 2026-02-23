// Package payment implements the payment processing step.
package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

type declinedError struct{}

func (declinedError) Error() string { return "payment declined" }
func (declinedError) Kind() string  { return "payment_declined" }

// ErrDeclined is returned when a payment is declined or the amount is invalid.
var ErrDeclined = declinedError{}

// Process runs the payment step for an order.
func Process(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker) error {
	// Track the running step
	tr.Inc()
	defer tr.Dec()

	delay := delayForStep(req.DelayMS, "payment", 150*time.Millisecond)

	if err := sleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "payment" {
		return fmt.Errorf("payment: %w", ErrDeclined)
	}

	if req.Amount <= 0 {
		return fmt.Errorf("payment: %w", ErrDeclined)
	}

	return nil
}

func delayForStep(delayMS map[string]int64, step string, defaultDelay time.Duration) time.Duration {
	if delayMS == nil {
		return defaultDelay
	}
	if ms, ok := delayMS[step]; ok && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultDelay
}

func sleepOrDone(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
