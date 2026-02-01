// Package apperr provides error handling and HTTP status code mapping.
package apperr

import (
	"context"
	"errors"
	"net/http"
)

var (
	ErrPaymentDeclined    = errors.New("payment declined")
	ErrVendorUnavailable  = errors.New("vendor unavailable")
	ErrNoCourierAvailable = errors.New("no courier available")
)

// Kind returns the kind of error as a string.
func Kind(err error) string {
	switch {
	case err == nil:
		return ""

	case errors.Is(err, ErrPaymentDeclined):
		return "payment_declined"

	case errors.Is(err, ErrVendorUnavailable):
		return "vendor_unavailable"

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

// HTTPStatus returns the HTTP status code for the error.
// It returns http.StatusOK for no error,
// http.StatusBadRequest for payment declined,
// http.StatusServiceUnavailable for vendor unavailable or no courier available,
// http.StatusGatewayTimeout for timeout,
// and http.StatusInternalServerError for other errors.
func HTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK

	case errors.Is(err, ErrPaymentDeclined):
		return http.StatusBadRequest

	case errors.Is(err, ErrVendorUnavailable),
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
