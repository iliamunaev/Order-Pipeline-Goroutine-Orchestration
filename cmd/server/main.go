package main

import (
	"log"
	"net/http"
	"time"

	"order-pipeline/internal/handler"
	"order-pipeline/internal/service"
)

func main() {
	mux := http.NewServeMux()
	pool := service.NewCourierPool(5)
	tracker := &service.Tracker{}
	handler := handler.New(pool, tracker)

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
