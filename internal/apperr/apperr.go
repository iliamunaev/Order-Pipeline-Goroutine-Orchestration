// Package apperr defines the AppError interface and helpers to extract
// error classification from any error in a chain.
package apperr

import (
	"context"
	"errors"
	"net/http"
)

// AppError is implemented by domain errors that carry their own
// HTTP status code and classification kind.
type AppError interface {
	error
	Kind() string
	HTTPStatus() int
}

func classify(err error) (kind string, status int) {
	if err == nil {
		return "", http.StatusOK
	}
	var ae AppError
	if errors.As(err, &ae) {
		return ae.Kind(), ae.HTTPStatus()
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout", http.StatusGatewayTimeout
	case errors.Is(err, context.Canceled):
		return "canceled", http.StatusRequestTimeout
	default:
		return "internal", http.StatusInternalServerError
	}
}

// Kind returns the error classification string for err.
func Kind(err error) string { k, _ := classify(err); return k }

// HTTPStatus returns the HTTP status code for err.
func HTTPStatus(err error) int { _, s := classify(err); return s }
