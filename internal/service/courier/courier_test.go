package courier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/pool"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
)

func TestAssign(t *testing.T) {
	t.Parallel()

	tr := &tracker.Tracker{}
	pool := pool.New(1)

	tests := []struct {
		name    string
		req     model.OrderRequest
		wantErr error
	}{
		{
			name: "success",
			req: model.OrderRequest{
				OrderID: "o-3",
				Amount:  800,
				DelayMS: map[string]int64{"courier": 1},
			},
		},
		{
			name: "fail_step",
			req: model.OrderRequest{
				OrderID:  "o-4",
				Amount:   800,
				FailStep: "courier",
				DelayMS:  map[string]int64{"courier": 1},
			},
			wantErr: ErrNoCourierAvailable,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Assign(context.Background(), tt.req, pool, tr)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestAssignContextTimeout(t *testing.T) {
	t.Parallel()

	tr := &tracker.Tracker{}
	pool := pool.New(1)
	if err := pool.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	defer pool.Release()

	req := model.OrderRequest{
		OrderID: "o-4",
		Amount:  800,
		DelayMS: map[string]int64{"courier": 50},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := Assign(ctx, req, pool, tr)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
