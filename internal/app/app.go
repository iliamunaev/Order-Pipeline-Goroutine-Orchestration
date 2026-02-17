// internal/app/app.go
package app

import (
	"time"

	"order-pipeline/internal/app/order"
	"order-pipeline/internal/service/pool"
	"order-pipeline/internal/service/tracker"
)

type Config struct {
	Couriers       int
	RequestTimeout time.Duration
}

type App struct {
	OrderService   *order.Service
	RequestTimeout time.Duration
}

func New(cfg Config) *App {
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 2 * time.Second
	}
	p := pool.New(cfg.Couriers)
	tr := &tracker.Tracker{}

	return &App{
		OrderService:   order.New(p, tr),
		RequestTimeout: cfg.RequestTimeout,
	}
}
