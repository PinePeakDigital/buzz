package main

import (
	"testing"
	"time"
)

// TestValidateDatapointInput tests the validateDatapointInput function
func TestValidateDatapointInput(t *testing.T) {
	tests := []struct {
		name        string
		inputDate   string
		inputValue  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid input",
			inputDate:   "2024-01-15",
			inputValue:  "5",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "valid decimal value",
			inputDate:   "2024-01-15",
			inputValue:  "3.14",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "empty date",
			inputDate:   "",
			inputValue:  "5",
			expectError: true,
			errorMsg:    "Date cannot be empty",
		},
		{
			name:        "empty value",
			inputDate:   "2024-01-15",
			inputValue:  "",
			expectError: true,
			errorMsg:    "Value cannot be empty",
		},
		{
			name:        "invalid date format",
			inputDate:   "15-01-2024",
			inputValue:  "5",
			expectError: true,
			errorMsg:    "Invalid date format (use YYYY-MM-DD)",
		},
		{
			name:        "invalid value not a number",
			inputDate:   "2024-01-15",
			inputValue:  "abc",
			expectError: true,
			errorMsg:    "Value must be a valid number",
		},
		{
			name:        "date too far in future",
			inputDate:   time.Now().AddDate(0, 0, 5).Format("2006-01-02"),
			inputValue:  "5",
			expectError: true,
			errorMsg:    "Date cannot be more than 1 day in the future",
		},
		{
			name:        "date today is valid",
			inputDate:   time.Now().Format("2006-01-02"),
			inputValue:  "5",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "date tomorrow is valid",
			inputDate:   time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
			inputValue:  "5",
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateDatapointInput(tt.inputDate, tt.inputValue)
			if tt.expectError {
				if result == "" {
					t.Errorf("Expected error message '%s', got no error", tt.errorMsg)
				} else if result != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, result)
				}
			} else {
				if result != "" {
					t.Errorf("Expected no error, got '%s'", result)
				}
			}
		})
	}
}

// TestValidateCreateGoalInput tests the validateCreateGoalInput function
func TestValidateCreateGoalInput(t *testing.T) {
	tests := []struct {
		name        string
		slug        string
		title       string
		goalType    string
		gunits      string
		goaldate    string
		goalval     string
		rate        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid input with goaldate and goalval",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "10",
			rate:        "null",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "valid input with goaldate and rate",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "null",
			rate:        "1",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "valid input with goalval and rate",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "null",
			goalval:     "10",
			rate:        "1",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "empty slug",
			slug:        "",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "10",
			rate:        "null",
			expectError: true,
			errorMsg:    "Slug cannot be empty",
		},
		{
			name:        "empty title",
			slug:        "testgoal",
			title:       "",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "10",
			rate:        "null",
			expectError: true,
			errorMsg:    "Title cannot be empty",
		},
		{
			name:        "empty goal type",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "10",
			rate:        "null",
			expectError: true,
			errorMsg:    "Goal type cannot be empty",
		},
		{
			name:        "empty gunits",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "",
			goaldate:    "1234567890",
			goalval:     "10",
			rate:        "null",
			expectError: true,
			errorMsg:    "Goal units cannot be empty",
		},
		{
			name:        "all three parameters provided",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "10",
			rate:        "1",
			expectError: true,
			errorMsg:    "Exactly 2 out of 3 (goaldate, goalval, rate) must be provided",
		},
		{
			name:        "only one parameter provided",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "null",
			rate:        "null",
			expectError: true,
			errorMsg:    "Exactly 2 out of 3 (goaldate, goalval, rate) must be provided",
		},
		{
			name:        "no parameters provided",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "",
			goalval:     "",
			rate:        "",
			expectError: true,
			errorMsg:    "Exactly 2 out of 3 (goaldate, goalval, rate) must be provided",
		},
		{
			name:        "invalid goaldate - partial null",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "nu",
			goalval:     "10",
			rate:        "1",
			expectError: true,
			errorMsg:    "Goal date must be a valid epoch timestamp or 'null'",
		},
		{
			name:        "invalid goaldate - non-numeric",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "abc",
			goalval:     "10",
			rate:        "null",
			expectError: true,
			errorMsg:    "Goal date must be a valid epoch timestamp or 'null'",
		},
		{
			name:        "invalid goaldate - mixed alphanumeric",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "123abc",
			goalval:     "10",
			rate:        "null",
			expectError: true,
			errorMsg:    "Goal date must be a valid epoch timestamp or 'null'",
		},
		{
			name:        "invalid goalval - partial null",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "n",
			rate:        "1",
			expectError: true,
			errorMsg:    "Goal value must be a valid number or 'null'",
		},
		{
			name:        "invalid goalval - non-numeric",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "xyz",
			rate:        "null",
			expectError: true,
			errorMsg:    "Goal value must be a valid number or 'null'",
		},
		{
			name:        "invalid rate - partial null",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "10",
			rate:        "nul",
			expectError: true,
			errorMsg:    "Rate must be a valid number or 'null'",
		},
		{
			name:        "invalid rate - non-numeric",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "null",
			goalval:     "10",
			rate:        "abc",
			expectError: true,
			errorMsg:    "Rate must be a valid number or 'null'",
		},
		{
			name:        "valid negative goalval",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "1234567890",
			goalval:     "-10.5",
			rate:        "null",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "valid decimal rate",
			slug:        "testgoal",
			title:       "Test Goal",
			goalType:    "hustler",
			gunits:      "units",
			goaldate:    "null",
			goalval:     "100",
			rate:        "0.5",
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCreateGoalInput(tt.slug, tt.title, tt.goalType, tt.gunits,
				tt.goaldate, tt.goalval, tt.rate)
			if tt.expectError {
				if result == "" {
					t.Errorf("Expected error message '%s', got no error", tt.errorMsg)
				} else if result != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, result)
				}
			} else {
				if result != "" {
					t.Errorf("Expected no error, got '%s'", result)
				}
			}
		})
	}
}

