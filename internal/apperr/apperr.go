package apperr

import (
	"context"
	"errors"
	"net/http"
)

var (
	ErrPaymentDeclined       = errors.New("payment declined")
	ErrRestaurantUnavailable = errors.New("restaurant unavailable")
	ErrNoCourierAvailable    = errors.New("no courier available")
)

func Kind(err error) string {
	switch {
	case err == nil:
		return ""

	case errors.Is(err, ErrPaymentDeclined):
		return "payment_declined"

	case errors.Is(err, ErrRestaurantUnavailable):
		return "restaurant_unavailable"

	case errors.Is(err, ErrNoCourierAvailable):
		return "no_courier"

	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"

	case errors.Is(err, context.Canceled):
		return "canceled"

	default:
		return "internal"
	}
}

func HTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK

	case errors.Is(err, ErrPaymentDeclined):
		return http.StatusBadRequest

	case errors.Is(err, ErrRestaurantUnavailable),
		errors.Is(err, ErrNoCourierAvailable):
		return http.StatusServiceUnavailable

	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout

	case errors.Is(err, context.Canceled):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
