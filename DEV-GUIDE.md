# order-pipeline — Developer Guide

Concurrent order-processing HTTP service in Go. Receives an order, runs
payment / vendor-notification / courier-assignment steps in parallel, and
returns a unified response with per-step outcomes.

External dependency: `golang.org/x/sync` (errgroup).

---

## Package layout

```
.
├── cmd
│   └── server
│       └── main.go                  composition root — wires steps, starts HTTP server
├── internal
│   ├── model
│   │   └── order.go                 request / response DTOs
│   ├── order
│   │   └── order.go                 orchestration — Step type, errgroup, deterministic results
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
│           ├── errors.go            error-kind extraction + HTTP status mapping
│           ├── handler.go           HTTP handler — decode, validate, delegate, respond
│           └── handler_test.go      unit tests (stub) + integration + stress tests
├── .github
│   └── workflows
│       └── go.yml                   CI pipeline (fmt → lint → test → race → fuzz)
├── go.mod
├── go.sum
├── Makefile
├── DEV-GUIDE.md
└── README.md
```

### Dependency flow

```
main.go
 ├── model
 ├── order          → model
 ├── httptransport  → model
 ├── payment        → model, shared, tracker
 ├── vendor         → model, shared, tracker
 ├── courier        → model, shared, tracker
 ├── pool           → (stdlib only)
 └── tracker        → (stdlib only)
```

Key rules:
- Dependencies point inward. The transport layer knows nothing about
  concrete services — it depends on the `orderProcessor` interface defined
  in `handler.go`.
- The order package knows nothing about concrete services — steps are
  injected as `[]order.Step` values from `main.go`.
- Services know nothing about HTTP.
- `main.go` (composition root) is the only file that imports all concrete types.

---

## How it works

### Request lifecycle

1. `HandleOrder` validates method (POST only) and JSON body (single object,
   no unknown fields, `order_id` required).
2. A `context.WithTimeout` wraps the request context with `requestTimeout`.
3. `order.Service.Process` launches goroutines via `errgroup` — one per
   injected `Step`.
4. Each step runs concurrently:
   - `payment.Process` — sleep, then check `FailStep` / amount.
   - `vendor.Notify` — sleep, then check `FailStep`.
   - `courier.Assign` — acquire pool slot, sleep, then check `FailStep`.
5. When any step fails, errgroup cancels the derived context, which cancels
   the other in-flight steps.
6. Each step's outcome (timing, status, error kind) is recorded in a
   mutex-protected map keyed by step name.
7. After `g.Wait()`, results are flattened to a slice in registration order
   (payment → vendor → courier), producing deterministic output regardless
   of goroutine completion order.
8. The handler maps the pipeline error to an HTTP status via `errors.go`
   and writes a JSON response.

### Concurrency model

- **errgroup** — structured concurrency with shared context. One failure
  cancels sibling goroutines.
- **pool.Pool** — channel-based semaphore. `Acquire` blocks until a slot
  opens or the context expires. Limits how many courier assignments run
  globally at once (configurable, 1–128).
- **tracker.Tracker** — atomic `Inc`/`Dec` counter. Every step increments on
  entry and decrements on exit. Useful for observability / drain checks.
- **`sync.WaitGroup.Go`** (Go 1.25+) — used in tests to launch goroutines
  without manual `Add`/`Done` pairing. Eliminates a common source of
  deadlocks and panics.
- **SleepOrDone** — `time.NewTimer` + `select` on `ctx.Done()`. Properly
  stops the timer on cancellation (no goroutine leak).

### Error handling

Services define typed sentinel errors that implement a `Kind() string` method
via structural typing:

```go
type noCourierError struct{}

func (noCourierError) Error() string { return "no courier available" }
func (noCourierError) Kind() string  { return "no_courier" }
```

There is no shared error interface package. Both the `order` package and the
`transport/http` package define their own local `kinder` interface:

```go
type kinder interface {
    Kind() string
}
```

This is idiomatic Go — "accept interfaces at the consumer." Each package
discovers error kinds independently via `errors.As`, with zero coupling
to service packages.

The transport layer's `errors.go` maps kinds to HTTP statuses using a simple
`kindToStatus` map, with fallbacks for `context.DeadlineExceeded` (→ 504)
and `context.Canceled` (→ 408).

| Sentinel                       | Kind                 | HTTP status |
|--------------------------------|----------------------|-------------|
| `payment.ErrDeclined`          | `payment_declined`   | 400         |
| `vendor.ErrUnavailable`        | `vendor_unavailable` | 503         |
| `courier.ErrNoCourierAvailable`| `no_courier`         | 503         |
| `context.DeadlineExceeded`     | `timeout`            | 504         |
| `context.Canceled`             | `canceled`           | 408         |
| anything else                  | `internal`           | 500         |

### Step injection

The `order` package defines a `Step` struct:

```go
type Step struct {
    Name string
    Run  func(ctx context.Context, req model.OrderRequest) error
}
```

