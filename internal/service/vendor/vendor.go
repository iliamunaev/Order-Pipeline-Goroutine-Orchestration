// Package vendor provides the vendor-notification step used by the order pipeline.
//
// Notify respects context cancellation and may return a classified domain
// error when the vendor is unavailable.
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

// Notify executes the vendor-notification step.
//
// It simulates latency using a per-step delay override and respects
// context cancellation. If the vendor is unavailable, it returns
// an error wrapping ErrUnavailable.
func Notify(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker) error {
	// Track the running step
	if tr != nil {
		tr.Inc()
		defer tr.Dec()
	}

	const stepName = "vendor"
	delay := resolveStepDelay(req.DelayMS, stepName, 200*time.Millisecond)

	// Block step until the delay elapses or the context is done
	if err := waitOrCancel(ctx, delay); err != nil {
		return err
	}

	// If the step is configured to fail, return an error
	if req.FailStep == stepName {
		return fmt.Errorf("vendor notify: %w", ErrUnavailable)
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
