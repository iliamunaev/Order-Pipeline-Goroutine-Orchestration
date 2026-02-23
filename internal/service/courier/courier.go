// Package courier implements the courier assignment step.
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

	delay := delayForStep(req.DelayMS, "courier", 100*time.Millisecond)

	if err := l.Acquire(ctx); err != nil {
		return err
	}
	defer l.Release()

	if err := sleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "courier" {
		return fmt.Errorf("courier assign: %w", ErrNoCourierAvailable)
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
