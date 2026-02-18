# order-pipeline — Developer Guide

Concurrent order-processing HTTP service in Go. Receives an order, runs
payment / vendor-notification / courier-assignment steps in parallel, and
returns a unified response with per-step outcomes.

External dependencies:
- `golang.org/x/sync` (errgroup).

---

## Package layout

```
.
├── cmd
│   └── server
│       └── main.go                  composition root — wires services, starts HTTP server
├── internal
│   ├── apperr
│   │   ├── apperr.go                AppError interface + Kind/HTTPStatus extractors
│   │   └── apperr_test.go
│   ├── model
│   │   └── order.go                 request / response DTOs
│   ├── order
│   │   └── order.go                 orchestration — runs steps concurrently via errgroup
│   ├── service
│   │   ├── courier
│   │   │   ├── courier.go           courier step — bounded-concurrency assignment
│   │   │   └── courier_test.go
│   │   ├── payment
│   │   │   ├── payment.go           payment step — validates amount, simulates decline
│   │   │   └── payment_test.go
│   │   ├── pool
│   │   │   ├── pool.go              channel-based semaphore (1–128 slots)
│   │   │   └── pool_test.go
│   │   ├── shared
│   │   │   └── shared.go            DelayForStep + SleepOrDone helpers
│   │   ├── tracker
│   │   │   ├── tracker.go           atomic counter for in-flight step monitoring
│   │   │   └── tracker_test.go
│   │   └── vendor
│   │       ├── vendor.go            vendor step — simulates notification delay / failure
│   │       └── vendor_test.go
│   └── transport
│       └── http
│           ├── handler.go           HTTP handler — decode, validate, delegate, respond
│           └── handler_test.go      unit tests (stub) + integration tests (real services)
├── go.mod
├── go.sum
├── Makefile
├── DEV-GUIDE.md
└── README.md
```

### Dependency flow

```
main  ──>  order  ──>  payment
       │          ├──>  vendor
       │          ├──>  courier ──> pool (limiter interface)
       │          ├──>  tracker
       │          └──>  apperr
       └──>  transport/http ──> apperr
```

Key rule: dependencies point inward. The transport layer knows nothing about
concrete services — it depends on the `orderProcessor` interface defined in
`handler.go`. Services know nothing about HTTP.

---

## How it works

### Request lifecycle

1. `HandleOrder` validates method (POST only) and JSON body (single object,
   no unknown fields, `order_id` required).
2. A `context.WithTimeout` wraps the request context with `requestTimeout`.
3. `order.Service.Process` launches three goroutines via `errgroup`:
   - `payment.Process` — sleep, then check `FailStep` / amount.
   - `vendor.Notify` — sleep, then check `FailStep`.
   - `courier.Assign` — acquire pool slot, sleep, then check `FailStep`.
4. When any step fails, errgroup cancels the derived context, which cancels
   the other in-flight steps.
5. `record` wraps each step to capture timing, status (`ok` / `error` /
   `canceled`), and error kind.
6. `flattenResults` returns steps in deterministic order (payment → vendor →
   courier) regardless of completion order.
7. The handler maps the first pipeline error to an HTTP status via
   `apperr.HTTPStatus` and writes a JSON response.

### Concurrency model

```
          errgroup (cancel-on-first-error)
         ┌──────────────────────────────────┐
         │                                  │
    ┌────┴────┐    ┌──────────┐    ┌───────┴───────┐
    │ payment │    │  vendor  │    │    courier    │
    └─────────┘    └──────────┘    │  pool.Acquire │
                                   │  ... work ... │
                                   │  pool.Release │
                                   └───────────────┘
```

- **errgroup** — structured concurrency with shared context. One failure
  cancels sibling goroutines.
- **pool.Pool** — channel-based semaphore. `Acquire` blocks until a slot
  opens or the context expires. Limits how many courier assignments run
  globally at once (configurable, 1–128).
- **tracker.Tracker** — atomic `Inc`/`Dec` counter. Every step increments on
  entry and decrements on exit. Useful for observability / drain checks.
- **SleepOrDone** — `time.NewTimer` + `select` on `ctx.Done()`. Properly
  stops the timer on cancellation (no goroutine leak).

### Error handling

Services define typed sentinel errors that implement `apperr.AppError`:

```go
type AppError interface {
    error
    Kind() string      // "payment_declined", "no_courier", ...
    HTTPStatus() int   // 400, 503, ...
}
```

Each service owns its error semantics — no central switch. `apperr.Kind` and
`apperr.HTTPStatus` use `errors.As` to find the first `AppError` in a
wrapped chain, with fallbacks for `context.DeadlineExceeded` (→ 504) and
`context.Canceled` (→ 408).

