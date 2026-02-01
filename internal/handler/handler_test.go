package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"order-pipeline/internal/model"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
)

func TestOrder_PaymentFailureCancelsOthers(t *testing.T) {
	courierPool := pool.New(1)
	tr := &tracker.Tracker{}
	h := New(courierPool, tr)

	// Make vendor + courier slower than payment so they get canceled.
	reqBody := model.OrderRequest{
		OrderID:  "o-test",
		Amount:   10,
		FailStep: "payment",
		DelayMS: map[string]int64{
			"payment": 150,
			"vendor":  800,
			"courier": 800,
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/order", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleOrder(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var out model.OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if out.Status != "error" {
		t.Fatalf("expected status=error, got %q", out.Status)
	}
	if out.Error == nil || out.Error.Kind != "payment_declined" {
		t.Fatalf("expected error.kind=payment_declined, got %+v", out.Error)
	}

	if len(out.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(out.Steps))
	}

	if out.Steps[0].Name != "payment" || out.Steps[0].Status != "error" {
		t.Fatalf("expected payment:error, got %+v", out.Steps[0])
	}

	if out.Steps[1].Name != "vendor" || out.Steps[1].Status != "canceled" {
		t.Fatalf("expected vendor:canceled, got %+v", out.Steps[1])
	}

	if out.Steps[2].Name != "courier" || out.Steps[2].Status != "canceled" {
		t.Fatalf("expected courier:canceled, got %+v", out.Steps[2])
	}
}

func waitRunningZero(t *testing.T, tr *tracker.Tracker) {
	t.Helper()

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if tr.Running() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected running steps to reach 0, got %d", tr.Running())
}

func TestHandler_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	courierPool := pool.New(4)
	tr := &tracker.Tracker{}
	h := New(courierPool, tr)

	const workers = 100
	const iterations = 200

	var wg sync.WaitGroup
	errCh := make(chan error, workers*iterations)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				idx := worker*iterations + i
				reqBody := model.OrderRequest{
					OrderID: fmt.Sprintf("o-%d", idx),
					Amount:  1200,
					DelayMS: map[string]int64{
						"payment": 1,
						"vendor":  1,
						"courier": 1,
					},
				}

				expectedStatus := http.StatusOK
				if idx%5 == 0 {
					reqBody.FailStep = "payment"
					expectedStatus = http.StatusBadRequest
				} else if idx%7 == 0 {
					reqBody.FailStep = "vendor"
					expectedStatus = http.StatusServiceUnavailable
				} else if idx%9 == 0 {
					reqBody.FailStep = "courier"
					expectedStatus = http.StatusServiceUnavailable
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					errCh <- fmt.Errorf("marshal: %w", err)
					continue
				}

				req := httptest.NewRequest(http.MethodPost, "/order", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				h.HandleOrder(w, req)

				resp := w.Result()
				if resp.StatusCode != expectedStatus {
					errCh <- fmt.Errorf("order %d expected status %d, got %d", idx, expectedStatus, resp.StatusCode)
					continue
				}

				var out model.OrderResponse
				if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
					errCh <- fmt.Errorf("decode response %d: %w", idx, err)
					continue
				}

				if expectedStatus == http.StatusOK && out.Status != "ok" {
					errCh <- fmt.Errorf("order %d expected status ok, got %q", idx, out.Status)
				}
				if expectedStatus != http.StatusOK && out.Status != "error" {
					errCh <- fmt.Errorf("order %d expected status error, got %q", idx, out.Status)
				}
			}
		}(w)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}

	waitRunningZero(t, tr)
}
