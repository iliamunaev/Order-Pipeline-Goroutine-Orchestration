package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
)

func TestDelayForStep(t *testing.T) {
	t.Parallel()

	defaultDelay := 10 * time.Millisecond

	if got := delayForStep(nil, "payment", defaultDelay); got != defaultDelay {
		t.Fatalf("expected default delay, got %v", got)
	}

	delayMS := map[string]int64{"payment": 3}
	if got := delayForStep(delayMS, "payment", defaultDelay); got != 3*time.Millisecond {
		t.Fatalf("expected override delay, got %v", got)
	}

	delayMS["payment"] = 0
	if got := delayForStep(delayMS, "payment", defaultDelay); got != defaultDelay {
		t.Fatalf("expected default delay when override is 0, got %v", got)
	}
}

func TestProcessPayment(t *testing.T) {
	t.Parallel()

	tr := &Tracker{}
	req := model.OrderRequest{
		OrderID: "o-1",
		Amount:  1200,
		DelayMS: map[string]int64{"payment": 1},
	}

	if err := ProcessPayment(context.Background(), req, tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.FailStep = "payment"
	if err := ProcessPayment(context.Background(), req, tr); !errors.Is(err, apperr.ErrPaymentDeclined) {
		t.Fatalf("expected payment declined, got %v", err)
	}

	req.FailStep = ""
	req.Amount = 0
	if err := ProcessPayment(context.Background(), req, tr); !errors.Is(err, apperr.ErrPaymentDeclined) {
		t.Fatalf("expected payment declined for invalid amount, got %v", err)
	}
}

func TestNotifyVendor(t *testing.T) {
	t.Parallel()

	tr := &Tracker{}
	req := model.OrderRequest{
		OrderID: "o-2",
		Amount:  500,
		DelayMS: map[string]int64{"vendor": 1},
	}

	if err := NotifyVendor(context.Background(), req, tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.FailStep = "vendor"
	if err := NotifyVendor(context.Background(), req, tr); !errors.Is(err, apperr.ErrVendorUnavailable) {
		t.Fatalf("expected vendor unavailable, got %v", err)
	}
}

func TestAssignCourier(t *testing.T) {
	t.Parallel()

	tr := &Tracker{}
	pool := NewCourierPool(1)
	req := model.OrderRequest{
		OrderID: "o-3",
		Amount:  800,
		DelayMS: map[string]int64{"courier": 1},
	}

	if err := AssignCourier(context.Background(), req, pool, tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.FailStep = "courier"
	if err := AssignCourier(context.Background(), req, pool, tr); !errors.Is(err, apperr.ErrNoCourierAvailable) {
		t.Fatalf("expected no courier available, got %v", err)
	}
}

func TestAssignCourierContextTimeout(t *testing.T) {
	t.Parallel()

	tr := &Tracker{}
	pool := NewCourierPool(1)
	if err := pool.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	defer pool.Release()

	req := model.OrderRequest{
		OrderID: "o-4",
		Amount:  800,
		DelayMS: map[string]int64{"courier": 50},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := AssignCourier(ctx, req, pool, tr)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
