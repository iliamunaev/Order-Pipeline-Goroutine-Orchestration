// Package tracker provides lightweight counters for running work.
package tracker

import "sync/atomic"

// Tracker counts running steps using atomics.
type Tracker struct {
	running atomic.Int64
}

// Inc increments the running counter.
func (t *Tracker) Inc() { t.running.Add(1) }
// Dec decrements the running counter.
func (t *Tracker) Dec() { t.running.Add(-1) }
// Running returns the current running count.
func (t *Tracker) Running() int64 { return t.running.Load() }
