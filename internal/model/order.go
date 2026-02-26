// Package model defines the request and response payloads for the order API.
package model

// OrderRequest is the input payload for processing an order.
type OrderRequest struct {
	OrderID  string           `json:"order_id"`
	Amount   uint64           `json:"amount"`
	FailStep string           `json:"fail_step,omitempty"` // "payment" | "vendor" | "courier"
	DelayMS  map[string]int64 `json:"delay_ms,omitempty"`  // per-step delay override in ms
}

// OrderResponse is the output payload returned after order processing.
type OrderResponse struct {
	Status  string        `json:"status"` // "ok" | "error"
	OrderID string        `json:"order_id"`
	Steps   []StepResult  `json:"steps,omitempty"`
	Error   *ErrorPayload `json:"error,omitempty"`
}

// StepResult captures the outcome of a single processing step.
type StepResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // "ok" | "error" | "canceled"
	DurationMS int64  `json:"duration_ms"`
	Detail     string `json:"detail,omitempty"`
}

// ErrorPayload describes an error in the response.
type ErrorPayload struct {
	Kind    string `json:"kind"` // "payment_declined", "timeout", etc.
	Message string `json:"message,omitempty"`
}
