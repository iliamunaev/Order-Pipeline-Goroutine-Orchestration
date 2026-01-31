package courier

import (
	"context"
	"fmt"
	"time"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
	"order-pipeline/internal/service"
	"order-pipeline/internal/service/shared"
)

func Assign(ctx context.Context, req model.OrderRequest, pool *service.CourierPool, tr *service.Tracker) error {
	tr.Inc()
	defer tr.Dec()

	delay := shared.DelayForStep(req.DelayMS, "courier", 100*time.Millisecond)

	if err := pool.Acquire(ctx); err != nil {
		return err
	}
	defer pool.Release()

	if err := shared.SleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "courier" {
		return fmt.Errorf("courier assign: %w", apperr.ErrNoCourierAvailable)
	}

	return nil
}
