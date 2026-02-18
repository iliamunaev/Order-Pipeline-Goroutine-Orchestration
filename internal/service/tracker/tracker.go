// Package tracker provides lightweight atomic counters for in-flight work.
package tracker

import "sync/atomic"

// Tracker counts running steps using atomics.
type Tracker struct {
	running atomic.Int64
}

// Inc increments the running step count.
func (t *Tracker) Inc() { t.running.Add(1) }

// Dec decrements the running step count.
func (t *Tracker) Dec() { t.running.Add(-1) }

// Running returns the current number of in-flight steps.
func (t *Tracker) Running() int64 { return t.running.Load() }
