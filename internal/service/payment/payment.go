// Package payment implements the payment processing step.
package payment

import (
	"context"
	"fmt"
	"time"

	"order-pipeline/internal/model"
	"order-pipeline/internal/service/shared"
	"order-pipeline/internal/service/tracker"
)

type declinedError struct{}

func (declinedError) Error() string { return "payment declined" }
func (declinedError) Kind() string  { return "payment_declined" }

// ErrDeclined is returned when a payment is declined or the amount is invalid.
var ErrDeclined = declinedError{}

// Process runs the payment step for an order.
func Process(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker) error {
	tr.Inc()
	defer tr.Dec()

	delay := shared.DelayForStep(req.DelayMS, "payment", 150*time.Millisecond)

	if err := shared.SleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "payment" {
		return fmt.Errorf("payment: %w", ErrDeclined)
	}

	if req.Amount <= 0 {
		return fmt.Errorf("payment: %w", ErrDeclined)
	}

	return nil
}
