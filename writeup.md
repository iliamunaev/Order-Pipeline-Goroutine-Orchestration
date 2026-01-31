# Go Concurrency & Error-Handling Write-Up
**Project: Order Pipeline (Payment → Vendor → Courier)**

## Project summary

This project models a backend order flow where **three independent steps** run in parallel:

- `payment` — business validation / payment provider
- `vendor` — notify external vendor system
- `courier` — assign a limited shared resource

The system is designed to:
- run steps concurrently
- cancel remaining work on failure
- classify errors meaningfully
- shut down cleanly without goroutine leaks
- be testable under the race detector

The implementation is intentionally small but mirrors real backend concerns.

---

## Question 1  
### What are some ways to orchestrate goroutines? How do you wait for multiple goroutines? Error handling? Gotchas?

### Approaches available
- `sync.WaitGroup` — basic synchronization, no error propagation
- channels — flexible but easy to leak or deadlock
- `errgroup.Group` — structured concurrency with cancellation

 I need to call an external service, but I must do it in parallel, with the constraint that if any request fails, the handler should return immediately.

### Chosen approach
**`errgroup.WithContext`**

### Why
- Combines **waiting**, **first-error propagation**, and **cancellation**
- Enforces a parent-child relationship between goroutines
- Avoids fire-and-forget goroutines

In the handler:
```go
g, ctx := errgroup.WithContext(ctx)

g.Go(func() error { return Process(ctx, req, tr) })
g.Go(func() error { return Notify(ctx, req, tr) })
g.Go(func() error { return Assign(ctx, req, pool, tr) })

err := g.Wait()
```

If any step fails:
- `errgroup` returns the error
- the shared context is canceled
- remaining goroutines exit cooperatively

### Gotchas encountered
- **Cancellation is not retroactive**  
  A fast goroutine may complete before cancellation occurs.
- **Blocking operations must be context-aware**  
  `time.Sleep`, channel sends, and semaphore acquisition must observe `ctx.Done()`.

### Error handling
- Each step returns wrapped sentinel errors (`fmt.Errorf("%w", ErrX)`).
- The handler classifies errors centrally using `errors.Is`.

---

## Question 2  
### How do you cancel background goroutines? How do you verify they exited?

### Cancellation mechanism
- Request-scoped `context.WithTimeout`
- `errgroup.WithContext` propagates cancellation automatically

All blocking points are context-aware:
```go
select {
case <-timer.C:
case <-ctx.Done():
    return ctx.Err()
}
```

### Real scenario modeled
- Payment failure cancels vendor notification and courier assignment
- Request timeout cancels all steps

### Verification
I did not rely on assumptions.

An **atomic tracker** counts active step goroutines:
```go
type Tracker struct {
    running atomic.Int64
}
```

Each step:
```go
tr.Inc()
defer tr.Dec()
```

Tests wait until `tracker.Running() == 0` after handler completion.  
This detects leaked or stuck goroutines deterministically.

---

## Question 3  
### How do you differentiate error kinds and take different actions?

### Error strategy
- Use **sentinel errors** for semantic categories
- Wrap errors with `%w`
- Classify centrally with `errors.Is`

Sentinel errors:
- `ErrPaymentDeclined` — business error (no retry)
- `ErrVendorUnavailable` — transient dependency failure
- `ErrNoCourierAvailable` — resource exhaustion

Classification:
```go
switch {
case errors.Is(err, ErrPaymentDeclined):
    return "payment_declined"
case errors.Is(err, context.DeadlineExceeded):
    return "timeout"
}
```

HTTP mapping:
- payment declined → `400`
- capacity errors → `503`
- timeout → `504`

### Why this matters
- Callers can react appropriately (retry vs fix input)
- Internal errors are not leaked
- Error handling remains stable even when errors are wrapped

---

## Question 4  
### OS threads vs goroutines? Scheduling model? Practical implications?

### Key differences
- Goroutines are user-space, lightweight (KB stacks)
- OS threads are kernel-scheduled and heavyweight
- Goroutines multiplex onto OS threads (M:N model)

### Scheduling
- Go uses **preemptive scheduling**
- Since Go 1.14, goroutines can be asynchronously preempted
- Blocking syscalls and cgo still affect scheduling

### Practical implications in this project
- Goroutines must not block indefinitely
- Resource usage is bounded via a courier pool (channel semaphore)
- Context-aware blocking prevents scheduler starvation

---

## Question 5  
### Improving test coverage, catching bugs, and data races

### Risks addressed
- Goroutine leaks on cancellation
- Steps ignoring context
- Data races when collecting step results

### Test design
- Handler-level tests using `httptest`
- Deterministic cancellation via injected delays
- Timeout tests covering `DeadlineExceeded`

### Edge cases
- payment failure
- request timeout
- invalid JSON
- wrong HTTP method

### Data race detection
```bash
go test -race ./...
```

Shared state is protected using `sync.Mutex` or atomics.

### Regression prevention
Any future change that leaks goroutines or blocks cancellation causes tests to fail.

---

## Final takeaway

This project demonstrates:
- Structured concurrency
- Cooperative cancellation
- Semantic error handling
- Explicit verification of correctness
- Production-grade concurrency habits

The scope is small, but the patterns scale directly to real backend services.

