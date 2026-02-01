## Go Question 1: Goroutine orchestration

**Options**
- `sync.WaitGroup` for waiting on a fixed set of goroutines.
- `errgroup.WithContext` for waiting plus error propagation and cancellation.
- Result channels or done channels when you want to fan-in results.
- Worker pools with buffered channels for bounded concurrency.

**Choice**
I use `errgroup.WithContext` in the handler because it gives me:
- one `Wait()` that returns the first error
- automatic cancellation of sibling goroutines
- a shared context to pass into step functions

**Gotchas**
- Workers must check `ctx.Done()` and return promptly.
- Shared data (results map) needs a mutex.
- Avoid goroutine leaks by always returning on cancel.

**Error handling**
- With `errgroup`, return errors from each worker and handle `g.Wait()` once.
- With `WaitGroup`, add a separate error channel/aggregation.

**Project example**
- `internal/handler/handler.go` runs payment, vendor, and courier concurrently
  and cancels the rest on first failure.

---

## Go Question 2: Canceling background tasks

**Scenario**
- Orders have a 2s deadline. If any step fails, the rest should stop.

**Signal**
- `context.WithTimeout` + `errgroup.WithContext` cancels the derived context
  when a goroutine returns an error.

**Verification**
- `shared.SleepOrDone` and pool `Acquire` return early on `ctx.Done()`.
- `Tracker.Running()` is checked in tests to ensure goroutines exit.

---

## Go Question 3: Error differentiation

**Approaches**
- Sentinel errors + `errors.Is`
- Custom error types + `errors.As`
- Error wrapping with `%w` for context

**Preference**
- Sentinel errors + `errors.Is` for simple mapping.

**Project example**
- `apperr.Kind/HTTPStatus` map `ErrPaymentDeclined`,
  `ErrVendorUnavailable`, and `ErrNoCourierAvailable` to client responses.

---

## Go Question 4: OS threads vs goroutines

**Key differences**
- Goroutines are lightweight (small stack, cheap to spawn).
- OS threads are heavier and managed by the OS.

**Scheduling**
- Go uses an M:N scheduler.
- Preemptive since Go 1.14+, so long-running goroutines get interrupted.

**Implications**
- You can spawn many goroutines, but must manage shared state, cancellation,
  and backpressure.

---

## Go Question 5: Test design improvements

**What was undertested**
- Concurrency paths and failure propagation.

**Restructure**
- Added handler validation tests and a stress test.
- Kept service tests table-driven for payment, vendor, and courier.
- Added direct unit tests for pool and tracker (including concurrency).

**Edge cases added**
- Payment failure cancels other steps.
- Pool acquisition timeout when saturated.
- Invalid JSON and missing fields at the handler level.

**Why it prevents regressions**
- Exercises concurrency, cancellation, and error mapping under load.

**How to catch data races**
- Run `go test -race ./...` regularly.
