package main

import (
	"fmt"
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
		{"exactly one column", 20, 1},
		{"two columns", 40, 2},
		{"three columns", 60, 3},
		{"large width", 200, 10},
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
		{"short slug with small pledge", "test", 5.0, "test          $5"},
		{"short slug with large pledge", "test", 270.0, "test        $270"},
		{"exact length slug", "the_slug", 5.0, "the_slug      $5"},
		{"long slug needs truncation", "a_very_long_slug", 5.0, "a_very_lon... $5"},
		{"very long slug", "this_is_an_extremely_long_slug_name", 10.0, "this_is_a... $10"},
		{"empty slug", "", 5.0, "              $5"},
		{"slug with spaces", "my goal", 15.0, "my goal      $15"},
		{"zero pledge", "test", 0.0, "test          $0"},
		{"large pledge value", "x", 10000.0, "x         $10000"},
		{"extremely large pledge that exceeds width", "", 999999999999999.0, "$999999999999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGoalFirstLine(tt.slug, tt.pledge)
			if result != tt.expected {
				t.Errorf("formatGoalFirstLine(%q, %.0f) = %q, want %q", tt.slug, tt.pledge, result, tt.expected)
			}
			if len(result) != 16 {
				t.Errorf("formatGoalFirstLine(%q, %.0f) length = %d, want 16", tt.slug, tt.pledge, len(result))
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
		{"short values", "+2", "3 days", "+2 in 3 days    "},
		{"medium values", "+10", "5 days", "+10 in 5 days   "},
		{"exact length", "1.315464", "5 h", "1.315464 in 5 h "},
		{"needs truncation", "1.315464", "5 days", "1.315464 in 5..."},
		{"very long timeframe", "+5", "10 days 3 hours", "+5 in 10 days..."},
		{"time format", "2:30:00", "6 hrs", "2:30:00 in 6 hrs"},
		{"negative value", "-3", "2 days", "-3 in 2 days    "},
		{"zero value", "0", "today", "0 in today      "},
		{"long delta value", "+1000000", "1 day", "+1000000 in 1..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGoalSecondLine(tt.deltaValue, tt.timeframe)
			if result != tt.expected {
				t.Errorf("formatGoalSecondLine(%q, %q) = %q, want %q", tt.deltaValue, tt.timeframe, result, tt.expected)
			}
			if len(result) != 16 {
				t.Errorf("formatGoalSecondLine(%q, %q) length = %d, want 16", tt.deltaValue, tt.timeframe, len(result))
			}
		})
	}
}

// TestIsTimeFormat tests the isTimeFormat function
func TestIsTimeFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple time HH:MM", "1:30", true},
		{"zero-padded time", "00:05", true},
		{"time with seconds", "2:45:30", true},
		{"negative time", "-1:30", true},
		{"positive time", "+1:30", true},
		{"decimal number", "1.5", false},
		{"integer", "5", false},
		{"negative integer", "-5", false},
		{"decimal with plus", "+2.5", false},
		{"zero", "0", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeFormat(tt.input)
			if result != tt.expected {
				t.Errorf("isTimeFormat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestTimeToDecimalHours tests the timeToDecimalHours function
func TestTimeToDecimalHours(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  float64
		shouldOk  bool
		tolerance float64
	}{
		{"1 hour 30 minutes", "1:30", 1.5, true, 0.0001},
		{"5 minutes", "00:05", 0.083333, true, 0.0001},
		{"2 hours 45 minutes", "2:45", 2.75, true, 0.0001},
		{"3 hours exact", "3:00", 3.0, true, 0.0001},
		{"30 seconds", "00:00:30", 0.008333, true, 0.0001},
		{"1 hour 30 min 45 sec", "1:30:45", 1.5125, true, 0.0001},
		{"negative time", "-1:30", -1.5, true, 0.0001},
		{"positive time with plus", "+1:30", 1.5, true, 0.0001},
		{"negative time with seconds", "-2:15:30", -2.258333, true, 0.0001},
		{"invalid format - no colon", "130", 0, false, 0},
		{"invalid format - too many parts", "1:30:45:60", 0, false, 0},
		{"invalid format - non-numeric", "a:b", 0, false, 0},
		{"invalid format - empty", "", 0, false, 0},
		{"zero time", "0:00", 0, true, 0.0001},
		{"invalid minutes - too high", "1:60", 0, false, 0},
		{"invalid seconds - too high", "1:30:60", 0, false, 0},
		{"negative minutes", "1:-30", 0, false, 0},
		{"decimal minutes", "1:30.5", 0, false, 0},
		{"large hours", "100:30", 100.5, true, 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := timeToDecimalHours(tt.input)
			if ok != tt.shouldOk {
				t.Errorf("timeToDecimalHours(%q) ok = %v, want %v", tt.input, ok, tt.shouldOk)
			}
			if tt.shouldOk {
				// Check if result is within tolerance
				diff := result - tt.expected
				if diff < 0 {
					diff = -diff
				}
				if diff > tt.tolerance {
					t.Errorf("timeToDecimalHours(%q) = %f, want %f (within %f)", tt.input, result, tt.expected, tt.tolerance)
				}
			}
		})
	}
}

// TestReadValueFromStdin tests the readValueFromStdin function
// Note: This test is limited because we can't easily mock os.Stdin in unit tests
// The actual stdin piping behavior is tested via integration tests
func TestReadValueFromStdinTerminal(t *testing.T) {
	// When running tests from a terminal, stdin is typically a character device
	// so readValueFromStdin should return an error
	_, err := readValueFromStdin()
	// In a test environment (terminal), stdin should not be a pipe
	// so we expect an error
	if err == nil {
		// If no error, stdin might be piped (e.g., in CI environment)
		// This is acceptable behavior, just log it
		t.Log("readValueFromStdin succeeded - stdin appears to be piped (expected in CI)")
	} else {
		// Expected case: stdin is a terminal, should return error
		if err.Error() != "stdin is not piped" && err.Error() != "no input from stdin" {
			t.Errorf("readValueFromStdin() returned unexpected error: %v", err)
		}
	}
}

// TestDetectMisplacedFlag tests the detectMisplacedFlag function
func TestDetectMisplacedFlag(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedFlag string
		description  string
	}{
		{
			name:         "flag --requestid after args",
			args:         []string{"goalslug", "1", "comment", "--requestid"},
			expectedFlag: "--requestid",
			description:  "Should detect --requestid flag after positional arguments",
		},
		{
			name:         "flag --requestid= after args",
			args:         []string{"goalslug", "1", "comment", "--requestid=123"},
			expectedFlag: "--requestid=123",
			description:  "Should detect --requestid=value flag after positional arguments",
		},
		{
			name:         "negative number",
			args:         []string{"goalslug", "-5.5", "comment"},
			expectedFlag: "",
			description:  "Should not detect negative numbers as flags",
		},
		{
			name:         "decorative dashes",
			args:         []string{"goalslug", "1", "--decorative--"},
			expectedFlag: "",
			description:  "Should not detect decorative dashes as flags",
		},
		{
			name:         "double dash alone",
			args:         []string{"goalslug", "1", "--"},
			expectedFlag: "",
			description:  "Should not detect standalone double dash as flag",
		},
		{
			name:         "comment with username-like pattern",
			args:         []string{"goalslug", "1", "--username123"},
			expectedFlag: "",
			description:  "Should not detect comment text that looks flag-like",
		},
		{
			name:         "no flags",
			args:         []string{"goalslug", "1", "normal", "comment"},
			expectedFlag: "",
			description:  "Should not detect any flags in normal comments",
		},
		{
			name:         "requestid in middle of comment",
			args:         []string{"goalslug", "1", "comment", "with", "--requestid", "in", "middle"},
			expectedFlag: "--requestid",
			description:  "Should detect --requestid even in middle of multi-word comment",
		},
		{
			name:         "multiple flags returns first",
			args:         []string{"--requestid=123", "--requestid=456"},
			expectedFlag: "--requestid=123",
			description:  "Should return the first detected flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectMisplacedFlag(tt.args)

			if result != tt.expectedFlag {
				t.Errorf("%s: detectMisplacedFlag() = %q, want %q", tt.description, result, tt.expectedFlag)
			}
		})
	}
}

