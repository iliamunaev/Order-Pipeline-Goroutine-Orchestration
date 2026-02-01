// Package model defines the request and response payloads used by the API.
// It keeps transport-level types in one place for reuse.
package model

// OrderRequest is the input payload for creating an order.
type OrderRequest struct {
	OrderID  string           `json:"order_id"`
	Amount   int64            `json:"amount"`
	FailStep string           `json:"fail_step,omitempty"` // "payment" | "vendor" | "courier"
	DelayMS  map[string]int64 `json:"delay_ms,omitempty"`  // per-step delay override
}

// OrderResponse is the output payload returned by the order handler.
type OrderResponse struct {
	Status  string        `json:"status"` // "ok" | "error"
	OrderID string        `json:"order_id"`
	Steps   []StepResult  `json:"steps,omitempty"`
	Error   *ErrorPayload `json:"error,omitempty"`
}

// StepResult captures the outcome of a processing step.
type StepResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // "ok" | "error" | "canceled"
	DurationMS int64  `json:"duration_ms"`
	Detail     string `json:"detail,omitempty"` // optional, human-readable error detail
}

// ErrorPayload describes an error response.
type ErrorPayload struct {
	Kind    string `json:"kind"`              // "payment_declined", "timeout"
	Message string `json:"message,omitempty"` // optional, human-readable error message
}
