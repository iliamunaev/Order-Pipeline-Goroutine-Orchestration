package vendor

import (
	"context"
	"fmt"
	"time"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
	"order-pipeline/internal/service"
	"order-pipeline/internal/service/shared"
)

func Notify(ctx context.Context, req model.OrderRequest, tr *service.Tracker) error {
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
