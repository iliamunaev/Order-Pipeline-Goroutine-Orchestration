package shared

import (
	"testing"
	"time"
)

func TestDelayForStep(t *testing.T) {
	t.Parallel()

	defaultDelay := 10 * time.Millisecond

	tests := []struct {
		name     string
		delayMS  map[string]int64
		step     string
		expected time.Duration
	}{
		{name: "nil_map", delayMS: nil, step: "payment", expected: defaultDelay},
		{name: "override", delayMS: map[string]int64{"payment": 3}, step: "payment", expected: 3 * time.Millisecond},
		{name: "zero_override", delayMS: map[string]int64{"payment": 0}, step: "payment", expected: defaultDelay},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DelayForStep(tt.delayMS, tt.step, defaultDelay); got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
