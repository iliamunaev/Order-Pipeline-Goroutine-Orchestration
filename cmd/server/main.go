// Order Pipeline is a concurrent order-processing HTTP server.
//
// It acts as the composition root: creates shared infrastructure
// (pool, tracker), builds pipeline steps as closures, wires the
// order orchestrator and HTTP handler, then starts the server.
//
// No other package imports all concrete service types — only main
// knows how the pieces fit together.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/order"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/courier"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/payment"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/pool"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/tracker"
	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/service/vendor"
	httptransport "github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/transport/http"
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

	// Create bounded concurrency semaphore
	p := pool.New(poolSize)

	// Set up goroutine tracker
	tr := &tracker.Tracker{}

	// Build the pipeline steps
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

	// Construct the order service
	orderSvc := order.New(steps)

	// Construct the HTTP handler
	h := httptransport.New(orderSvc, requestTimeout)

	// Set up routing
	mux := http.NewServeMux()
	mux.HandleFunc("/order", h.HandleOrder)

	// Configure the HTTP server
	srv := &http.Server{
		Addr:              "127.0.0.1:8080",
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
