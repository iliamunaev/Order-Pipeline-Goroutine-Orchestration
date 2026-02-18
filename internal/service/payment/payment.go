// Package payment contains the payment step logic for an order workflow.
// It validates amounts and simulates failures and delays.
package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"order-pipeline/internal/model"
	shared "order-pipeline/internal/service/shared"
	"order-pipeline/internal/service/tracker"
)

var ErrDeclined = errors.New("payment declined")

// Process runs the payment step for an order.
// It increments the tracker,
// sleeps for the delay,
// returns an error if the payment fails,
// and returns an error if the amount is less than or equal to 0.
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
