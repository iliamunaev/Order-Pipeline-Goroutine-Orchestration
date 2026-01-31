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
	"order-pipeline/internal/service"
)

type Handler struct {
	pool *service.CourierPool
	tr   *service.Tracker
}

func New(pool *service.CourierPool, tr *service.Tracker) *Handler {
	if tr == nil {
		tr = &service.Tracker{}
	}
	return &Handler{pool: pool, tr: tr}
}

func (h *Handler) HandleOrder(w http.ResponseWriter, r *http.Request) {
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

	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, model.OrderResponse{
			Status:  "error",
			OrderID: req.OrderID,
			Error:   &model.ErrorPayload{Kind: "bad_request", Message: "order_id is required"},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	results := make(map[string]model.StepResult, 3)
	var mu sync.Mutex

	record := func(name string, fn func() error) func() error {
		return func() error {
			start := time.Now()
			err := fn()
			durMS := time.Since(start).Milliseconds()

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

	g.Go(record("payment", func() error { return service.ProcessPayment(ctx, req, h.tr) }))
	g.Go(record("vendor", func() error { return service.NotifyVendor(ctx, req, h.tr) }))
	g.Go(record("courier", func() error { return service.AssignCourier(ctx, req, h.pool, h.tr) }))

	err := g.Wait()

	resp := model.OrderResponse{
		Status:  "ok",
		OrderID: req.OrderID,
		Steps:   flattenResults(results),
	}

	if err != nil {
		resp.Status = "error"
		resp.Error = &model.ErrorPayload{
			Kind:    apperr.Kind(err),
			Message: "order failed",
		}
	}

	status := apperr.HTTPStatus(err)
	writeJSON(w, status, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

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
