package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"order-pipeline/internal/handler"
	"order-pipeline/internal/service"
)

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	pool := service.NewCourierPool(1)
	tracker := &service.Tracker{}
	Handler := handler.New(pool, tracker)

	mux.HandleFunc("/order", Handler.HandleOrder)

	return mux
}

func TestOrderEndpoint(t *testing.T) {
	t.Parallel()

	mux := newMux()

	body := map[string]any{
		"order_id": "o-1",
		"amount":   1200,
		"delay_ms": map[string]int64{
			"payment": 1,
			"vendor":  1,
			"courier": 1,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/order", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Fatalf("expected ok status in response, got %q", rec.Body.String())
	}
}
