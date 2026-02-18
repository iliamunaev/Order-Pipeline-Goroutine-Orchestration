// Package vendor contains the vendor notification step for an order workflow.
// It simulates vendor delays and failures.
package vendor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"order-pipeline/internal/model"
	"order-pipeline/internal/service/tracker"
)

var ErrUnavailable = errors.New("vendor unavailable")

type Helper interface {
	DelayForStep(delayMS map[string]int64, step string, defaultMS time.Duration) time.Duration
	SleepOrDone(ctx context.Context, delay time.Duration) error
}

// Notify runs the vendor notification step for an order.
// It increments the tracker,
// sleeps for the delay,
// returns an error if the vendor fails,
// and returns nil if the vendor succeeds.
func Notify(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker, h Helper) error {
	if h == nil {
		return fmt.Errorf("vendor notify: nil helper")
	}
	// increment the tracker
	tr.Inc()
	defer tr.Dec()

	// delay the vendor notification, use the default delay if no delay is provided
	delay := h.DelayForStep(req.DelayMS, "vendor", 200*time.Millisecond)

	// sleep or done, return an error if the context is cancelled or deadline exceeded
	if err := h.SleepOrDone(ctx, delay); err != nil {
		return err
	}

	// if the vendor fails, return an error
	if req.FailStep == "vendor" {
		return fmt.Errorf("vendor notify: %w", ErrUnavailable)
	}

	return nil
}
