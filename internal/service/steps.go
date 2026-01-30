package service

import (
	"context"
	"fmt"
	"time"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
)

func sleepOrDone(ctx context.Context, d time.Duration) error {
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

func ProcessPayment(ctx context.Context, req model.OrderRequest, tr *Tracker) error {
	tr.Inc()
	defer tr.Dec()

	delay := delayForStep(req.DelayMS, "payment", 150*time.Millisecond)

	if err := sleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "payment" {
		return fmt.Errorf("payment: %w", apperr.ErrPaymentDeclined)
	}

	if req.Amount <= 0 {
		return fmt.Errorf("payment: %w", apperr.ErrPaymentDeclined)
	}

	return nil
}

func NotifyVendor(ctx context.Context, req model.OrderRequest, tr *Tracker) error {
	tr.Inc()
	defer tr.Dec()

	delay := delayForStep(req.DelayMS, "vendor", 200*time.Millisecond)

	if err := sleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "vendor" {
		return fmt.Errorf("vendor notify: %w", apperr.ErrVendorUnavailable)
	}

	return nil
}

func AssignCourier(ctx context.Context, req model.OrderRequest, pool *CourierPool, tr *Tracker) error {
	tr.Inc()
	defer tr.Dec()

	delay := delayForStep(req.DelayMS, "courier", 100*time.Millisecond)

	if err := pool.Acquire(ctx); err != nil {
		return err
	}
	defer pool.Release()

	if err := sleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "courier" {
		return fmt.Errorf("courier assign: %w", apperr.ErrNoCourierAvailable)
	}

	return nil
}
