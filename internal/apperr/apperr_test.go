package apperr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestKind(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("wrapped: %w", ErrPaymentDeclined)

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "payment_declined", err: ErrPaymentDeclined, want: "payment_declined"},
		{name: "payment_declined_wrapped", err: wrapped, want: "payment_declined"},
		{name: "vendor_unavailable", err: ErrVendorUnavailable, want: "vendor_unavailable"},
		{name: "no_courier", err: ErrNoCourierAvailable, want: "no_courier"},
		{name: "deadline", err: context.DeadlineExceeded, want: "timeout"},
		{name: "canceled", err: context.Canceled, want: "canceled"},
		{name: "unknown", err: errors.New("unknown"), want: "internal"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Kind(tt.err); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("wrapped: %w", ErrNoCourierAvailable)

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: http.StatusOK},
		{name: "payment_declined", err: ErrPaymentDeclined, want: http.StatusBadRequest},
		{name: "vendor_unavailable", err: ErrVendorUnavailable, want: http.StatusServiceUnavailable},
		{name: "no_courier", err: ErrNoCourierAvailable, want: http.StatusServiceUnavailable},
		{name: "no_courier_wrapped", err: wrapped, want: http.StatusServiceUnavailable},
		{name: "deadline", err: context.DeadlineExceeded, want: http.StatusGatewayTimeout},
		{name: "canceled", err: context.Canceled, want: http.StatusBadRequest},
		{name: "unknown", err: errors.New("unknown"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := HTTPStatus(tt.err); got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}
