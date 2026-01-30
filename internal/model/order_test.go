package model

import (
	"encoding/json"
	"testing"
)

func TestOrderRequestJSONTags(t *testing.T) {
	t.Parallel()

	req := OrderRequest{
		OrderID:  "o-1",
		Amount:   1200,
		FailStep: "vendor",
		DelayMS: map[string]int64{
			"payment": 10,
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if raw["order_id"] != "o-1" {
		t.Fatalf("expected order_id, got %v", raw["order_id"])
	}
	if raw["amount"] != float64(1200) {
		t.Fatalf("expected amount, got %v", raw["amount"])
	}
	if raw["fail_step"] != "vendor" {
		t.Fatalf("expected fail_step, got %v", raw["fail_step"])
	}

	delayRaw, ok := raw["delay_ms"].(map[string]any)
	if !ok {
		t.Fatalf("expected delay_ms map, got %T", raw["delay_ms"])
	}
	if delayRaw["payment"] != float64(10) {
		t.Fatalf("expected delay_ms.payment=10, got %v", delayRaw["payment"])
	}
}

func TestOrderResponseOmitEmptyFields(t *testing.T) {
	t.Parallel()

	resp := OrderResponse{
		Status:  "ok",
		OrderID: "o-2",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if _, ok := raw["steps"]; ok {
		t.Fatalf("expected steps to be omitted")
	}
	if _, ok := raw["error"]; ok {
		t.Fatalf("expected error to be omitted")
	}
}
