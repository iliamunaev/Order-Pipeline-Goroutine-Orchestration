# Order Pipeline — Concurrent Order Processing Service

An HTTP service that processes orders by running payment, vendor notification,
and courier assignment steps **concurrently**. When any step fails or the
request times out, in-flight work is canceled immediately.

The project simulates a real-world delivery order flow to demonstrate
Go concurrency patterns, structured error handling, clean architecture, and
thorough testing.

## Package docs

[https://pkg.go.dev/github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration](https://pkg.go.dev/github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration)


## Features

- **Structured concurrency** — `errgroup.WithContext` launches parallel steps
  and cancels siblings on first failure.
- **Bounded concurrency** — a channel-based semaphore limits simultaneous
  courier assignments, preventing resource exhaustion.
- **Context propagation** — request timeouts flow through every goroutine;
  all blocking operations (`sleep`, `pool.Acquire`) respect `ctx.Done()`.
- **Modern Go idioms** — uses `sync.WaitGroup.Go` (Go 1.25+) to eliminate
  manual `Add`/`Done` pairing in tests.
- **Interface-driven design** — the HTTP handler depends on an `orderProcessor`
  interface, not concrete types. Services carry error semantics via structural
  typing (`Kind() string`) — no shared error package needed.
- **Injected pipeline steps** — `order.Service` receives `[]order.Step` at
  construction, keeping the orchestrator decoupled from concrete services.
- **Layered architecture** — transport (HTTP) / orchestration (order) / domain
  services are cleanly separated. Dependencies point inward.
- **Testing at every level** — unit tests with stubs, integration tests with
  real services, table-driven tests, stress tests (20k concurrent requests),
  race detection, benchmarks, and fuzz testing.

## Requirements

- Go 1.25+

## Quick start

### Run server:
```bash
git clone https://github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration
cd Order-Pipeline-Goroutine-Orchestration
make run
```

Server listens on `127.0.0.1:8080`.

### Make a request:

Examples:
**Success (200 OK)**

```bash
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o-1","amount":1200}'
```

```json
{
  "status": "ok",
  "order_id": "o-1",
  "steps": [
    { "name": "payment", "status": "ok", "duration_ms": 152 },
    { "name": "vendor",  "status": "ok", "duration_ms": 201 },
    { "name": "courier", "status": "ok", "duration_ms": 103 }
  ]
}
```

**Payment declined (400)**

```bash
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o-2","amount":10,"fail_step":"payment","delay_ms":{"vendor":800,"courier":800}}'
```

```json
{
  "status": "error",
  "order_id": "o-2",
  "steps": [
    { "name": "payment", "status": "error", "duration_ms": 151, "detail": "payment_declined" },
    { "name": "vendor",  "status": "canceled", "duration_ms": 152 },
    { "name": "courier", "status": "canceled", "duration_ms": 152 }
  ],
  "error": { "kind": "payment_declined", "message": "order failed" }
}
```

**Vendor unavailable (503)**

```bash
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o-3","amount":10,"fail_step":"vendor","delay_ms":{"vendor":100,"courier":800}}'
```

**Timeout (504)**

```bash
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o-4","amount":10,"delay_ms":{"payment":15000,"vendor":15000,"courier":15000}}'
```

## API

### `POST /order`

**Request body**

| Field       | Type              | Required | Description                                            |
|-------------|-------------------|----------|--------------------------------------------------------|
| `order_id`  | string            | yes      | Order identifier                                       |
| `amount`    | int               | no       | Payment amount (<=0 triggers `payment_declined`)       |
| `fail_step` | string            | no       | Force a failure: `"payment"`, `"vendor"`, `"courier"`  |
| `delay_ms`  | map[string]int    | no       | Per-step delay overrides in ms                         |

## Project layout

```
.
├── cmd
│   └── server
│       └── main.go                  composition root — wires steps, starts server
├── internal
│   ├── model
│   │   └── order.go                 request / response DTOs
│   ├── order
│   │   └── order.go                 orchestration — Step type, errgroup, deterministic results
│   ├── service
│   │   ├── courier
│   │   │   ├── courier.go           courier assignment with bounded concurrency
│   │   │   └── courier_test.go
│   │   ├── payment
│   │   │   ├── payment.go           payment validation and processing
│   │   │   └── payment_test.go
│   │   ├── pool
│   │   │   ├── pool.go              channel-based semaphore (1–128 slots)
│   │   │   └── pool_test.go
│   │   ├── tracker
│   │   │   ├── tracker.go           atomic in-flight counter
│   │   │   └── tracker_test.go
│   │   └── vendor
│   │       ├── vendor.go            vendor notification
│   │       └── vendor_test.go
│   └── transport
│       └── http
│           ├── errors.go            error-kind extraction + HTTP status mapping
│           ├── handler.go           HTTP handler — validate, delegate, respond
│           └── handler_test.go      unit + integration + stress tests
├── .github
│   └── workflows
│       └── go.yml                   CI pipeline (fmt → lint → test → race → fuzz)
├── go.mod
├── go.sum
├── Makefile
├── DEV-GUIDE.md                     detailed architecture guide
└── README.md
```

## Dependency layout

```
main.go
 ├── model
 ├── order          → model
 ├── httptransport  → model
 ├── payment        → model, tracker
 ├── vendor         → model, tracker
 ├── courier        → model, tracker
 ├── pool           → (stdlib only)
 └── tracker        → (stdlib only)
```

Dependencies point inward. The transport layer has zero imports of service
packages — it uses the `orderProcessor` interface. The order package has zero
imports of service packages — steps are injected via `[]order.Step`.

## Context tree 

```
r.Context()                           ← Level 0: HTTP request context (net/http)
  └─ context.WithTimeout(r.Context()) ← Level 1: handler adds 10s deadline
       └─ errgroup.WithContext(ctx)    ← Level 2: errgroup adds cancel-on-first-error
            ├─ payment.Process(ctx)    ← leaf: receives Level 2 ctx
            ├─ vendor.Notify(ctx)      ← leaf: receives Level 2 ctx
            └─ courier.Assign(ctx)     ← leaf: receives Level 2 ctx
```

## Testing and formatting

```bash
make ci              # fmt + vet + lint + race (quick pre-push check)
make test            # go test ./...
make test-race       # go test ./... -race -count=1
make test-bench      # benchmarks (pool throughput at various capacities)
make test-fuzz       # fuzz pool acquire/release (10s, override: FUZZ=FuzzName)
make test-cover      # coverage report
make fmt             # go fmt ./...
make vet             # go vet ./...
make lint            # golangci-lint
```

### Test strategy

| Layer          | What is tested                                             | Approach               |
|----------------|------------------------------------------------------------|------------------------|
| Handler        | HTTP method, JSON validation, error mapping, success path  | Stub-based unit tests  |
| Handler        | Error kind extraction + HTTP status mapping                | Table-driven           |
| Handler        | Payment failure cancels vendor + courier                   | Integration test       |
| Handler        | 20,000 concurrent requests with mixed outcomes             | Stress test            |
| Payment        | Success, decline, invalid amount                           | Table-driven           |
| Vendor         | Success, unavailable                                       | Table-driven           |
| Courier        | Success, failure, context timeout                          | Table-driven           |
| Pool           | Size clamping, acquire/release, blocking, timeout          | Table-driven + fuzz    |
| Pool           | Throughput at 1/2/8/64/128 capacity                        | Parallel benchmark     |
| Tracker        | Inc/dec correctness, concurrent safety (`WaitGroup.Go`)    | Parallel goroutines    |

### Coverage

```bash
make test-cover
go tool cover -html=coverage.out -o coverage.html
```

Then open `coverage.html` in a browser, or serve it:

```bash
python3 -m http.server 8000
# open http://localhost:8000/coverage.html
```

## Error mapping

| Error                          | Kind                 | HTTP   |
|--------------------------------|----------------------|--------|
| `payment.ErrDeclined`          | `payment_declined`   | 400    |
| `vendor.ErrUnavailable`        | `vendor_unavailable` | 503    |
| `courier.ErrNoCourierAvailable`| `no_courier`         | 503    |
| `context.DeadlineExceeded`     | `timeout`            | 504    |
| `context.Canceled`             | `canceled`           | 408    |
| unknown                        | `internal`           | 500    |

Each service defines a typed sentinel error with a `Kind() string` method
(structural typing). The transport layer's `errors.go` uses `errors.As` to
extract the kind from any error chain and maps it to an HTTP status via
`kindToStatus` — it has zero knowledge of service packages.

## CI

GitHub Actions pipeline (`.github/workflows/go.yml`) runs on every push
and pull request to `master`:

1. **fmt** — verifies `gofmt` formatting
2. **lint** — `golangci-lint` v2.4.0
3. **test** — builds and runs unit tests
4. **race** — runs tests with `-race`
5. **fuzz** — 10-second fuzz smoke test on pool

## Architecture notes

See [DEV-GUIDE.md](DEV-GUIDE.md) for a detailed walkthrough of the request
lifecycle, concurrency model, dependency flow, configuration, and design
decisions.
