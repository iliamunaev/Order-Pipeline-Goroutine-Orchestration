// Package httptransport implements the HTTP transport layer
// for order processing.
package httptransport

import (
	"context"
	"encoding/json"
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
	// Request validation
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req model.OrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.OrderResponse{
			Status: "error",
			Error:  &model.ErrorPayload{Kind: "bad_request", Message: "invalid JSON"},
		})
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, model.OrderResponse{
			Status: "error",
			Error:  &model.ErrorPayload{Kind: "bad_request", Message: "invalid JSON"},
		})
		return
	}

	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, model.OrderResponse{
			Status:  "error",
			OrderID: req.OrderID,
			Error:   &model.ErrorPayload{Kind: "bad_request", Message: "order_id is required"},
		})
		return
	}

	// Set a deadline for the entire pipline
	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()

	// Call the order processor
	steps, err := h.orderProcessor.Process(ctx, req)

	// Build the response
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

	// Write the response
	writeJSON(w, httpStatus(err), resp)
}

// writeJSON writes v as a JSON response with the given status code.
// The Content-Type is set to application/json.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