| Sentinel                       | Kind                 | HTTP status |
|--------------------------------|----------------------|-------------|
| `payment.ErrDeclined`          | `payment_declined`   | 400         |
| `vendor.ErrUnavailable`        | `vendor_unavailable` | 503         |
| `courier.ErrNoCourierAvailable`| `no_courier`         | 503         |
| `context.DeadlineExceeded`     | `timeout`            | 504         |
| `context.Canceled`             | `canceled`           | 408         |
| anything else                  | `internal`           | 500         |

---

## API

### `POST /order`

**Request**

```json
{
  "order_id": "o-123",
  "amount": 1200,
  "fail_step": "payment",
  "delay_ms": { "payment": 100, "vendor": 200, "courier": 150 }
}
```

- `order_id` (required) — order identifier.
- `amount` — payment amount; ≤ 0 triggers `payment_declined`.
- `fail_step` — force a step to fail (`"payment"` | `"vendor"` | `"courier"`).
- `delay_ms` — per-step delay overrides in milliseconds (defaults: payment 150ms, vendor 200ms, courier 100ms).

**Success (200)**

```json
{
  "status": "ok",
  "order_id": "o-123",
  "steps": [
    { "name": "payment", "status": "ok", "duration_ms": 102 },
    { "name": "vendor",  "status": "ok", "duration_ms": 201 },
    { "name": "courier", "status": "ok", "duration_ms": 153 }
  ]
}
```

**Error (4xx / 5xx)**

```json
{
  "status": "error",
  "order_id": "o-123",
  "steps": [
    { "name": "payment", "status": "error", "duration_ms": 105, "detail": "payment_declined" },
    { "name": "vendor",  "status": "canceled", "duration_ms": 105 },
    { "name": "courier", "status": "canceled", "duration_ms": 106 }
  ],
  "error": { "kind": "payment_declined", "message": "order failed" }
}
```

---

## Configuration

All values are constants in `cmd/server/main.go`:

| Parameter          | Value  | Purpose                                      |
|--------------------|--------|----------------------------------------------|
| `requestTimeout`   | 10 s   | Context deadline for the entire pipeline     |
| `pool size`        | 5      | Max concurrent courier assignments           |
| `Addr`             | :8080  | Listen address                               |
| `ReadTimeout`      | 10 s   | HTTP server read timeout                     |
| `ReadHeaderTimeout`| 3 s    | HTTP server header read timeout              |
| `WriteTimeout`     | 15 s   | HTTP server write timeout (requestTimeout + buffer) |
| `IdleTimeout`      | 60 s   | HTTP server keep-alive idle timeout          |

---

## Running

```bash
go run ./cmd/server

# test it
curl -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o-1","amount":1200,"delay_ms":{"payment":50,"vendor":50,"courier":50}}'
```

---

## Testing

```bash
make test            # all tests
make test-race       # with -race
make test-bench      # benchmarks (pool throughput at various capacities)
make test-fuzz       # fuzz pool acquire/release for 10s
make test-cover      # coverage report
make test-vet        # go vet
make test-all        # test + test-race + test-bench + test-fuzz
```

### Test strategy

- **Unit tests (stub-based)** — `handler_test.go` uses a `stubProcessor` to
  test HTTP validation, success responses, and error mapping in isolation
  from real services.
- **Integration tests** — `TestOrder_PaymentFailureCancelsOthers` exercises
  the full pipeline through real services and verifies cancellation
  propagation.
- **Stress test** — `TestHandler_Stress` fires 100 workers × 200 iterations
  (20,000 requests) with mixed success/failure scenarios, verifying
  correctness under concurrency. Skipped with `-short`.
- **Service tests** — each service package has table-driven tests for success
  and failure paths.
- **Pool tests** — size clamping, acquire/release blocking semantics, context
  timeout, parallel benchmark, fuzz test.
- **Tracker tests** — basic inc/dec, concurrent safety.
- **apperr tests** — kind and HTTP status extraction for every error type,
  including wrapped errors.

---

## Design decisions

**Why errgroup, not raw goroutines?** — errgroup gives cancel-on-first-error
for free. The derived context automatically propagates cancellation to
sibling steps, so a payment decline immediately stops vendor and courier work
instead of wasting resources.

**Why a channel semaphore instead of `semaphore.Weighted`?** — The channel
approach is zero-allocation on the hot path (send/receive on a buffered
channel) and integrates naturally with `select` + `ctx.Done()`. For this
use case (bounded integer slots, no weighted acquisition) it's simpler and
faster than `semaphore.Weighted`.

**Why typed error sentinels instead of `errors.New`?** — Each service's error
type implements `apperr.AppError` (via structural typing — no import needed).
This means `apperr` has zero knowledge of service packages. Adding a new
service with new error kinds requires no changes to `apperr`.

**Why `DisallowUnknownFields` + double decode?** — `DisallowUnknownFields`
rejects payloads with typos or extra fields early. The second `Decode` call
ensures the body contains exactly one JSON value (rejects concatenated
objects like `{...}{...}`).
