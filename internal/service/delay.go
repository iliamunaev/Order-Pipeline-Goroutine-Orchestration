package service

import "time"

func delayForStep(delayMS map[string]int64, step string, defaultDelay time.Duration) time.Duration {
	if delayMS == nil {
		return defaultDelay
	}
	if ms, ok := delayMS[step]; ok && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultDelay
}
