package vendor

import (
	"context"
	"errors"
	"testing"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

func TestNotify(t *testing.T) {
	t.Parallel()

	tr := &tracker.Tracker{}

	tests := []struct {
		name    string
		req     model.OrderRequest
		tr      *tracker.Tracker
		wantErr error
	}{
		{
			name: "success",
			req: model.OrderRequest{
				OrderID: "o-2",
				Amount:  500,
				DelayMS: map[string]int64{"vendor": 1},
			},
			tr: tr,
		},
		{
			name: "fail_step",
			req: model.OrderRequest{
				OrderID:  "o-3",
				Amount:   500,
				FailStep: "vendor",
				DelayMS:  map[string]int64{"vendor": 1},
			},
			tr:      tr,
			wantErr: ErrUnavailable,
		},
		{
			name: "nil_tracker",
			req: model.OrderRequest{
				OrderID: "o-4",
				Amount:  500,
				DelayMS: map[string]int64{"vendor": 1},
			},
			tr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Notify(context.Background(), tt.req, tt.tr)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNotify_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := model.OrderRequest{
		OrderID: "o-5",
		Amount:  500,
		DelayMS: map[string]int64{"vendor": 100},
	}

	err := Notify(ctx, req, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
