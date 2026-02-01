// Package shared provides helper functions used by service steps,
// such as delay calculation and cancellation-aware sleeps.
package shared

import "time"

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
