// Package handler exposes HTTP handlers that orchestrate the concurrent
// order-processing steps and shape responses.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"order-pipeline/internal/apperr"
	"order-pipeline/internal/model"
	"order-pipeline/internal/service/courier"
	"order-pipeline/internal/service/payment"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
	"order-pipeline/internal/service/vendor"
)

// Handler wires HTTP requests to order processing steps.
type Handler struct {
	pool *pool.Pool
	tr   *tracker.Tracker
}

// New creates a Handler with the provided dependencies.
func New(pool *pool.Pool, tr *tracker.Tracker) *Handler {
	if tr == nil {
		tr = &tracker.Tracker{}
	}
	return &Handler{pool: pool, tr: tr}
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

	// create an error group with the context
	g, ctx := errgroup.WithContext(ctx)

	// create a map to store the results
	results := make(map[string]model.StepResult, 3)
	var mu sync.Mutex

	// create a function to record the results
	record := func(name string, fn func() error) func() error {
		return func() error {
			start := time.Now()
			err := fn()
			durMS := time.Since(start).Milliseconds() // calculate the duration of the step

			st := "ok"
			detail := ""

			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					st = "canceled"
				} else {
					st = "error"
					detail = apperr.Kind(err)
				}
			}

			// store the results in the map
			mu.Lock()
			results[name] = model.StepResult{
				Name:       name,
				Status:     st,
				DurationMS: durMS,
				Detail:     detail,
			}
			mu.Unlock()

			return err
		}
	}

	// start the concurrent steps
	// record the results of the steps
	g.Go(record("payment", func() error { return payment.Process(ctx, req, h.tr) }))
	g.Go(record("vendor", func() error { return vendor.Notify(ctx, req, h.tr) }))
	g.Go(record("courier", func() error { return courier.Assign(ctx, req, h.pool, h.tr) }))

	err := g.Wait()

	// create the response
	resp := model.OrderResponse{
		Status:  "ok",
		OrderID: req.OrderID,
		Steps:   flattenResults(results),
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

// flattenResults returns a slice of step results in the order of
// payment, vendor, and courier.
func flattenResults(m map[string]model.StepResult) []model.StepResult {
	out := make([]model.StepResult, 0, 3)
	if r, ok := m["payment"]; ok {
		out = append(out, r)
	}
	if r, ok := m["vendor"]; ok {
		out = append(out, r)
	}
	if r, ok := m["courier"]; ok {
		out = append(out, r)
	}
	return out
}
