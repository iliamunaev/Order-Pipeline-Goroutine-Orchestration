package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"order-pipeline/internal/model"
	"order-pipeline/internal/order"
	"order-pipeline/internal/service/courier"
	"order-pipeline/internal/service/payment"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
	"order-pipeline/internal/service/vendor"
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
	const poolSize = 5

	p := pool.New(poolSize)
	tr := &tracker.Tracker{}

	steps := []order.Step{
		{Name: "payment", Run: func(ctx context.Context, req model.OrderRequest) error {
			return payment.Process(ctx, req, tr)
		}},
		{Name: "vendor", Run: func(ctx context.Context, req model.OrderRequest) error {
			return vendor.Notify(ctx, req, tr)
		}},
		{Name: "courier", Run: func(ctx context.Context, req model.OrderRequest) error {
			return courier.Assign(ctx, req, p, tr)
		}},
	}

	orderSvc := order.New(steps)
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
