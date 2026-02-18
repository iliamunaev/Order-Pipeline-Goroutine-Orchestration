package vendor

import (
	"context"
	"errors"
	"testing"
	"time"

	"order-pipeline/internal/model"
	"order-pipeline/internal/service/tracker"
)

type fakeVendorService struct{}

func (fakeVendorService) DelayForStep(delayMS map[string]int64, step string, defaultMS time.Duration) time.Duration {
	if delayMS == nil {
		return defaultMS
	}
	if ms, ok := delayMS[step]; ok && ms >= 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultMS
}

func (fakeVendorService) SleepOrDone(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func TestNotify(t *testing.T) {
	t.Parallel()

	tr := &tracker.Tracker{}
	vs := fakeVendorService{}

	tests := []struct {
		name    string
		req     model.OrderRequest
		wantErr error
	}{
		{
			name: "success",
			req: model.OrderRequest{
				OrderID: "o-2",
				Amount:  500,
				DelayMS: map[string]int64{"vendor": 1},
			},
		},
		{
			name: "fail_step",
			req: model.OrderRequest{
				OrderID:  "o-3",
				Amount:   500,
				FailStep: "vendor",
				DelayMS:  map[string]int64{"vendor": 1},
			},
			wantErr: ErrUnavailable,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Notify(context.Background(), tt.req, tr, vs)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
