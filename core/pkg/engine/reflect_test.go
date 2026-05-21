package engine

import (
	"testing"
)

// TestCheckMaxSteps tests max steps guard.
func TestCheckMaxSteps(t *testing.T) {
	tests := []struct {
		step     int
		maxSteps int
		expected bool
	}{
		{1, 25, false},
		{25, 25, false},
		{26, 25, true},
		{100, 25, true},
		{5, 0, false}, // maxSteps=0 means unlimited
		{100, 0, false},
		{1, -1, false}, // negative maxSteps means unlimited
	}

	for _, tt := range tests {
		result := checkMaxSteps(tt.step, tt.maxSteps)
		if result != tt.expected {
			t.Errorf("checkMaxSteps(%d, %d) = %v; want %v", tt.step, tt.maxSteps, result, tt.expected)
		}
	}
}
