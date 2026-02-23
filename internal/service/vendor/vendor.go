// Package vendor implements the vendor notification step.
package vendor

import (
	"context"
	"fmt"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/shared"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

type unavailableError struct{}

func (unavailableError) Error() string { return "vendor unavailable" }
func (unavailableError) Kind() string  { return "vendor_unavailable" }

// ErrUnavailable is returned when the vendor cannot be reached.
var ErrUnavailable = unavailableError{}

// Notify runs the vendor notification step for an order.
func Notify(ctx context.Context, req model.OrderRequest, tr *tracker.Tracker) error {
	// Track the running step
	tr.Inc()
	defer tr.Dec()

	delay := shared.DelayForStep(req.DelayMS, "vendor", 200*time.Millisecond)

	if err := shared.SleepOrDone(ctx, delay); err != nil {
		return err
	}

	if req.FailStep == "vendor" {
		return fmt.Errorf("vendor notify: %w", ErrUnavailable)
	}

	return nil
}
