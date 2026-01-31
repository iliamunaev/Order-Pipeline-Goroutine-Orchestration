package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"order-pipeline/internal/handler"
	"order-pipeline/internal/middleware"
	"order-pipeline/internal/service"
)

func main() {
	// setup logging
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	// build dependencies
	mux := http.NewServeMux()
	pool := service.NewCourierPool(5)
	tracker := &service.Tracker{}
	handler := handler.NewOrderHandler(pool, tracker)

	// register routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("health_check: ok\n"))
	})

	mux.HandleFunc("/order", handler.HandleOrder)

	// wrap the mux with logging middleware
	srv := &http.Server{
		Addr:    ":8080",
		Handler: middleware.Logging(mux),

		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
