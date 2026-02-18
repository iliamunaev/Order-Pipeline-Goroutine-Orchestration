// Package shared provides delay and sleep helpers used across service steps.
package shared

import (
	"context"
	"time"
)

// DelayForStep returns an override delay when provided, otherwise defaultDelay.
func DelayForStep(delayMS map[string]int64, step string, defaultDelay time.Duration) time.Duration {
	if delayMS == nil {
		return defaultDelay
	}
	if ms, ok := delayMS[step]; ok && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultDelay
}

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
