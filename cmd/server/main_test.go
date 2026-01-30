package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-pipeline/internal/handler"
)

func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/order", handler.HandleOrder)

	return mux
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	mux := newMux()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", body)
	}
}

func TestOrderEndpoint(t *testing.T) {
	t.Parallel()

	mux := newMux()

	req := httptest.NewRequest(http.MethodPost, "/order", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, rec.Code)
	}
	if body := rec.Body.String(); body != "not implemented" {
		t.Fatalf("expected body %q, got %q", "not implemented", body)
	}
}
