// Package httptransport implements the HTTP transport layer
// for order processing.
package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
)

type orderProcessor interface {
	Process(ctx context.Context, req model.OrderRequest) ([]model.StepResult, error)
}

// Handler handles HTTP requests to order orchestration.
type Handler struct {
	orderProcessor orderProcessor
	requestTimeout time.Duration
}

// New returns a Handler configured with the given orderProcessor
// and request timeout.
//
// It panics if orderProcessor is nil. If requestTimeout is non-positive,
// a default timeout is applied.
func New(orderProcessor orderProcessor, requestTimeout time.Duration) *Handler {
	if orderProcessor == nil {
		panic("handler.New: nil order processor")
	}
	if requestTimeout <= 0 {
		requestTimeout = 2 * time.Second
	}
	return &Handler{
		orderProcessor: orderProcessor,
		requestTimeout: requestTimeout,
	}
}

// HandleOrder processes an order request.
//
// The request must be a POST with a valid JSON body.
// Processing is executed with a per-request timeout.
// The response always contains a structured OrderResponse.
func (h *Handler) HandleOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req model.OrderRequest
	if err := decodeStrictJSON(r, &req); err != nil {
		badRequest(w, "invalid JSON")
		return
	}

	if req.OrderID == "" {
		badRequest(w, "order_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()

	steps, err := h.orderProcessor.Process(ctx, req)

	resp := model.OrderResponse{
		Status:  "ok",
		OrderID: req.OrderID,
		Steps:   steps,
	}
	if err != nil {
		resp.Status = "error"
		resp.Error = &model.ErrorPayload{
			Kind:    errorKind(err),
			Message: "order failed",
		}
	}

	writeJSON(w, httpStatus(err), resp)
}

// decodeStrictJSON decodes the JSON request body into the given destination.
// It disallows unknown fields and enforces a single JSON value in the body.
func decodeStrictJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	// Enforce a single JSON value in the body.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("multiple JSON values")
	}

	return nil
}

// badRequest writes a JSON response with a 400 status code and a bad_request error.
// It is used to respond to requests with invalid JSON.
func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, model.OrderResponse{
		Status: "error",
		Error:  &model.ErrorPayload{Kind: "bad_request", Message: msg},
	})
}

// writeJSON writes v as a JSON response with the given status code.
// The Content-Type is set to application/json.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
