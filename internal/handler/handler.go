// Package handler exposes HTTP handlers that orchestrate the concurrent
// order-processing steps and shape responses.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	apporder "order-pipeline/internal/app/order"
	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
)

// Handler wires HTTP requests to order orchestration.
type Handler struct {
	orderSvc *apporder.Service
}

// New creates a Handler with the provided order service.
func New(orderSvc *apporder.Service) *Handler {
	return &Handler{orderSvc: orderSvc}
}

// HandleOrder processes an order request.
// It validates the request, orchestrates the concurrent steps,
// and writes the response.
func (h *Handler) HandleOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req model.OrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // prevent JSON injection
	// decode the request
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.OrderResponse{
			Status: "error",
			Error:  &model.ErrorPayload{Kind: "bad_request", Message: "invalid JSON"},
		})
		return
	}

	// validate the request
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, model.OrderResponse{
			Status:  "error",
			OrderID: req.OrderID,
			Error:   &model.ErrorPayload{Kind: "bad_request", Message: "order_id is required"},
		})
		return
	}

	// create a context with a timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	steps, err := h.orderSvc.Process(ctx, req)

	// create the response
	resp := model.OrderResponse{
		Status:  "ok",
		OrderID: req.OrderID,
		Steps:   steps,
	}

	// if there was an error, set the response to error
	if err != nil {
		resp.Status = "error"
		resp.Error = &model.ErrorPayload{
			Kind:    apperr.Kind(err),
			Message: "order failed",
		}
	}

	status := apperr.HTTPStatus(err)
	writeJSON(w, status, resp) // write the response
}

// writeJSON writes the response to the HTTP writer
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

