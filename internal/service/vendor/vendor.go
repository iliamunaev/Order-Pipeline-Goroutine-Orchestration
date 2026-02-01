// Package vendor contains the vendor notification step for an order workflow.
// It simulates vendor delays and failures.
package vendor

import (
	"context"
	"fmt"
	"time"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
	"order-pipeline/internal/service/shared"
	"order-pipeline/internal/service/tracker"
)

// Notify runs the vendor notification step for an order.
func Notify(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker) error {
	tr.Inc()
	defer tr.Dec()

	delay := shared.DelayForStep(req.DelayMS, "vendor", 200*time.Millisecond)

	if err := shared.SleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "vendor" {
		return fmt.Errorf("vendor notify: %w", apperr.ErrVendorUnavailable)
	}

	return nil
}
