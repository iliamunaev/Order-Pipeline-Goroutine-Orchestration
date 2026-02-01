// Command server starts a minimal HTTP server that triggers the order workflow.
// The server exists to demonstrate goroutine orchestration.
package main

import (
	"log"
	"net/http"
	"time"

	"order-pipeline/internal/handler"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
)

// main starts the HTTP server, 
// builds dependencies, and registers routes
func main() {
	mux := http.NewServeMux()
	courierPool := pool.NewCourierPool(5) // concurrency limiter
	runTracker := &tracker.Tracker{} // tracking order processing steps
	handler := handler.New(courierPool, runTracker)

	// register routes
	mux.HandleFunc("/order", handler.HandleOrder)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,

		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
