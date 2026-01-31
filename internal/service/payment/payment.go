package payment

import (
	"context"
	"fmt"
	"time"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
	"order-pipeline/internal/service"
	"order-pipeline/internal/service/shared"
)

func Process(ctx context.Context, req model.OrderRequest, tr *service.Tracker) error {
	tr.Inc()
	defer tr.Dec()

	delay := shared.DelayForStep(req.DelayMS, "payment", 150*time.Millisecond)

	if err := shared.SleepOrDone(ctx, delay); err != nil {
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
