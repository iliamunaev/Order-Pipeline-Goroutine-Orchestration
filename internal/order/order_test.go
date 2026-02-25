package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iliamunaev/Order-Pipeline-Goroutine-Orchestration/internal/model"
)

type testKindErr struct {
	kind string
}

func (e testKindErr) Error() string { return e.kind }
func (e testKindErr) Kind() string  { return e.kind }

func TestNew_EmptyStepsPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty steps")
		}
	}()
	New(nil)
}

func TestProcess_AllSuccess(t *testing.T) {
	t.Parallel()

	steps := []Step{
		{Name: "a", Run: func(context.Context, model.OrderRequest) error { return nil }},
		{Name: "b", Run: func(context.Context, model.OrderRequest) error { return nil }},
		{Name: "c", Run: func(context.Context, model.OrderRequest) error { return nil }},
	}
	svc := New(steps)

	results, err := svc.Process(context.Background(), model.OrderRequest{OrderID: "o-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != "ok" {
			t.Errorf("step %s: expected status ok, got %q", r.Name, r.Status)
		}
	}
}

// Errgroup cancels the shared context when one step returns an error.
// Sibling steps must observe ctx.Done() and report "canceled".
func TestProcess_DomainErrorCancelsSiblings(t *testing.T) {
	t.Parallel()

	domainErr := testKindErr{kind: "payment_declined"}

	steps := []Step{
		{Name: "fast_fail", Run: func(context.Context, model.OrderRequest) error {
			return domainErr
		}},
		{Name: "slow", Run: func(ctx context.Context, _ model.OrderRequest) error {
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}},
	}
	svc := New(steps)

	results, err := svc.Process(context.Background(), model.OrderRequest{OrderID: "o-2"})
	if !errors.Is(err, domainErr) {
		t.Fatalf("expected domain error, got %v", err)
	}

	if results[0].Status != "error" || results[0].Detail != "payment_declined" {
		t.Fatalf("expected fast_fail: error/payment_declined, got %+v", results[0])
	}
	if results[1].Status != "canceled" {
		t.Fatalf("expected slow: canceled, got %+v", results[1])
	}
}

func TestProcess_ContextAlreadyCanceled(t *testing.T) {
	t.Parallel()

	steps := []Step{
		{Name: "a", Run: func(ctx context.Context, _ model.OrderRequest) error {
			return ctx.Err()
		}},
		{Name: "b", Run: func(ctx context.Context, _ model.OrderRequest) error {
			return ctx.Err()
		}},
	}
	svc := New(steps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results, err := svc.Process(ctx, model.OrderRequest{OrderID: "o-3"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	for _, r := range results {
		if r.Status != "canceled" {
			t.Errorf("step %s: expected canceled, got %q", r.Name, r.Status)
		}
	}
}

func TestProcess_DeadlineExceeded(t *testing.T) {
	t.Parallel()

	steps := []Step{
		{Name: "slow", Run: func(ctx context.Context, _ model.OrderRequest) error {
			<-ctx.Done()
			return ctx.Err()
		}},
	}
	svc := New(steps)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	results, err := svc.Process(ctx, model.OrderRequest{OrderID: "o-4"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if results[0].Status != "canceled" {
		t.Fatalf("expected canceled, got %q", results[0].Status)
	}
}

func TestProcess_ErrorWithoutKind(t *testing.T) {
	t.Parallel()

	plainErr := errors.New("boom")
	steps := []Step{
		{Name: "fail", Run: func(context.Context, model.OrderRequest) error {
			return plainErr
		}},
	}
	svc := New(steps)

	results, err := svc.Process(context.Background(), model.OrderRequest{OrderID: "o-5"})
	if !errors.Is(err, plainErr) {
		t.Fatalf("expected plainErr, got %v", err)
	}
	if results[0].Status != "error" {
		t.Fatalf("expected error, got %q", results[0].Status)
	}
	if results[0].Detail != "" {
		t.Fatalf("expected empty detail, got %q", results[0].Detail)
	}
}

// Results are indexed by registration order, not completion order.
// "slow" finishes after "fast" but must still appear at index 0.
func TestProcess_ResultOrder(t *testing.T) {
	t.Parallel()

	steps := []Step{
		{Name: "slow", Run: func(ctx context.Context, _ model.OrderRequest) error {
			select {
			case <-time.After(20 * time.Millisecond):
			case <-ctx.Done():
			}
			return nil
		}},
		{Name: "fast", Run: func(context.Context, model.OrderRequest) error { return nil }},
	}
	svc := New(steps)

	results, err := svc.Process(context.Background(), model.OrderRequest{OrderID: "o-6"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Name != "slow" || results[1].Name != "fast" {
		t.Fatalf("expected [slow, fast], got [%s, %s]", results[0].Name, results[1].Name)
	}
}
