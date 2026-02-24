// Package payment provides the payment-processing step used by the order pipeline.
//
// Process respects context cancellation and may return a classified domain
// error when payment is declined or invalid.
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

// Process executes the payment step.
//
// It simulates latency using a per-step delay override and respects
// context cancellation. If payment fails validation or is declined,
// it returns an error wrapping ErrDeclined.
func Process(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker) error {
	// Track the running step
	if tr != nil {
		tr.Inc()
		defer tr.Dec()
	}

	const stepName = "payment"
	delay := resolveStepDelay(req.DelayMS, stepName, 150*time.Millisecond)

	// Block step until the delay elapses or the context is done
	if err := waitOrCancel(ctx, delay); err != nil {
		return err
	}

	// If the step is configured to fail, return an error
	if req.FailStep == stepName {
		return fmt.Errorf("payment: %w", ErrDeclined)
	}

	if req.Amount <= 0 {
		return fmt.Errorf("payment: %w", ErrDeclined)
	}

	return nil
}

// resolveStepDelay returns the effective delay for a step.
//
// If delayMS contains a positive value for the given step (in milliseconds),
// that value is used. Otherwise, defaultDelay is returned.
func resolveStepDelay(delayMS map[string]int64, step string, defaultDelay time.Duration) time.Duration {
	if delayMS == nil {
		return defaultDelay
	}
	if ms, ok := delayMS[step]; ok && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultDelay
}

// waitOrCancel blocks for d or until ctx is canceled.
//
// It returns nil if the duration elapses, or ctx.Err() if the context
// is done first. If d <= 0, it returns immediately.
func waitOrCancel(ctx context.Context, d time.Duration) error {
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
