# Order Pipeline

This project demonstrates how to orchestrate multiple concurrent steps in a safe, cancellable way. The goal is to run payment, vendor notification, and courier assignment in parallel, stop the whole workflow on failure/timeout, and keep concurrency bounded.

Approach:
- Structured concurrency with `errgroup.WithContext`
- `Context` deadlines to cancel slow work
- Bounded concurrency using a courier pool (`semaphore`)
- Thread‑safe result aggregation + lightweight tracking
- Consistent `error mapping` to response status/kind

## Notes
- The HTTP layer exists only to trigger the workflow; it is not the goal of the project.
- The project (and tests) focus on goroutine orchestration, cancellation, error propagation, and concurrency invariants (pool/tracker behavior).
## Requirements

- Go 1.24+

## Usage

```bash
git clone https://github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration
cd Order-Pipeline-Goroutine-Orchestration
```

```bash
go run ./cmd/server
```
Server listens on `:8080`.


In other console run:
```bash
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o1","amount":10}'
```

```bash
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o2","amount":10,"fail_step":"payment","delay_ms":{"vendor":800,"courier":800}}'
```

```bash
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o3","amount":10,"delay_ms":{"payment":3000,"vendor":3000,"courier":3000}}'
```

## Endpoint

- `POST /order` -> processes an order

### Order request body

```json
{
  "order_id": "o1",
  "amount": 1200,
  "fail_step": "payment",
  "delay_ms": {
    "payment": 150,
    "vendor": 200,
    "courier": 100
  }
}
```

- `fail_step` can be `payment`, `vendor`, or `courier`
- `delay_ms` overrides per-step delays in milliseconds


## Testing

Run the full test suite:

```bash
go test ./...
```

Run with the race detector:

```bash
go test ./... -race
```

## Project layout

```text
.
├── cmd
│   └── server
│       └── main.go
├── go.mod
├── go.sum
├── internal
│   ├── apperr
│   │   ├── apperr.go
│   │   └── apperr_test.go
│   ├── handler
│   │   ├── handler.go
│   │   └── handler_test.go
│   ├── model
│   │   └── order.go
│   └── service
│       ├── courier
│       │   ├── courier.go
│       │   └── courier_test.go
│       ├── payment
│       │   ├── payment.go
│       │   └── payment_test.go
│       ├── pool
│       │   ├── courier_pool.go
│       │   └── courier_pool_test.go
│       ├── shared
│       │   ├── delay.go
│       │   └── sleep.go
│       ├── tracker
│       │   ├── tracker.go
│       │   └── tracker_test.go
│       └── vendor
│           ├── vendor.go
│           └── vendor_test.go
└── README.md
```
