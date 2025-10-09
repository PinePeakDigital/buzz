package main

import (
	"testing"
)

// TestMin tests the min function
func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a smaller", 5, 10, 5},
		{"b smaller", 10, 5, 5},
		{"equal", 7, 7, 7},
		{"negative numbers", -5, -10, -10},
		{"mixed signs", -5, 5, -5},
		{"zero and positive", 0, 5, 0},
		{"zero and negative", 0, -5, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestMax tests the max function
func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a larger", 10, 5, 10},
		{"b larger", 5, 10, 10},
		{"equal", 7, 7, 7},
		{"negative numbers", -5, -10, -5},
		{"mixed signs", -5, 5, 5},
		{"zero and positive", 0, 5, 5},
		{"zero and negative", 0, -5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := max(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestCalculateColumns tests the calculateColumns function
func TestCalculateColumns(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		expected int
	}{
		{"very narrow", 10, 1},
		{"exactly one column", 22, 1},
		{"two columns", 44, 2},
		{"three columns", 66, 3},
		{"large width", 220, 10},
		{"zero width", 0, 1},
		{"negative width", -10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateColumns(tt.width)
			if result != tt.expected {
				t.Errorf("calculateColumns(%d) = %d, want %d", tt.width, result, tt.expected)
			}
		})
	}
}

// TestTruncateString tests the truncateString function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"shorter than max", "hello", 10, "hello     "},
		{"exactly max length", "hello", 5, "hello"},
		{"longer than max", "hello world", 8, "hello..."},
		{"much longer", "this is a very long string", 10, "this is..."},
		{"empty string", "", 5, "     "},
		{"max length 3", "hello", 3, "..."},
		{"single char", "a", 5, "a    "},
		{"unicode characters", "hello🎉", 8, "hello..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestWrapText tests the wrapText function
func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		{
			"short text",
			"hello world",
			20,
			[]string{"hello world"},
		},
		{
			"text that needs wrapping",
			"hello world this is a test",
			10,
			[]string{"hello", "world this", "is a test"},
		},
		{
			"single word",
			"hello",
			10,
			[]string{"hello"},
		},
		{
			"empty text",
			"",
			10,
			[]string{""},
		},
		{
			"zero width",
			"hello world",
			0,
			[]string{"hello world"},
		},
		{
			"text with multiple spaces",
			"hello  world  test",
			10,
			[]string{"hello", "world test"},
		},
		{
			"long word exceeds width",
			"supercalifragilisticexpialidocious test",
			10,
			[]string{"supercalifragilisticexpialidocious", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			if len(result) != len(tt.expected) {
				t.Errorf("wrapText(%q, %d) returned %d lines, want %d lines", tt.text, tt.width, len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Want: %v", tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("wrapText(%q, %d) line %d = %q, want %q", tt.text, tt.width, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestFuzzyMatch tests the fuzzyMatch function
func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		text     string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"case insensitive", "Hello", "hello", true},
		{"fuzzy match in order", "hlo", "hello", true},
		{"fuzzy match with gaps", "hw", "hello world", true},
		{"pattern not in order", "olh", "hello", false},
		{"empty pattern", "", "hello", true},
		{"empty text", "hello", "", false},
		{"both empty", "", "", true},
		{"pattern longer than text", "hello world", "hello", false},
		{"partial fuzzy", "bm", "beeminder", true},
		{"no match", "xyz", "hello", false},
		{"special characters", "h.w", "hello.world", true},
		{"numbers", "123", "abc123def", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fuzzyMatch(tt.pattern, tt.text)
			if result != tt.expected {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.pattern, tt.text, result, tt.expected)
			}
		})
	}
}

// TestFormatGoalFirstLine tests the formatGoalFirstLine function
func TestFormatGoalFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		slug     string
		pledge   float64
		expected string
	}{
		{"short slug with small pledge", "test", 5.0, "test            $5"},
		{"short slug with large pledge", "test", 270.0, "test          $270"},
		{"exact length slug", "the_slug", 5.0, "the_slug        $5"},
		{"long slug needs truncation", "a_very_long_slug", 5.0, "a_very_long_... $5"},
		{"very long slug", "this_is_an_extremely_long_slug_name", 10.0, "this_is_an_... $10"},
		{"empty slug", "", 5.0, "                $5"},
		{"slug with spaces", "my goal", 15.0, "my goal        $15"},
		{"zero pledge", "test", 0.0, "test            $0"},
		{"large pledge value", "x", 10000.0, "x           $10000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGoalFirstLine(tt.slug, tt.pledge)
			if result != tt.expected {
				t.Errorf("formatGoalFirstLine(%q, %.0f) = %q, want %q", tt.slug, tt.pledge, result, tt.expected)
			}
			if len(result) != 18 {
				t.Errorf("formatGoalFirstLine(%q, %.0f) length = %d, want 18", tt.slug, tt.pledge, len(result))
			}
		})
	}
}

// TestFormatGoalSecondLine tests the formatGoalSecondLine function
func TestFormatGoalSecondLine(t *testing.T) {
	tests := []struct {
		name       string
		deltaValue string
		timeframe  string
		expected   string
	}{
		{"short values", "+2", "3 days", "+2 in 3 days      "},
		{"medium values", "+10", "5 days", "+10 in 5 days     "},
		{"exact length", "1.315464", "5 h", "1.315464 in 5 h   "},
		{"needs truncation", "1.315464", "5 days", "1.315464 in 5 days"},
		{"very long timeframe", "+5", "10 days 3 hours", "+5 in 10 days 3..."},
		{"time format", "2:30:00", "6 hrs", "2:30:00 in 6 hrs  "},
		{"negative value", "-3", "2 days", "-3 in 2 days      "},
		{"zero value", "0", "today", "0 in today        "},
		{"long delta value", "+1000000", "1 day", "+1000000 in 1 day "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGoalSecondLine(tt.deltaValue, tt.timeframe)
			if result != tt.expected {
				t.Errorf("formatGoalSecondLine(%q, %q) = %q, want %q", tt.deltaValue, tt.timeframe, result, tt.expected)
			}
			if len(result) != 18 {
				t.Errorf("formatGoalSecondLine(%q, %q) length = %d, want 18", tt.deltaValue, tt.timeframe, len(result))
			}
		})
	}
}
