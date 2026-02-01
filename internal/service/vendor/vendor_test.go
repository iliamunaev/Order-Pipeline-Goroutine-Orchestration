package vendor

import (
	"context"
	"errors"
	"testing"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
	"order-pipeline/internal/service/tracker"
)

func TestNotify(t *testing.T) {
	t.Parallel()

	tr := &tracker.Tracker{}

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
			wantErr: apperr.ErrVendorUnavailable,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Notify(context.Background(), tt.req, tr)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
