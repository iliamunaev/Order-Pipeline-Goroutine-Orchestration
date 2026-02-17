package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"order-pipeline/internal/app"
	"order-pipeline/internal/handler"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}


// Run initializes the application and handler, 
// creates a new HTTP server, and listens for requests.
// Config is used to initialize the application and handler.
// RequestTimeout is used to set the timeout for the HTTP server.
// RequestTimeout is used to protect the server from hanging/slow downstream calls,
// prevent goroutines/resources from being tied up too long, 
// give clients predictable failure instead of indefinite waiting, 
// and enable retries on client side.
func run() error {
	a := app.New(app.Config{Couriers: 5, RequestTimeout: 10 * time.Second})
	h := handler.New(a.OrderService)

	mux := http.NewServeMux()
	mux.HandleFunc("/order", h.HandleOrder)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
