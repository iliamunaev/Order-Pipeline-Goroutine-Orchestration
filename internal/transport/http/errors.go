package httptransport

import (
	"context"
	"errors"
	"net/http"
)

// kinder is satisfied by domain errors
// that carry a classification kind.
type kinder interface {
	Kind() string
}

// kindToStatus maps error classification kinds
// to HTTP status codes.
var kindToStatus = map[string]int{
	"payment_declined":   http.StatusBadRequest,
	"vendor_unavailable": http.StatusServiceUnavailable,
	"no_courier":         http.StatusServiceUnavailable,
	"timeout":            http.StatusGatewayTimeout,
	"canceled":           http.StatusRequestTimeout,
}

// errorKind returns the kind of an error.
func errorKind(err error) string {
	if err == nil {
		return ""
	}
	var k kinder
	if errors.As(err, &k) {
		return k.Kind()
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "internal"
	}
}

func httpStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if s, ok := kindToStatus[errorKind(err)]; ok {
		return s
	}
	return http.StatusInternalServerError
}
