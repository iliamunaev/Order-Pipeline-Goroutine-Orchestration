// Package courier provides the courier-assignment step used by the order pipeline.
//
// Assign respects context cancellation and may enforce a concurrency limit via a
// provided limiter. Domain failures are returned as errors that may implement
// Kind() for classification.
package courier

import (
	"context"
	"fmt"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

type noCourierError struct{}

func (noCourierError) Error() string { return "no courier available" }
func (noCourierError) Kind() string  { return "no_courier" }

// ErrNoCourierAvailable is returned when no courier can be assigned.
var ErrNoCourierAvailable = noCourierError{}

// limiter abstracts a bounded concurrency gate.
type limiter interface {
	Acquire(context.Context) error
	Release()
}

// Assign assigns a courier for the given order.
//
// Assign blocks on the provided limiter before doing work. It returns ctx.Err()
// if acquisition or execution is aborted due to cancellation or deadline.
// On domain failure it returns an error wrapping ErrNoCourierAvailable.
func Assign(ctx context.Context, req model.OrderRequest, l limiter, tr *tracker.Tracker) error {
	// Track the running step
	if tr != nil {
		tr.Inc()
		defer tr.Dec()
	}

	const stepName = "courier"

	// Assign provided delay time or use default value
	delay := resolveStepDelay(req.DelayMS, stepName, 100*time.Millisecond)

	if err := l.Acquire(ctx); err != nil {
		return err
	}
	defer l.Release()

	// Block step until the delay elapses or the context is done
	if err := waitOrCancel(ctx, delay); err != nil {
		return err
	}

	// If the step is configured to fail, return an error
	if req.FailStep == stepName {
		return fmt.Errorf("courier assign: %w", ErrNoCourierAvailable)
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
