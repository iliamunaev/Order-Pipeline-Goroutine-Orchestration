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
	p := pool.New(1)

	tests := []struct {
		name    string
		req     model.OrderRequest
		tr      *tracker.Tracker
		wantErr error
	}{
		{
			name: "success",
			req: model.OrderRequest{
				OrderID: "o-3",
				Amount:  800,
				DelayMS: map[string]int64{"courier": 1},
			},
			tr: tr,
		},
		{
			name: "fail_step",
			req: model.OrderRequest{
				OrderID:  "o-4",
				Amount:   800,
				FailStep: "courier",
				DelayMS:  map[string]int64{"courier": 1},
			},
			tr:      tr,
			wantErr: ErrNoCourierAvailable,
		},
		{
			name: "nil_tracker",
			req: model.OrderRequest{
				OrderID: "o-5",
				Amount:  800,
				DelayMS: map[string]int64{"courier": 1},
			},
			tr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Assign(context.Background(), tt.req, p, tt.tr)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// Pool is pre-saturated so Assign blocks until the context expires.
func TestAssignContextTimeout(t *testing.T) {
	t.Parallel()

	p := pool.New(1)
	if err := p.Acquire(context.Background()); err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	defer p.Release()

	req := model.OrderRequest{
		OrderID: "o-6",
		Amount:  800,
		DelayMS: map[string]int64{"courier": 50},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := Assign(ctx, req, p, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestAssign_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := pool.New(1)
	req := model.OrderRequest{
		OrderID: "o-7",
		Amount:  800,
		DelayMS: map[string]int64{"courier": 100},
	}

	err := Assign(ctx, req, p, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