// TestIsAlphanumericOrDash tests the isAlphanumericOrDash function
func TestIsAlphanumericOrDash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"lowercase letter", "a", true},
		{"uppercase letter", "Z", true},
		{"digit", "5", true},
		{"dash", "-", true},
		{"underscore", "_", true},
		{"space", " ", false},
		{"special char", "@", false},
		{"empty string", "", false},
		{"multiple chars", "ab", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAlphanumericOrDash(tt.input)
			if result != tt.expected {
				t.Errorf("isAlphanumericOrDash(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsLetter tests the isLetter function
func TestIsLetter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"lowercase letter", "a", true},
		{"uppercase letter", "Z", true},
		{"digit", "5", false},
		{"space", " ", false},
		{"special char", "@", false},
		{"empty string", "", false},
		{"multiple chars", "ab", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLetter(tt.input)
			if result != tt.expected {
				t.Errorf("isLetter(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsNumericOrNull tests the isNumericOrNull function
func TestIsNumericOrNull(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"digit", "5", true},
		{"n from null", "n", true},
		{"u from null", "u", true},
		{"l from null", "l", true},
		{"letter a", "a", false},
		{"space", " ", false},
		{"empty string", "", false},
		{"multiple chars", "12", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericOrNull(tt.input)
			if result != tt.expected {
				t.Errorf("isNumericOrNull(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsNumericWithDecimal tests the isNumericWithDecimal function
func TestIsNumericWithDecimal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"digit", "5", true},
		{"decimal point", ".", true},
		{"negative sign", "-", true},
		{"n from null", "n", true},
		{"u from null", "u", true},
		{"l from null", "l", true},
		{"letter a", "a", false},
		{"space", " ", false},
		{"empty string", "", false},
		{"multiple chars", "12", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericWithDecimal(tt.input)
			if result != tt.expected {
				t.Errorf("isNumericWithDecimal(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsValidInteger tests the isValidInteger function
func TestIsValidInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid positive integer", "1234567890", true},
		{"valid negative integer", "-123", true},
		{"zero", "0", true},
		{"invalid - partial null", "nu", false},
		{"invalid - null string", "null", false},
		{"invalid - empty string", "", false},
		{"invalid - letters", "abc", false},
		{"invalid - mixed alphanumeric", "123abc", false},
		{"invalid - float", "123.45", false},
		{"invalid - decimal point only", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidInteger(tt.input)
			if result != tt.expected {
				t.Errorf("isValidInteger(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsValidFloat tests the isValidFloat function
func TestIsValidFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid positive integer", "123", true},
		{"valid negative integer", "-456", true},
		{"valid positive float", "123.45", true},
		{"valid negative float", "-67.89", true},
		{"valid decimal starting with point", ".5", true},
		{"zero", "0", true},
		{"zero float", "0.0", true},
		{"scientific notation", "1e10", true},
		{"invalid - partial null", "n", false},
		{"invalid - null string", "null", false},
		{"invalid - empty string", "", false},
		{"invalid - letters", "xyz", false},
		{"invalid - mixed alphanumeric", "12.3abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidFloat(tt.input)
			if result != tt.expected {
				t.Errorf("isValidFloat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIssueEdgeCases verifies the specific edge cases mentioned in issue #84
func TestIssueEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		goaldate string
		goalval  string
		rate     string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "partial 'nu' should be rejected",
			goaldate: "nu",
			goalval:  "10",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal date must be a valid epoch timestamp or 'null'",
		},
		{
			name:     "partial 'n' should be rejected",
			goaldate: "1234567890",
			goalval:  "n",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "exact 'null' should be accepted",
			goaldate: "null",
			goalval:  "10",
			rate:     "1",
			wantErr:  false,
		},
		{
			name:     "valid epoch timestamp should be accepted",
			goaldate: "1234567890",
			goalval:  "10.5",
			rate:     "null",
			wantErr:  false,
		},
		{
			name:     "valid float should be accepted",
			goaldate: "null",
			goalval:  "-5.5",
			rate:     "0.25",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCreateGoalInput("slug", "title", "hustler", "units", tt.goaldate, tt.goalval, tt.rate)
			gotErr := result != ""
			if gotErr != tt.wantErr {
				t.Errorf("got error=%v, want error=%v; error message: %q", gotErr, tt.wantErr, result)
			}
			if tt.wantErr && result != tt.errMsg {
				t.Errorf("got error message %q, want %q", result, tt.errMsg)
			}
		})
	}
}
