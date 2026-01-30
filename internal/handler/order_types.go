package handler

type OrderRequest struct {
	OrderID  string           `json:"order_id"`
	Amount   int64            `json:"amount"`
	FailStep string           `json:"fail_step,omitempty"` // "payment" | "vendor" | "courier"
	DelayMS  map[string]int64 `json:"delay_ms,omitempty"`  // per-step delay override
}

type OrderResponse struct {
	Status  string        `json:"status"` // "ok" | "error"
	OrderID string        `json:"order_id"`
	Steps   []StepResult  `json:"steps,omitempty"`
	Error   *ErrorPayload `json:"error,omitempty"`
}

type StepResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // "ok" | "error" | "canceled"
	DurationMS int64  `json:"duration_ms"`
	Detail     string `json:"detail,omitempty"` // optional
}

type ErrorPayload struct {
	Kind    string `json:"kind"`              // "payment_declined", "timeout"
	Message string `json:"message,omitempty"` // human-readable
}
