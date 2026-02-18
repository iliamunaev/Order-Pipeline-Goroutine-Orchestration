// Package apperr provides error handling and HTTP status code mapping.
package apperr

import (
	"context"
	"errors"
	"net/http"

	"order-pipeline/internal/service/courier"
	"order-pipeline/internal/service/payment"
	"order-pipeline/internal/service/vendor"
)

type Info struct {
    Kind   string
    Status int
}

func classify(err error) Info {
    switch {
    case err == nil:
        return Info{"", http.StatusOK}
    case errors.Is(err, payment.ErrDeclined):
        return Info{"payment_declined", http.StatusBadRequest}
    case errors.Is(err, vendor.ErrUnavailable):
        return Info{"vendor_unavailable", http.StatusServiceUnavailable}
    case errors.Is(err, courier.ErrNoCourierAvailable):
        return Info{"no_courier", http.StatusServiceUnavailable}
    case errors.Is(err, context.DeadlineExceeded):
        return Info{"timeout", http.StatusGatewayTimeout}
    case errors.Is(err, context.Canceled):
        return Info{"canceled", http.StatusRequestTimeout}
    default:
        return Info{"internal", http.StatusInternalServerError}
    }
}

func Kind(err error) string    { return classify(err).Kind }
func HTTPStatus(err error) int { return classify(err).Status }
