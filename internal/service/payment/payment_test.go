package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

func TestProcess(t *testing.T) {
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
				OrderID: "o-1",
				Amount:  1200,
				DelayMS: map[string]int64{"payment": 1},
			},
			tr: tr,
		},
		{
			name: "fail_step",
			req: model.OrderRequest{
				OrderID:  "o-2",
				Amount:   1200,
				FailStep: "payment",
				DelayMS:  map[string]int64{"payment": 1},
			},
			tr:      tr,
			wantErr: ErrDeclined,
		},
		{
			name: "invalid_amount",
			req: model.OrderRequest{
				OrderID: "o-3",
				Amount:  0,
				DelayMS: map[string]int64{"payment": 1},
			},
			tr:      tr,
			wantErr: ErrDeclined,
		},
		{
			name: "nil_tracker",
			req: model.OrderRequest{
				OrderID: "o-4",
				Amount:  500,
				DelayMS: map[string]int64{"payment": 1},
			},
			tr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Process(context.Background(), tt.req, tt.tr)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestProcess_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := model.OrderRequest{
		OrderID: "o-6",
		Amount:  100,
		DelayMS: map[string]int64{"payment": 100},
	}

	err := Process(ctx, req, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
