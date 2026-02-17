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

Clean port if needed:
```bash
sudo lsof -i :8080
```
```bash
kill <process>
```

## Endpoint

- `POST /order` -> processes an order

In other console run:
```bash
# 200 OK
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o1","amount":10}'
```

```bash
# 400 Bad Request
# "error":{"kind":"payment_declined","message":"order failed"}}
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o2","amount":10,"fail_step":"payment","delay_ms":{"vendor":800,"courier":800}}'
```

```bash
# 504 Gateway Timeout
# "error":{"kind":"timeout","message":"order failed"}}
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o3","amount":10,"delay_ms":{"payment":3000,"vendor":3000,"courier":3000}}'
```

```bash
# 503 Service Unavailable
# "error":{"kind":"vendor_unavailable","message":"order failed"}}
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o2","amount":10,"fail_step":"vendor","delay_ms":{"vendor":800,"courier":800}}'
```


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


## Testing

### All
```bash
make test-all
```

### All, except benchmark
```bash
make test
```

### Benchmark test only
```bash
make test-bench
```

### Fuzzing test only
```bash
make test-fuzz
```

### Race detector

```bash
make test-race
```

### Coverage:
```bash
go test ./... -cover
```

### Display coverage in a browser
_Requirements: python3_
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```
```bash
python3 -m http.server 8000
```
Open in a browser:
```bash
http://localhost:8000/coverage.html
```
Example:

############## Add images
