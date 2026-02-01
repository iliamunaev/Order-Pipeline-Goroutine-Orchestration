// Package shared provides helper functions used by service steps,
// such as delay calculation and cancellation-aware sleeps.
package shared

import (
	"context"
	"time"
)

// SleepOrDone waits for the duration or returns early on context cancellation.
func SleepOrDone(ctx context.Context, d time.Duration) error {
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
