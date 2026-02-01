## Go Question 1: Goroutine orchestration

**Options**
- `sync.WaitGroup` for waiting on a fixed set of goroutines.
- `errgroup.WithContext` for waiting plus error propagation and cancellation.
- Result channels or done channels when you want to fan-in results.
- Worker pools with buffered channels for bounded concurrency.

**Choice**
I prefer `errgroup.WithContext` for request-scoped work because it gives me:
- a single `Wait()` that returns the first error
- automatic cancellation of the other goroutines
- a shared context to pass into work functions

**Gotchas**
- Remember to check `ctx.Done()` inside workers.
- Protect shared state (map writes) with a mutex.
- Don’t leak goroutines by forgetting to return on cancel.

**Error handling**
- With `errgroup`, return errors from each worker and handle `g.Wait()` once.
- With `WaitGroup`, you need separate error aggregation (channel or shared struct).

**Project example**
- We use `errgroup.WithContext` in `internal/handler/handler.go` to run payment,
  vendor, and courier steps concurrently and to cancel on first failure.

---

## Go Question 2: Canceling background tasks

**Scenario**
- Orders have a 2s deadline. If one step fails, the rest should stop.

**Signal**
- We use `context.WithTimeout` and `errgroup.WithContext`. When a goroutine returns
  an error, the derived context is canceled.

**Verification**
- Each step uses `shared.SleepOrDone` and `CourierPool.Acquire`, which honor
  `ctx.Done()`. Tests check `Tracker.Running()` returns to zero after completion.

---

## Go Question 3: Error differentiation

**Approaches**
- Sentinel errors + `errors.Is`
- Custom error types + `errors.As`
- Error wrapping with `%w` for context

**Preference**
- Sentinel errors + `errors.Is` for simplicity and consistent mapping.

**Project example**
- We wrap step failures and map them via `apperr.Kind/HTTPStatus`
  (e.g., `ErrPaymentDeclined`, `ErrVendorUnavailable`, `ErrNoCourierAvailable`).

---

## Go Question 4: OS threads vs goroutines

**Key differences**
- Goroutines are lightweight (small stack, cheap to create).
- OS threads are heavier and managed by the OS.

**Scheduling**
- Go uses an M:N scheduler (many goroutines multiplexed onto fewer threads).
- Scheduling is preemptive (since Go 1.14+), so long-running goroutines can be
  interrupted by the runtime.

**Implications**
- It’s safe to spawn many goroutines, but you still need to manage shared state,
  cancellation, and backpressure.

---

## Go Question 5: Test design improvements

**What was undertested**
- Concurrency paths and failure propagation.

**Restructure**
- Added a stress test with multiple goroutines.
- Converted service tests to table-driven style.

**Edge cases added**
- Payment failure cancels other steps.
- Courier assignment timeout while pool is saturated.
- Delay overrides per step.

**Why it prevents regressions**
- Exercises concurrency, cancellation, and error mapping under load.

**How to catch data races**
- Run `go test -race ./...` regularly.
