package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"order-pipeline/internal/handler"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
)

// main starts the HTTP server and listens on port 8080.
func main() {
	srv := newServer(":8080")
	log.Printf("listening on %s", srv.Addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// newMux creates a new HTTP serve mux with the given handler.
// It creates a new courier pool with 5 couriers and a new tracker.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	courierPool := pool.New(5)
	tr := &tracker.Tracker{}
	h := handler.New(courierPool, tr)

	mux.HandleFunc("/order", h.HandleOrder)
	return mux
}

// newServer creates a new HTTP server with the given address.
// It sets the read, read header, write, and idle timeouts.
func newServer(addr string) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: newMux(),

		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
