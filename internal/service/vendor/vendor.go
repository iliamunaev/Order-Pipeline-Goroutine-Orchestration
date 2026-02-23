// Package vendor implements the vendor notification step.
package vendor

import (
	"context"
	"fmt"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

type unavailableError struct{}

func (unavailableError) Error() string { return "vendor unavailable" }
func (unavailableError) Kind() string  { return "vendor_unavailable" }

// ErrUnavailable is returned when the vendor cannot be reached.
var ErrUnavailable = unavailableError{}

// Notify runs the vendor notification step for an order.
func Notify(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker) error {
	// Track the running step
	tr.Inc()
	defer tr.Dec()

	delay := delayForStep(req.DelayMS, "vendor", 200*time.Millisecond)

	if err := sleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "vendor" {
		return fmt.Errorf("vendor notify: %w", ErrUnavailable)
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
