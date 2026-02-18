package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"order-pipeline/internal/order"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
	httptransport "order-pipeline/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run wires services and starts the HTTP server.
// RequestTimeout bounds downstream calls so clients get predictable failures
// and goroutines are not tied up indefinitely.
func run() error {
	const requestTimeout = 10 * time.Second

	p := pool.New(5)
	tr := &tracker.Tracker{}
	orderSvc := order.New(p, tr)
	h := httptransport.New(orderSvc, requestTimeout)

	mux := http.NewServeMux()
	mux.HandleFunc("/order", h.HandleOrder)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      requestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
