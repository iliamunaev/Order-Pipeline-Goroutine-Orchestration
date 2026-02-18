package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"order-pipeline/internal/model"
	"order-pipeline/internal/order"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
)

// --- stubs for unit tests ---

type stubProcessor struct {
	steps []model.StepResult
	err   error
}

func (s *stubProcessor) Process(_ context.Context, _ model.OrderRequest) ([]model.StepResult, error) {
	return s.steps, s.err
}

// testAppErr satisfies apperr.AppError via structural typing.
type testAppErr struct {
	kind   string
	status int
}

func (e testAppErr) Error() string   { return e.kind }
func (e testAppErr) Kind() string    { return e.kind }
func (e testAppErr) HTTPStatus() int { return e.status }

// --- unit tests (stub-based) ---

func TestHandleOrderValidation(t *testing.T) {
	t.Parallel()

	h := New(&stubProcessor{}, 2*time.Second)

	tests := []struct {
		name       string
		method     string
		body       []byte
		wantStatus int
		wantKind   string
	}{
		{
			name:       "method_not_allowed",
			method:     http.MethodGet,
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "invalid_json",
			method:     http.MethodPost,
			body:       []byte(`{"order_id":`),
			wantStatus: http.StatusBadRequest,
			wantKind:   "bad_request",
		},
		{
			name:       "missing_order_id",
			method:     http.MethodPost,
			body:       []byte(`{"amount":10}`),
			wantStatus: http.StatusBadRequest,
			wantKind:   "bad_request",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "/order", bytes.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.HandleOrder(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantKind == "" {
				return
			}

			var out model.OrderResponse
			if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Error == nil || out.Error.Kind != tt.wantKind {
				t.Fatalf("expected error.kind=%s, got %+v", tt.wantKind, out.Error)
			}
		})
	}
}

func TestHandleOrder_Success(t *testing.T) {
	t.Parallel()

	stub := &stubProcessor{
		steps: []model.StepResult{
			{Name: "payment", Status: "ok", DurationMS: 10},
			{Name: "vendor", Status: "ok", DurationMS: 20},
			{Name: "courier", Status: "ok", DurationMS: 15},
		},
	}
	h := New(stub, 2*time.Second)

	body, _ := json.Marshal(model.OrderRequest{OrderID: "o-1", Amount: 100})
	req := httptest.NewRequest(http.MethodPost, "/order", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleOrder(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var out model.OrderResponse
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Status != "ok" {
		t.Fatalf("expected status=ok, got %q", out.Status)
	}
	if len(out.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(out.Steps))
	}
}

func TestHandleOrder_AppError(t *testing.T) {
	t.Parallel()

	stub := &stubProcessor{
		steps: []model.StepResult{
			{Name: "payment", Status: "error", Detail: "payment_declined"},
		},
		err: testAppErr{kind: "payment_declined", status: http.StatusBadRequest},
	}
	h := New(stub, 2*time.Second)

	body, _ := json.Marshal(model.OrderRequest{OrderID: "o-1", Amount: 100})
	req := httptest.NewRequest(http.MethodPost, "/order", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var out model.OrderResponse
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Status != "error" {
		t.Fatalf("expected status=error, got %q", out.Status)
	}
	if out.Error == nil || out.Error.Kind != "payment_declined" {
		t.Fatalf("expected error.kind=payment_declined, got %+v", out.Error)
	}
}

func TestNew_NilProcessorPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil processor")
		}
	}()
	New(nil, 2*time.Second)
}

func TestNew_DefaultTimeout(t *testing.T) {
	t.Parallel()

	h := New(&stubProcessor{}, 0)
	if h.requestTimeout != 2*time.Second {
		t.Fatalf("expected default timeout 2s, got %v", h.requestTimeout)
	}
}

// --- integration tests (real services) ---

func TestOrder_PaymentFailureCancelsOthers(t *testing.T) {
	p := pool.New(1)
	tr := &tracker.Tracker{}
	orderSvc := order.New(p, tr)
	h := New(orderSvc, 2*time.Second)

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

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var out model.OrderResponse
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
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

	p := pool.New(4)
	tr := &tracker.Tracker{}
	orderSvc := order.New(p, tr)
	h := New(orderSvc, 2*time.Second)

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