// TestRedactAuthToken tests the redactAuthToken function
func TestRedactAuthToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"query parameter with ?",
			"https://example.com/api?auth_token=secret123",
			"https://example.com/api?auth_token=***",
		},
		{
			"query parameter with &",
			"https://example.com/api?user=alice&auth_token=secret123&other=value",
			"https://example.com/api?user=alice&auth_token=***&other=value",
		},
		{
			"multiple occurrences",
			"url1?auth_token=abc123 and url2?auth_token=xyz789",
			"url1?auth_token=*** and url2?auth_token=***",
		},
		{
			"form data",
			"auth_token=secret123&username=alice",
			"auth_token=***&username=alice",
		},
		{
			"no auth token",
			"https://example.com/api?user=alice",
			"https://example.com/api?user=alice",
		},
		{
			"auth_token at end of URL",
			"https://example.com/api?user=alice&auth_token=secret123",
			"https://example.com/api?user=alice&auth_token=***",
		},
		{
			"auth_token with special characters",
			"https://example.com/api?auth_token=abc-123_xyz.789",
			"https://example.com/api?auth_token=***",
		},
		{
			"empty string",
			"",
			"",
		},
		{
			"URL with no query parameters",
			"https://example.com/api",
			"https://example.com/api",
		},
		{
			"auth_token in URL path (should not match)",
			"https://example.com/auth_token/endpoint",
			"https://example.com/auth_token/endpoint",
		},
		{
			"error message with URL",
			"failed to fetch: GET https://api.beeminder.com/api/v1/users/alice/goals.json?auth_token=abc123",
			"failed to fetch: GET https://api.beeminder.com/api/v1/users/alice/goals.json?auth_token=***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactAuthToken(tt.input)
			if result != tt.expected {
				t.Errorf("redactAuthToken(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestRedactError tests the redactError function
func TestRedactError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			"nil error",
			nil,
			"",
		},
		{
			"error without auth token",
			fmt.Errorf("failed to connect"),
			"failed to connect",
		},
		{
			"error with auth token in URL",
			fmt.Errorf("Get \"https://example.com/api?auth_token=secret123\": connection failed"),
			"Get \"https://example.com/api?auth_token=***\": connection failed",
		},
		{
			"wrapped error with auth token",
			fmt.Errorf("failed to fetch: %w", fmt.Errorf("Get \"https://api.com/v1/users/alice/goals.json?auth_token=abc123\": dial tcp: timeout")),
			"failed to fetch: Get \"https://api.com/v1/users/alice/goals.json?auth_token=***\": dial tcp: timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactError(tt.err)
			if result != tt.expected {
				t.Errorf("redactError(%v) = %q, want %q", tt.err, result, tt.expected)
			}
		})
	}
}
