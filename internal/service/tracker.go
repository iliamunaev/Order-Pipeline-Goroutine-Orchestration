package service

import "sync/atomic"

type Tracker struct {
	running atomic.Int64
}

// https://go.dev/ref/mem#atomic
func (t *Tracker) Inc()           { t.running.Add(1) }
func (t *Tracker) Dec()           { t.running.Add(-1) }
func (t *Tracker) Running() int64 { return t.running.Load() }
