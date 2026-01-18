package main

import (
	"testing"
)

// TestFormatListRate tests the formatListRate function
func TestFormatListRate(t *testing.T) {
	tests := []struct {
		name     string
		rate     *float64
		runits   string
		expected string
	}{
		{
			name:     "nil rate",
			rate:     nil,
			runits:   "d",
			expected: "-",
		},
		{
			name:     "zero rate",
			rate:     float64Ptr(0.0),
			runits:   "d",
			expected: "0/d",
		},
		{
			name:     "integer rate",
			rate:     float64Ptr(1.0),
			runits:   "d",
			expected: "1/d",
		},
		{
			name:     "decimal rate",
			rate:     float64Ptr(0.5),
			runits:   "w",
			expected: "0.5/w",
		},
		{
			name:     "decimal rate with multiple digits",
			rate:     float64Ptr(2.75),
			runits:   "d",
			expected: "2.75/d",
		},
		{
			name:     "large integer rate",
			rate:     float64Ptr(100.0),
			runits:   "m",
			expected: "100/m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatListRate(tt.rate, tt.runits)
			if result != tt.expected {
				t.Errorf("formatListRate(%v, %q) = %q, want %q", tt.rate, tt.runits, result, tt.expected)
			}
		})
	}
}

// float64Ptr is a helper function to create a pointer to a float64
func float64Ptr(f float64) *float64 {
	return &f
}
