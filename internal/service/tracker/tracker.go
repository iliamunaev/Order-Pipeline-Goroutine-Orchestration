// Package tracker provides lightweight atomic counters
// to track the number of in-flight steps (goroutines).
//
// It proves that all steps have completed when the counter is zero,
// no goroutines are left running.
package tracker

import "sync/atomic"

// Tracker counts running steps using lock-free atomic operations.
//
// It uses atomic.Int64 to store the counter value safely
// under concurrent access.
type Tracker struct {
	running atomic.Int64
}

// Inc increments the running step count.
// Each goroutine should call Inc before starting its work.
func (t *Tracker) Inc() { t.running.Add(1) }

// Dec decrements the running step count.
// Each goroutine should call Dec after completing its work.
func (t *Tracker) Dec() { t.running.Add(-1) }

// Running returns the current number of in-flight steps.
func (t *Tracker) Running() int64 { return t.running.Load() }
