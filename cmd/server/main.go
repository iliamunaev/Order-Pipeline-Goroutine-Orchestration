package main

import (
	"log"
	"net/http"
	"time"

	"order-pipeline/internal/handler"
	"order-pipeline/internal/service"
)

func main() {
	// build dependencies
	mux := http.NewServeMux()
	pool := service.NewCourierPool(5)
	tracker := &service.Tracker{}
	handler := handler.New(pool, tracker)

	// register routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("health_check: ok\n"))
	})

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
