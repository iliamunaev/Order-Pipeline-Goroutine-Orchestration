// Package tracker provides lightweight atomic counters
// to track the number of in-flight steps (goroutines).
//
// It proves that all steps have completed when the counter is zero,
// no goroutines are left running.
package tracker

import "sync/atomic"

// Tracker counts running steps using atomics.
//
// It is used to track the number of in-flight steps (goroutines).
type Tracker struct {
	running atomic.Int64
}

// Inc increments the running step count.
//
// Internally, it uses atomic.AddInt64 to increment the counter.
// Each goroutine should call Inc before starting its work,
// and Dec after completing its work.
func (t *Tracker) Inc() { t.running.Add(1) }

// Dec decrements the running step count.
//
// It uses atomic.AddInt64 to decrement the counter.
// Each goroutine should call Dec after completing its work
// to ensure the counter is decremented correctly.
func (t *Tracker) Dec() { t.running.Add(-1) }

// Running returns the current number of in-flight steps.
//
// It uses atomic.LoadInt64 to return the current value of the counter.
func (t *Tracker) Running() int64 { return t.running.Load() }