`main.go` constructs steps as closures that adapt service functions to this
signature, injecting shared dependencies (pool, tracker):

```go
steps := []order.Step{
    {Name: "payment", Run: func(ctx context.Context, req model.OrderRequest) error {
        return payment.Process(ctx, req, tr)
    }},
    {Name: "vendor", Run: func(ctx context.Context, req model.OrderRequest) error {
        return vendor.Notify(ctx, req, tr)
    }},
    {Name: "courier", Run: func(ctx context.Context, req model.OrderRequest) error {
        return courier.Assign(ctx, req, p, tr)
    }},
}
```

This keeps the orchestrator fully decoupled from service packages — it only
knows about `model.OrderRequest` and the `Step` contract.

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
make vet             # go vet
make lint            # golangci-lint
make test-all        # test + test-race + test-bench + test-fuzz
```

### Test strategy

- **Unit tests (stub-based)** — `handler_test.go` uses a `stubProcessor` to
  test HTTP validation, success responses, and error mapping in isolation
  from real services.
- **Error classification tests** — `handler_test.go` verifies `errorKind()`
  and `httpStatus()` for every sentinel error, wrapped errors, context
  errors, and unknown errors via table-driven tests.
- **Integration tests** — `TestOrder_PaymentFailureCancelsOthers` exercises
  the full pipeline through real services and verifies cancellation
  propagation.
- **Stress test** — `TestHandler_Stress` fires 100 workers × 200 iterations
  (20,000 requests) with mixed success/failure scenarios using
  `sync.WaitGroup.Go`, verifying correctness under concurrency.
  Skipped with `-short`.
- **Service tests** — each service package has table-driven tests for success
  and failure paths.
- **Courier timeout test** — `TestAssignContextTimeout` verifies that a
  blocked pool acquire returns `context.DeadlineExceeded` when the context
  expires.
- **Pool tests** — size clamping, acquire/release blocking semantics, context
  timeout, parallel benchmark at 1/2/8/64/128 capacity, fuzz test.
- **Tracker tests** — basic inc/dec, concurrent safety with 10 goroutines ×
  100 iterations using `sync.WaitGroup.Go`.

### CI

GitHub Actions (`.github/workflows/go.yml`) runs sequentially:

1. **fmt** — `gofmt -l .` (rejects unformatted files)
2. **lint** — `golangci-lint` with 5-minute timeout
3. **test** — `go build ./...` + `make test`
4. **race** — `make test-race` (parallel with fuzz)
5. **fuzz** — `make test-fuzz` 10-second smoke (parallel with race)

`concurrency` with `cancel-in-progress: true` cancels stale runs on the
same branch.

---

## Design decisions

**Errgroup, not raw goroutines** — errgroup gives cancel-on-first-error
for free. The derived context automatically propagates cancellation to
sibling steps, so a payment decline immediately stops vendor and courier work
instead of wasting resources.

**A channel semaphore instead of `semaphore.Weighted`** — The channel
approach is zero-allocation on the hot path (send/receive on a buffered
channel) and integrates naturally with `select` + `ctx.Done()`. For this
use case (bounded integer slots, no weighted acquisition) it's simpler and
faster than `semaphore.Weighted`.

**Typed error sentinels with `Kind()` instead of `errors.New`** — Each
service's error type carries a `Kind() string` method via structural typing.
The transport layer and the order package each define their own local `kinder`
interface to extract the kind via `errors.As`. This means adding a new service
with a new error kind requires zero changes to the handler or orchestrator.

**No shared error interface package** — The original design had an
`apperr` package with `HTTPStatus()` on domain errors, which leaked HTTP
concerns into the domain layer. The current design keeps error classification
(kind → HTTP status) entirely in `transport/http/errors.go` where it belongs.
Domain errors only carry `Kind()` — they have no knowledge of HTTP.

**`Step` struct instead of an interface** — A `Step` struct with `Name`
and `Run` fields is simpler and more flexible than a multi-method interface.
Any function with the right signature becomes a step — no adapter types needed.
The composition root builds steps as closures, naturally capturing dependencies.

**`DisallowUnknownFields` + double decode** — `DisallowUnknownFields`
rejects payloads with typos or extra fields early. The second `Decode` call
ensures the body contains exactly one JSON value (rejects concatenated
objects like `{...}{...}`).

**`WriteTimeout = requestTimeout + 5s`** — The pipeline has up to
`requestTimeout` to complete. `WriteTimeout` must be strictly larger to
allow the handler to write the response after the pipeline finishes or
times out. The 5-second buffer accounts for JSON encoding and I/O.

**`sync.WaitGroup.Go` in tests** — Go 1.25 introduced `WaitGroup.Go` which
handles `Add(1)` and `defer Done()` internally. The stress test and tracker
concurrency test use it to eliminate the `Add`/`Done` boilerplate and the
risk of mismatched calls. The production pipeline uses `errgroup.Go` for
its cancel-on-first-error semantics.
