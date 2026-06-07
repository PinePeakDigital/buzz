package main

import (
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// mockKeyMsg creates a mock KeyMsg for testing
func mockKeyMsg(runes []rune) tea.KeyMsg {
	return tea.KeyMsg{
		Runes: runes,
	}
}

// mustModel asserts that a tea.Model is the concrete model type, failing the
// test (rather than panicking) if not.
func mustModel(t *testing.T, tm tea.Model) model {
	t.Helper()
	m, ok := tm.(model)
	if !ok {
		t.Fatalf("expected model, got %T", tm)
	}
	return m
}

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
			name:        "NaN value rejected",
			inputDate:   "2024-01-15",
			inputValue:  "NaN",
			expectError: true,
			errorMsg:    "Value must be a valid number",
		},
		{
			name:        "Inf value rejected",
			inputDate:   "2024-01-15",
			inputValue:  "Inf",
			expectError: true,
			errorMsg:    "Value must be a valid number",
		},
		{
			name:        "+Inf value rejected",
			inputDate:   "2024-01-15",
			inputValue:  "+Inf",
			expectError: true,
			errorMsg:    "Value must be a valid number",
		},
		{
			name:        "-Inf value rejected",
			inputDate:   "2024-01-15",
			inputValue:  "-Inf",
			expectError: true,
			errorMsg:    "Value must be a valid number",
		},
		{
			name:        "Infinity value rejected",
			inputDate:   "2024-01-15",
			inputValue:  "Infinity",
			expectError: true,
			errorMsg:    "Value must be a valid number",
		},
		{
			name:        "+Infinity value rejected",
			inputDate:   "2024-01-15",
			inputValue:  "+Infinity",
			expectError: true,
			errorMsg:    "Value must be a valid number",
		},
		{
			name:        "-Infinity value rejected",
			inputDate:   "2024-01-15",
			inputValue:  "-Infinity",
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
		name         string
		input        string
		currentValue string
		expected     bool
	}{
		// Numeric inputs
		{"digit", "5", "", true},
		{"digit after digit", "3", "12", true},

		// Valid null prefixes
		{"n from null on empty", "n", "", true},
		{"u after n", "u", "n", true},
		{"first l after nu", "l", "nu", true},
		{"second l after nul", "l", "nul", true},

		// Invalid null sequences
		{"u without n", "u", "", false},
		{"l without nu", "l", "", false},
		{"l after n only", "l", "n", false},
		{"n after n", "n", "n", false},
		{"u after nu", "u", "nu", false},
		{"extra char after null", "x", "null", false},

		// Invalid arbitrary combinations
		{"l without context", "l", "12", false},
		{"u in middle of number", "u", "12", false},
		{"n in middle of number", "n", "12", false},

		// Other invalid inputs
		{"letter a", "a", "", false},
		{"space", " ", "", false},
		{"empty string", "", "", false},
		{"multiple chars", "12", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericOrNull(tt.input, tt.currentValue)
			if result != tt.expected {
				t.Errorf("isNumericOrNull(%q, %q) = %v, want %v", tt.input, tt.currentValue, result, tt.expected)
			}
		})
	}
}

// TestIsNumericWithDecimal tests the isNumericWithDecimal function
func TestIsNumericWithDecimal(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		currentValue string
		expected     bool
	}{
		// Numeric inputs
		{"digit", "5", "", true},
		{"digit after digit", "3", "12", true},
		{"decimal point", ".", "", true},
		{"decimal after digit", ".", "5", true},
		{"negative sign", "-", "", true},
		{"negative at start", "-", "", true},

		// Valid null prefixes
		{"n from null on empty", "n", "", true},
		{"u after n", "u", "n", true},
		{"first l after nu", "l", "nu", true},
		{"second l after nul", "l", "nul", true},

		// Invalid null sequences
		{"u without n", "u", "", false},
		{"l without nu", "l", "", false},
		{"l after n only", "l", "n", false},
		{"n after n", "n", "n", false},
		{"u after nu", "u", "nu", false},
		{"extra char after null", "x", "null", false},

		// Invalid arbitrary combinations
		{"l without context", "l", "12", false},
		{"u in middle of number", "u", "12.5", false},
		{"n in middle of number", "n", "-3.14", false},

		// Other invalid inputs
		{"letter a", "a", "", false},
		{"space", " ", "", false},
		{"empty string", "", "", false},
		{"multiple chars", "12", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericWithDecimal(tt.input, tt.currentValue)
			if result != tt.expected {
				t.Errorf("isNumericWithDecimal(%q, %q) = %v, want %v", tt.input, tt.currentValue, result, tt.expected)
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
		{"invalid - NaN", "NaN", false},
		{"invalid - Inf", "Inf", false},
		{"invalid - +Inf", "+Inf", false},
		{"invalid - -Inf", "-Inf", false},
		{"invalid - Infinity", "Infinity", false},
		{"invalid - +Infinity", "+Infinity", false},
		{"invalid - -Infinity", "-Infinity", false},
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
		// Non-finite values must be rejected end-to-end for both goalval and
		// rate, preserving each field's specific error message. ParseFloat
		// accepts NaN, Inf, +Inf, -Inf, Infinity, +Infinity, and -Infinity, so
		// cover them all for each field.
		{
			name:     "NaN goalval should be rejected",
			goaldate: "null",
			goalval:  "NaN",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "Inf goalval should be rejected",
			goaldate: "null",
			goalval:  "Inf",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "+Inf goalval should be rejected",
			goaldate: "null",
			goalval:  "+Inf",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "-Inf goalval should be rejected",
			goaldate: "null",
			goalval:  "-Inf",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "Infinity goalval should be rejected",
			goaldate: "null",
			goalval:  "Infinity",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "+Infinity goalval should be rejected",
			goaldate: "null",
			goalval:  "+Infinity",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "-Infinity goalval should be rejected",
			goaldate: "null",
			goalval:  "-Infinity",
			rate:     "1",
			wantErr:  true,
			errMsg:   "Goal value must be a valid number or 'null'",
		},
		{
			name:     "NaN rate should be rejected",
			goaldate: "null",
			goalval:  "10",
			rate:     "NaN",
			wantErr:  true,
			errMsg:   "Rate must be a valid number or 'null'",
		},
		{
			name:     "Inf rate should be rejected",
			goaldate: "null",
			goalval:  "10",
			rate:     "Inf",
			wantErr:  true,
			errMsg:   "Rate must be a valid number or 'null'",
		},
		{
			name:     "+Inf rate should be rejected",
			goaldate: "null",
			goalval:  "10",
			rate:     "+Inf",
			wantErr:  true,
			errMsg:   "Rate must be a valid number or 'null'",
		},
		{
			name:     "-Inf rate should be rejected",
			goaldate: "null",
			goalval:  "10",
			rate:     "-Inf",
			wantErr:  true,
			errMsg:   "Rate must be a valid number or 'null'",
		},
		{
			name:     "Infinity rate should be rejected",
			goaldate: "null",
			goalval:  "10",
			rate:     "Infinity",
			wantErr:  true,
			errMsg:   "Rate must be a valid number or 'null'",
		},
		{
			name:     "+Infinity rate should be rejected",
			goaldate: "null",
			goalval:  "10",
			rate:     "+Infinity",
			wantErr:  true,
			errMsg:   "Rate must be a valid number or 'null'",
		},
		{
			name:     "-Infinity rate should be rejected",
			goaldate: "null",
			goalval:  "10",
			rate:     "-Infinity",
			wantErr:  true,
			errMsg:   "Rate must be a valid number or 'null'",
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

// TestHandleNumericDecimalInput was removed: the handleNumericDecimalInput
// helper was folded into form.handleRune via the filterDecimalOrNull field
// filter. The underlying predicate is covered by TestIsNumericWithDecimal, and
// the form path by form_test.go.

// TestHandleBackspaceSearchTrimsWholeRune verifies search-mode backspace removes
// an entire multibyte rune, leaving the query as valid UTF-8.
func TestHandleBackspaceSearchTrimsWholeRune(t *testing.T) {
	m := model{
		appModel: appModel{
			searchActive: true,
			searchQuery:  "a中😀",
		},
	}

	updated, _ := handleBackspace(m)
	got := mustModel(t, updated).appModel.searchQuery
	if got != "a中" {
		t.Errorf("after backspace, searchQuery = %q, want %q", got, "a中")
	}
	if !utf8.ValidString(got) {
		t.Errorf("searchQuery is not valid UTF-8 after backspace: %q", got)
	}
}

// TestHandleTabKeyCreateGoal verifies tab/shift+tab cycle focus through the
// create-goal form when its modal is open.
func TestHandleTabKeyCreateGoal(t *testing.T) {
	m := model{appModel: appModel{mode: modeCreateGoal, createGoal: newCreateGoalForm()}}

	updated, _ := handleTabKey(m, false)
	if got := mustModel(t, updated).appModel.createGoal.focus; got != 1 {
		t.Errorf("after tab, createGoal.focus = %d, want 1", got)
	}

	updated, _ = handleTabKey(mustModel(t, updated), true)
	if got := mustModel(t, updated).appModel.createGoal.focus; got != 0 {
		t.Errorf("after shift+tab, createGoal.focus = %d, want 0", got)
	}

	// Tab is a no-op while a goal creation is in flight.
	busy := model{appModel: appModel{mode: modeCreateGoal, createGoal: newCreateGoalForm()}}
	busy.appModel.createGoal.creating = true
	updated, _ = handleTabKey(busy, false)
	if got := mustModel(t, updated).appModel.createGoal.focus; got != 0 {
		t.Errorf("tab while creating should not move focus, got %d", got)
	}
}

// TestHandleTabKeyDatapoint verifies tab/shift+tab cycle focus through the
// datapoint form when in input mode.
func TestHandleTabKeyDatapoint(t *testing.T) {
	m := model{appModel: appModel{mode: modeDatapointInput, datapoint: newDatapointForm("1")}}

	updated, _ := handleTabKey(m, false)
	if got := mustModel(t, updated).appModel.datapoint.focus; got != 1 {
		t.Errorf("after tab, datapoint.focus = %d, want 1", got)
	}

	// Shift+tab from focus 0 wraps to the last field (index 2).
	updated, _ = handleTabKey(model{appModel: appModel{mode: modeDatapointInput, datapoint: newDatapointForm("1")}}, true)
	if got := mustModel(t, updated).appModel.datapoint.focus; got != 2 {
		t.Errorf("after shift+tab wrap, datapoint.focus = %d, want 2", got)
	}
}

// TestHandleBackspaceCreateGoal verifies backspace trims the focused create-goal
// field (rune-aware) when the modal is open.
func TestHandleBackspaceCreateGoal(t *testing.T) {
	cg := newCreateGoalForm()
	cg.focus = cgTitle
	cg.fields[cgTitle].value = "Hi中"
	m := model{appModel: appModel{mode: modeCreateGoal, createGoal: cg}}

	updated, _ := handleBackspace(m)
	am := mustModel(t, updated).appModel
	if got := am.createGoal.title(); got != "Hi" {
		t.Errorf("after backspace, title() = %q, want %q", got, "Hi")
	}
}

// TestHandleBackspaceDatapoint verifies backspace trims the focused datapoint
// field when in input mode.
func TestHandleBackspaceDatapoint(t *testing.T) {
	dp := newDatapointForm("1")
	dp.focus = dpComment
	dp.fields[dpComment].value = "note😀"
	m := model{appModel: appModel{mode: modeDatapointInput, datapoint: dp}}

	updated, _ := handleBackspace(m)
	am := mustModel(t, updated).appModel
	if got := am.datapoint.comment(); got != "note" {
		t.Errorf("after backspace, comment() = %q, want %q", got, "note")
	}
}

// TestHandleSearchInputUnicode tests Unicode support in search mode
func TestHandleSearchInputUnicode(t *testing.T) {
	tests := []struct {
		name     string
		runes    []rune
		expected bool
	}{
		{"ASCII character", []rune{'a'}, true},
		{"accented character", []rune{'é'}, true},
		{"Chinese character", []rune{'中'}, true},
		{"emoji", []rune{'😀'}, true},
		{"Greek character", []rune{'α'}, true},
		{"Cyrillic character", []rune{'Ж'}, true},
		{"Hebrew character", []rune{'א'}, true},
		{"Arabic character", []rune{'ع'}, true},
		{"space", []rune{' '}, true},
		{"multiple runes", []rune{'a', 'b'}, false},
		{"empty runes", []rune{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock model in search mode
			m := model{
				appModel: appModel{
					searchActive: true,
					mode:         modeBrowse,
				},
			}

			// Create a mock KeyMsg
			msg := mockKeyMsg(tt.runes)

			// Test the handler
			updatedModel, handled := handleSearchInput(m, msg)

			if handled != tt.expected {
				t.Errorf("handleSearchInput with runes %v: handled = %v, want %v",
					tt.runes, handled, tt.expected)
			}

			// If expected to handle, check that the search query was updated
			if tt.expected && handled {
				expectedQuery := string(tt.runes)
				if updatedModel.appModel.searchQuery != expectedQuery {
					t.Errorf("searchQuery = %q, want %q", updatedModel.appModel.searchQuery, expectedQuery)
				}
			}
		})
	}
}

// TestHandleCreateModalInputUnicode tests Unicode support in create goal modal
func TestHandleCreateModalInputUnicode(t *testing.T) {
	tests := []struct {
		name       string
		focus      int
		runes      []rune
		expected   bool
		checkField func(appModel) string
		fieldName  string
	}{
		{"Title with ASCII", cgTitle, []rune{'a'}, true, func(a appModel) string { return a.createGoal.title() }, "Title"},
		{"Title with accented char", cgTitle, []rune{'é'}, true, func(a appModel) string { return a.createGoal.title() }, "Title"},
		{"Title with Chinese", cgTitle, []rune{'中'}, true, func(a appModel) string { return a.createGoal.title() }, "Title"},
		{"Title with emoji", cgTitle, []rune{'😀'}, true, func(a appModel) string { return a.createGoal.title() }, "Title"},
		{"Title with Greek", cgTitle, []rune{'Ω'}, true, func(a appModel) string { return a.createGoal.title() }, "Title"},
		{"Gunits with ASCII", cgGunits, []rune{'x'}, true, func(a appModel) string { return a.createGoal.gunits() }, "Gunits"},
		{"Gunits with accented char", cgGunits, []rune{'ñ'}, true, func(a appModel) string { return a.createGoal.gunits() }, "Gunits"},
		{"Gunits with Cyrillic", cgGunits, []rune{'Д'}, true, func(a appModel) string { return a.createGoal.gunits() }, "Gunits"},
		{"Gunits with emoji", cgGunits, []rune{'📊'}, true, func(a appModel) string { return a.createGoal.gunits() }, "Gunits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock model with create modal open. Start the title and
			// gunits fields empty so the typed character is the whole value.
			cg := newCreateGoalForm()
			cg.fields[cgTitle].value = ""
			cg.fields[cgGunits].value = ""
			cg.focus = tt.focus
			m := model{
				appModel: appModel{
					mode:       modeCreateGoal,
					createGoal: cg,
				},
			}

			// Create a mock KeyMsg
			msg := mockKeyMsg(tt.runes)

			// Test the handler
			updatedModel, handled := handleCreateModalInput(m, msg)

			if handled != tt.expected {
				t.Errorf("handleCreateModalInput for %s with runes %v: handled = %v, want %v",
					tt.fieldName, tt.runes, handled, tt.expected)
			}

			// If expected to handle, check that the field was updated
			if tt.expected && handled {
				fieldValue := tt.checkField(updatedModel.appModel)
				expectedValue := string(tt.runes)
				if fieldValue != expectedValue {
					t.Errorf("%s = %q, want %q", tt.fieldName, fieldValue, expectedValue)
				}
			}
		})
	}
}

// TestHandleDatapointInputUnicode tests Unicode support in datapoint comment field
func TestHandleDatapointInputUnicode(t *testing.T) {
	tests := []struct {
		name     string
		runes    []rune
		expected bool
	}{
		{"ASCII character", []rune{'a'}, true},
		{"accented character", []rune{'ü'}, true},
		{"Japanese character", []rune{'あ'}, true},
		{"emoji", []rune{'🎉'}, true},
		{"Arabic character", []rune{'ب'}, true},
		{"Korean character", []rune{'한'}, true},
		{"Thai character", []rune{'ก'}, true},
		{"space", []rune{' '}, true},
		{"multiple runes", []rune{'a', 'b'}, false},
		{"empty runes", []rune{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock model in input mode with comment field focused.
			// Start the comment empty so the typed character is the whole value.
			dp := newDatapointForm("1")
			dp.fields[dpComment].value = ""
			dp.focus = dpComment
			m := model{
				appModel: appModel{
					mode:      modeDatapointInput,
					datapoint: dp,
				},
			}

			// Create a mock KeyMsg
			msg := mockKeyMsg(tt.runes)

			// Test the handler
			updatedModel, handled := handleDatapointInput(m, msg)

			if handled != tt.expected {
				t.Errorf("handleDatapointInput with runes %v: handled = %v, want %v",
					tt.runes, handled, tt.expected)
			}

			// If expected to handle, check that the comment was updated
			if tt.expected && handled {
				expectedComment := string(tt.runes)
				if updatedModel.appModel.datapoint.comment() != expectedComment {
					t.Errorf("comment = %q, want %q", updatedModel.appModel.datapoint.comment(), expectedComment)
				}
			}
		})
	}
}

// TestNavigationTimeout tests the auto-disable highlight feature
func TestNavigationTimeout(t *testing.T) {
	// Create a test model with some goals
	m := model{
		appModel: appModel{
			goals: []Goal{
				{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
				{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
			},
			hasNavigated:       false,
			lastNavigationTime: time.Time{},
			width:              80,
			height:             24,
		},
	}

	t.Run("navigation sets hasNavigated and lastNavigationTime", func(t *testing.T) {
		// Navigate down
		updatedModel, cmd := handleNavigationDown(m)
		appModel := updatedModel.(model).appModel

		// Check hasNavigated is true
		if !appModel.hasNavigated {
			t.Error("hasNavigated should be true after navigation")
		}

		// Check lastNavigationTime is set
		if appModel.lastNavigationTime.IsZero() {
			t.Error("lastNavigationTime should be set after navigation")
		}

		// Check command is returned
		if cmd == nil {
			t.Error("navigationTimeoutCmd should be returned after navigation")
		}
	})

	t.Run("timeout message disables highlight after 3 seconds", func(t *testing.T) {
		// Create model with navigation that happened 4 seconds ago
		pastTime := time.Now().Add(-4 * time.Second)
		testModel := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				hasNavigated:       true,
				lastNavigationTime: pastTime,
			},
		}

		// Process navigationTimeoutMsg
		result, _ := testModel.updateApp(navigationTimeoutMsg{})
		resultAppModel := result.(model).appModel

		// hasNavigated should be false after timeout
		if resultAppModel.hasNavigated {
			t.Error("hasNavigated should be false after timeout")
		}
	})

	t.Run("timeout message does not disable if less than 3 seconds", func(t *testing.T) {
		// Create model with navigation that happened 2 seconds ago
		recentTime := time.Now().Add(-2 * time.Second)
		testModel := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				hasNavigated:       true,
				lastNavigationTime: recentTime,
			},
		}

		// Process navigationTimeoutMsg
		result, _ := testModel.updateApp(navigationTimeoutMsg{})
		resultAppModel := result.(model).appModel

		// hasNavigated should still be true
		if !resultAppModel.hasNavigated {
			t.Error("hasNavigated should still be true if less than 3 seconds elapsed")
		}
	})

	t.Run("timeout does not disable highlight while modal is open", func(t *testing.T) {
		// Create model with navigation that happened 4 seconds ago, modal open
		pastTime := time.Now().Add(-4 * time.Second)
		testModel := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				hasNavigated:       true,
				lastNavigationTime: pastTime,
				mode:               modeGoalDetail,
			},
		}

		// Process navigationTimeoutMsg
		result, _ := testModel.updateApp(navigationTimeoutMsg{})
		resultAppModel := result.(model).appModel

		// hasNavigated should still be true (modal is open)
		if !resultAppModel.hasNavigated {
			t.Error("hasNavigated should remain true when modal is open")
		}
	})

	t.Run("timeout does not disable highlight while in search mode", func(t *testing.T) {
		// Create model with navigation that happened 4 seconds ago, in search mode
		pastTime := time.Now().Add(-4 * time.Second)
		testModel := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				hasNavigated:       true,
				lastNavigationTime: pastTime,
				searchActive:       true,
			},
		}

		// Process navigationTimeoutMsg
		result, _ := testModel.updateApp(navigationTimeoutMsg{})
		resultAppModel := result.(model).appModel

		// hasNavigated should still be true (search mode is active)
		if !resultAppModel.hasNavigated {
			t.Error("hasNavigated should remain true when in search mode")
		}
	})

	t.Run("all navigation handlers set time and return command", func(t *testing.T) {
		// Test model with multiple goals in a grid layout
		testModel := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
				},
				cursor: 0,
				width:  80,
				height: 24,
			},
		}

		handlers := []struct {
			name    string
			handler func(model) (tea.Model, tea.Cmd)
		}{
			{"up", handleNavigationUp},
			{"down", handleNavigationDown},
			{"left", handleNavigationLeft},
			{"right", handleNavigationRight},
		}

		for _, h := range handlers {
			t.Run(h.name, func(t *testing.T) {
				result, cmd := h.handler(testModel)
				resultModel := result.(model)

				if !resultModel.appModel.hasNavigated {
					t.Errorf("%s: hasNavigated should be true", h.name)
				}

				if resultModel.appModel.lastNavigationTime.IsZero() {
					t.Errorf("%s: lastNavigationTime should be set", h.name)
				}

				if cmd == nil {
					t.Errorf("%s: should return navigationTimeoutCmd", h.name)
				}
			})
		}
	})
}

// TestEnsureRowVisible tests the ensureRowVisible helper function
func TestEnsureRowVisible(t *testing.T) {
	tests := []struct {
		name        string
		selectedRow int
		firstRow    int
		visibleRows int
		totalRows   int
		expectedRow int
		description string
	}{
		{
			name:        "selection within viewport - no scroll",
			selectedRow: 2,
			firstRow:    0,
			visibleRows: 5,
			totalRows:   10,
			expectedRow: 0,
			description: "When selection is already visible, no scroll needed",
		},
		{
			name:        "selection above viewport - scroll up",
			selectedRow: 1,
			firstRow:    3,
			visibleRows: 3,
			totalRows:   10,
			expectedRow: 1,
			description: "When selection is above viewport, scroll up to show it",
		},
		{
			name:        "selection below viewport - scroll down",
			selectedRow: 5,
			firstRow:    0,
			visibleRows: 3,
			totalRows:   10,
			expectedRow: 3,
			description: "When selection is below viewport, scroll down to show it at bottom",
		},
		{
			name:        "clamp at top",
			selectedRow: 0,
			firstRow:    -1,
			visibleRows: 3,
			totalRows:   10,
			expectedRow: 0,
			description: "scrollRow should never be negative",
		},
		{
			name:        "clamp at bottom",
			selectedRow: 9,
			firstRow:    0,
			visibleRows: 3,
			totalRows:   10,
			expectedRow: 7,
			description: "scrollRow should not exceed totalRows - visibleRows",
		},
		{
			name:        "all rows fit in viewport",
			selectedRow: 2,
			firstRow:    0,
			visibleRows: 10,
			totalRows:   5,
			expectedRow: 0,
			description: "When all rows fit, scrollRow should be 0",
		},
		{
			name:        "tiny viewport (1 row)",
			selectedRow: 5,
			firstRow:    0,
			visibleRows: 1,
			totalRows:   10,
			expectedRow: 5,
			description: "With 1 visible row, scrollRow should equal selectedRow",
		},
		{
			name:        "guard against zero visibleRows",
			selectedRow: 2,
			firstRow:    0,
			visibleRows: 0,
			totalRows:   10,
			expectedRow: 2,
			description: "visibleRows < 1 should be treated as 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureRowVisible(tt.selectedRow, tt.firstRow, tt.visibleRows, tt.totalRows)
			if result != tt.expectedRow {
				t.Errorf("ensureRowVisible(%d, %d, %d, %d) = %d, want %d\n%s",
					tt.selectedRow, tt.firstRow, tt.visibleRows, tt.totalRows,
					result, tt.expectedRow, tt.description)
			}
		})
	}
}

// TestScrollFollowsNavigation tests that navigation handlers update scrollRow appropriately
func TestScrollFollowsNavigation(t *testing.T) {
	// Setup: Create a grid with 10 goals, 2 columns, height allowing 3 visible rows
	// With 2 cols and 10 goals, we have 5 total rows
	// With height=24, visibleRows = max(1, (24-4)/4) = 5 rows visible
	// So all rows fit - no scrolling should happen in this case

	// Let's test with a taller grid that requires scrolling:
	// 10 goals, 2 columns = 5 rows total
	// height=16 -> visibleRows = max(1, (16-4)/4) = 3 rows visible
	// So rows 0,1,2 are visible initially, rows 3,4 need scrolling

	t.Run("down navigation scrolls viewport when moving past bottom edge", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
					{Slug: "goal5", Title: "Goal 5", Losedate: 1234567894},
					{Slug: "goal6", Title: "Goal 6", Losedate: 1234567895},
					{Slug: "goal7", Title: "Goal 7", Losedate: 1234567896},
					{Slug: "goal8", Title: "Goal 8", Losedate: 1234567897},
					{Slug: "goal9", Title: "Goal 9", Losedate: 1234567898},
					{Slug: "goal10", Title: "Goal 10", Losedate: 1234567899},
				},
				cursor:    4, // Row 2 (0-indexed), column 0
				scrollRow: 0,
				width:     40, // 2 columns
				height:    16, // 3 visible rows
			},
		}

		// Navigate down from row 2 to row 3 (cursor 4 -> 6)
		result, _ := handleNavigationDown(m)
		resultModel := result.(model)

		// Cursor should move down by 2 (1 full row)
		if resultModel.appModel.cursor != 6 {
			t.Errorf("cursor should be 6, got %d", resultModel.appModel.cursor)
		}

		// ScrollRow should adjust to keep row 3 visible
		// Row 3 is outside viewport [0,2], so scrollRow should move to 1
		// making viewport [1,3] which includes row 3
		if resultModel.appModel.scrollRow != 1 {
			t.Errorf("scrollRow should be 1 to show row 3, got %d", resultModel.appModel.scrollRow)
		}
	})

	t.Run("up navigation scrolls viewport when moving past top edge", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
					{Slug: "goal5", Title: "Goal 5", Losedate: 1234567894},
					{Slug: "goal6", Title: "Goal 6", Losedate: 1234567895},
					{Slug: "goal7", Title: "Goal 7", Losedate: 1234567896},
					{Slug: "goal8", Title: "Goal 8", Losedate: 1234567897},
				},
				cursor:    4,  // Row 2, column 0
				scrollRow: 2,  // Viewport shows rows [2,4]
				width:     40, // 2 columns
				height:    16, // 3 visible rows
			},
		}

		// Navigate up from row 2 to row 1 (cursor 4 -> 2)
		result, _ := handleNavigationUp(m)
		resultModel := result.(model)

		// Cursor should move up by 2 (1 full row)
		if resultModel.appModel.cursor != 2 {
			t.Errorf("cursor should be 2, got %d", resultModel.appModel.cursor)
		}

		// ScrollRow should adjust to keep row 1 visible
		// Row 1 is outside viewport [2,4], so scrollRow should move to 1
		// making viewport [1,3] which includes row 1
		if resultModel.appModel.scrollRow != 1 {
			t.Errorf("scrollRow should be 1 to show row 1, got %d", resultModel.appModel.scrollRow)
		}
	})

	t.Run("scrollRow clamps to valid range at boundaries", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
				},
				cursor:    0, // First goal
				scrollRow: 0,
				width:     40, // 2 columns
				height:    16, // 3 visible rows
			},
		}

		// Try to navigate up from the first row - cursor shouldn't move
		result, _ := handleNavigationUp(m)
		resultModel := result.(model)

		if resultModel.appModel.cursor != 0 {
			t.Errorf("cursor should stay at 0, got %d", resultModel.appModel.cursor)
		}

		if resultModel.appModel.scrollRow != 0 {
			t.Errorf("scrollRow should stay at 0, got %d", resultModel.appModel.scrollRow)
		}
	})

	t.Run("navigation within visible viewport does not scroll", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
					{Slug: "goal5", Title: "Goal 5", Losedate: 1234567894},
					{Slug: "goal6", Title: "Goal 6", Losedate: 1234567895},
				},
				cursor:    0, // Row 0, col 0
				scrollRow: 0,
				width:     40, // 2 columns
				height:    16, // 3 visible rows
			},
		}

		// Navigate down to row 1 (cursor 0 -> 2)
		result, _ := handleNavigationDown(m)
		resultModel := result.(model)

		// Cursor should move
		if resultModel.appModel.cursor != 2 {
			t.Errorf("cursor should be 2, got %d", resultModel.appModel.cursor)
		}

		// ScrollRow should not change since row 1 is still visible in [0,2]
		if resultModel.appModel.scrollRow != 0 {
			t.Errorf("scrollRow should stay at 0, got %d", resultModel.appModel.scrollRow)
		}
	})
}

// mockMouseMsg creates a mock MouseMsg for testing
func mockMouseMsg(x, y int, button tea.MouseButton, action tea.MouseAction) tea.MouseMsg {
	return tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: button,
		Action: action,
	}
}

// TestHandleMouseClick tests the mouse click handler function
func TestHandleMouseClick(t *testing.T) {
	t.Run("click on first goal opens modal", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80, // 4 columns (80/20)
				height:       24,
				hasNavigated: false,
			},
		}

		// Click on the first goal (row 0, col 0)
		// Header is 2 lines, so clicking at Y=2 is the first row of goals
		// X=0 is in the first column
		msg := mockMouseMsg(0, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, cmd := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Check modal is open
		if resultModel.appModel.mode != modeGoalDetail {
			t.Error("mode should be modeGoalDetail after clicking a goal")
		}

		// Check correct goal is selected
		if resultModel.appModel.modalGoal == nil {
			t.Error("modalGoal should not be nil after clicking a goal")
		} else if resultModel.appModel.modalGoal.Slug != "goal1" {
			t.Errorf("modalGoal.Slug should be 'goal1', got '%s'", resultModel.appModel.modalGoal.Slug)
		}

		// Check hasNavigated is true
		if !resultModel.appModel.hasNavigated {
			t.Error("hasNavigated should be true after clicking a goal")
		}

		// Check command is returned
		if cmd == nil {
			t.Error("command should not be nil after clicking a goal")
		}
	})

	t.Run("click on second column goal", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        40, // 2 columns (40/20)
				height:       24,
				hasNavigated: false,
			},
		}

		// Click on the second goal (row 0, col 1)
		// With 2 columns and width 40, each cell is ~20 pixels wide
		// X=25 is in the second column
		msg := mockMouseMsg(25, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, _ := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Check modal is open with goal2
		if resultModel.appModel.mode != modeGoalDetail {
			t.Error("mode should be modeGoalDetail after clicking a goal")
		}

		if resultModel.appModel.modalGoal == nil {
			t.Error("modalGoal should not be nil after clicking a goal")
		} else if resultModel.appModel.modalGoal.Slug != "goal2" {
			t.Errorf("modalGoal.Slug should be 'goal2', got '%s'", resultModel.appModel.modalGoal.Slug)
		}
	})

	t.Run("click on second row goal", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        40, // 2 columns
				height:       24,
				hasNavigated: false,
			},
		}

		// Click on the third goal (row 1, col 0)
		// Each row is 4 lines high, so Y=6 (2 header + 4 first row) is in second row
		msg := mockMouseMsg(0, 6, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, _ := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Check modal is open with goal3
		if resultModel.appModel.mode != modeGoalDetail {
			t.Error("mode should be modeGoalDetail after clicking a goal")
		}

		if resultModel.appModel.modalGoal == nil {
			t.Error("modalGoal should not be nil after clicking a goal")
		} else if resultModel.appModel.modalGoal.Slug != "goal3" {
			t.Errorf("modalGoal.Slug should be 'goal3', got '%s'", resultModel.appModel.modalGoal.Slug)
		}
	})

	t.Run("click with scroll offset", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
					{Slug: "goal4", Title: "Goal 4", Losedate: 1234567893},
					{Slug: "goal5", Title: "Goal 5", Losedate: 1234567894},
					{Slug: "goal6", Title: "Goal 6", Losedate: 1234567895},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    1,  // Scrolled down by 1 row
				width:        40, // 2 columns
				height:       24,
				hasNavigated: false,
			},
		}

		// Click on the first visible row (row 1 on screen = row 2 in data)
		// Since scrollRow is 1, clicking the first visible row selects goal3 and goal4
		// X=0 is first column = goal3
		msg := mockMouseMsg(0, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, _ := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Check modal is open with goal3 (row 1, col 0)
		if resultModel.appModel.mode != modeGoalDetail {
			t.Error("mode should be modeGoalDetail after clicking a goal")
		}

		if resultModel.appModel.modalGoal == nil {
			t.Error("modalGoal should not be nil after clicking a goal")
		} else if resultModel.appModel.modalGoal.Slug != "goal3" {
			t.Errorf("modalGoal.Slug should be 'goal3', got '%s'", resultModel.appModel.modalGoal.Slug)
		}
	})

	t.Run("click on empty space does nothing", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80, // 4 columns
				height:       24,
				hasNavigated: false,
			},
		}

		// Click on empty space (second column but only 1 goal exists)
		msg := mockMouseMsg(25, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, cmd := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Modal should not open
		if resultModel.appModel.mode != modeBrowse {
			t.Error("mode should stay modeBrowse when clicking empty space")
		}

		// No command should be returned
		if cmd != nil {
			t.Error("command should be nil when clicking empty space")
		}
	})

	t.Run("click on header area does nothing", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80,
				height:       24,
				hasNavigated: false,
			},
		}

		// Click on header area (Y=0 or Y=1)
		msg := mockMouseMsg(0, 0, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, cmd := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Modal should not open
		if resultModel.appModel.mode != modeBrowse {
			t.Error("mode should stay modeBrowse when clicking header area")
		}

		if cmd != nil {
			t.Error("command should be nil when clicking header area")
		}
	})

	t.Run("click with no goals does nothing", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals:        []Goal{},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80,
				height:       24,
				hasNavigated: false,
			},
		}

		msg := mockMouseMsg(0, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, cmd := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Modal should not open
		if resultModel.appModel.mode != modeBrowse {
			t.Error("mode should stay modeBrowse when no goals exist")
		}

		if cmd != nil {
			t.Error("command should be nil when no goals exist")
		}
	})

	t.Run("click updates cursor to match original goals list", func(t *testing.T) {
		m := model{
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
					{Slug: "goal3", Title: "Goal 3", Losedate: 1234567892},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80,
				height:       24,
				hasNavigated: false,
			},
		}

		// Click on goal2 (assuming it's in the second column or first row position 1)
		// With width 80, we have 4 columns, so X=25 should be in column 1
		msg := mockMouseMsg(25, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, _ := handleMouseClick(m, msg)
		resultModel := result.(model)

		// Cursor should point to goal2's position in original list (index 1)
		if resultModel.appModel.cursor != 1 {
			t.Errorf("cursor should be 1, got %d", resultModel.appModel.cursor)
		}
	})
}

// handleAddDatapoint is the modal-open path that pre-fills the input form
// with the goal's last datapoint value (defaulting to "1" if the value is
// zero or the API errors). It was previously untestable because every
// invocation hit the real HTTPClient; now FakeClient lets us cover all
// three branches cheaply.

func TestHandleAddDatapointPrefillsLastValue(t *testing.T) {
	fake := &FakeClient{
		GetLastDatapointValueFunc: func(slug string) (float64, error) {
			if slug != "exercise" {
				t.Errorf("client called with slug=%q, want exercise", slug)
			}
			return 2.5, nil
		},
	}
	m := model{
		state: "app",
		appModel: appModel{
			client:    fake,
			modalGoal: &Goal{Slug: "exercise"},
			mode:      modeGoalDetail,
		},
	}

	updated, _ := handleAddDatapoint(m)
	got := updated.(model).appModel
	if got.datapoint.value() != "2.5" {
		t.Errorf("inputValue = %q, want %q", got.datapoint.value(), "2.5")
	}
	if got.mode != modeDatapointInput {
		t.Error("expected mode to be modeDatapointInput after handleAddDatapoint")
	}
	if got.datapoint.comment() != "Added via buzz" {
		t.Errorf("inputComment = %q, want default %q", got.datapoint.comment(), "Added via buzz")
	}
}

func TestHandleAddDatapointDefaultsToOneOnZeroValue(t *testing.T) {
	// API returned the goal but the last datapoint value was zero — buzz
	// treats that as "no useful default" and falls back to "1".
	// Track whether the client was called so a future change that stops
	// querying it (and just hard-codes "1") would fail this test.
	called := false
	fake := &FakeClient{
		GetLastDatapointValueFunc: func(string) (float64, error) {
			called = true
			return 0, nil
		},
	}
	m := model{
		state: "app",
		appModel: appModel{
			client:    fake,
			modalGoal: &Goal{Slug: "any"},
			mode:      modeGoalDetail,
		},
	}

	updated, _ := handleAddDatapoint(m)
	if !called {
		t.Error("expected GetLastDatapointValue to be called")
	}
	got := updated.(model).appModel
	if got.datapoint.value() != "1" {
		t.Errorf("inputValue with zero last value = %q, want %q", got.datapoint.value(), "1")
	}
	if got.mode != modeDatapointInput {
		t.Error("expected mode to be modeDatapointInput after handleAddDatapoint")
	}
	if got.datapoint.comment() != "Added via buzz" {
		t.Errorf("inputComment = %q, want default %q", got.datapoint.comment(), "Added via buzz")
	}
}

func TestHandleAddDatapointDefaultsToOneOnFetchError(t *testing.T) {
	// API errored — same fallback to "1" rather than blocking the modal.
	// Track the call so the fallback can't accidentally short-circuit it.
	called := false
	fake := &FakeClient{
		GetLastDatapointValueFunc: func(string) (float64, error) {
			called = true
			return 0, errFakeNotConfigured
		},
	}
	m := model{
		state: "app",
		appModel: appModel{
			client:    fake,
			modalGoal: &Goal{Slug: "any"},
			mode:      modeGoalDetail,
		},
	}

	updated, _ := handleAddDatapoint(m)
	if !called {
		t.Error("expected GetLastDatapointValue to be called")
	}
	got := updated.(model).appModel
	if got.datapoint.value() != "1" {
		t.Errorf("inputValue on fetch error = %q, want %q", got.datapoint.value(), "1")
	}
	if got.mode != modeDatapointInput {
		t.Error("expected mode to be modeDatapointInput after handleAddDatapoint")
	}
	if got.datapoint.comment() != "Added via buzz" {
		t.Errorf("inputComment = %q, want default %q", got.datapoint.comment(), "Added via buzz")
	}
}

// TestMouseClickIntegration tests mouse click through the full updateApp path
func TestMouseClickIntegration(t *testing.T) {
	t.Run("mouse click only triggers on left button release", func(t *testing.T) {
		m := model{
			state: "app",
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80,
				height:       24,
				hasNavigated: false,
			},
		}

		// Test right click - should not open modal
		rightClickMsg := mockMouseMsg(0, 2, tea.MouseButtonRight, tea.MouseActionRelease)
		result, _ := m.updateApp(rightClickMsg)
		resultModel := result.(model)
		if resultModel.appModel.mode != modeBrowse {
			t.Error("right click should not open modal")
		}

		// Test left button press (not release) - should not open modal
		pressMsg := mockMouseMsg(0, 2, tea.MouseButtonLeft, tea.MouseActionPress)
		result, _ = m.updateApp(pressMsg)
		resultModel = result.(model)
		if resultModel.appModel.mode != modeBrowse {
			t.Error("mouse press should not open modal")
		}

		// Test left button release - should open modal
		releaseMsg := mockMouseMsg(0, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, _ = m.updateApp(releaseMsg)
		resultModel = result.(model)
		if resultModel.appModel.mode != modeGoalDetail {
			t.Error("left click release should open modal")
		}
	})

	t.Run("mouse click ignored when modal is open", func(t *testing.T) {
		m := model{
			state: "app",
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
					{Slug: "goal2", Title: "Goal 2", Losedate: 1234567891},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80,
				height:       24,
				mode:         modeGoalDetail,
				modalGoal:    &Goal{Slug: "goal1"},
				hasNavigated: false,
			},
		}

		// Click should be ignored since modal is open
		msg := mockMouseMsg(25, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, _ := m.updateApp(msg)
		resultModel := result.(model)

		// Modal should still be showing the same goal
		if resultModel.appModel.mode != modeGoalDetail {
			t.Error("modal should still be open")
		}
		if resultModel.appModel.modalGoal.Slug != "goal1" {
			t.Error("modal goal should not change when clicking with modal open")
		}
	})

	t.Run("mouse click ignored when create modal is open", func(t *testing.T) {
		m := model{
			state: "app",
			appModel: appModel{
				goals: []Goal{
					{Slug: "goal1", Title: "Goal 1", Losedate: 1234567890},
				},
				config:       &Config{Username: "testuser", AuthToken: "testtoken"},
				cursor:       0,
				scrollRow:    0,
				width:        80,
				height:       24,
				mode:         modeCreateGoal,
				hasNavigated: false,
			},
		}

		// Click should be ignored since create modal is open
		msg := mockMouseMsg(0, 2, tea.MouseButtonLeft, tea.MouseActionRelease)
		result, _ := m.updateApp(msg)
		resultModel := result.(model)

		// Goal modal should not open
		if resultModel.appModel.mode != modeCreateGoal {
			t.Error("goal modal should not open when create modal is open")
		}
	})
}

// TestHandleEscapeKeyLadder covers the Esc "back out one level" ladder, the
// busy-form lock during in-flight writes, and the search-vs-modal precedence.
func TestHandleEscapeKeyLadder(t *testing.T) {
	t.Run("datapoint input cancels back to goal detail", func(t *testing.T) {
		m := model{appModel: appModel{modalGoal: &Goal{Slug: "g"}, mode: modeGoalDetail}}
		m.appModel.startDatapointInput(newDatapointForm("1"))
		got := mustModel(t, mustTeaModel(handleEscapeKey(m))).appModel
		if got.mode != modeGoalDetail {
			t.Errorf("Esc from datapoint input: mode = %d, want modeGoalDetail", got.mode)
		}
	})

	t.Run("Esc is locked while a datapoint submit is in-flight", func(t *testing.T) {
		m := model{appModel: appModel{modalGoal: &Goal{Slug: "g"}, mode: modeDatapointInput}}
		m.appModel.datapoint = newDatapointForm("1")
		m.appModel.datapoint.submitting = true
		got := mustModel(t, mustTeaModel(handleEscapeKey(m))).appModel
		if got.mode != modeDatapointInput {
			t.Errorf("Esc during submit should be locked: mode = %d, want modeDatapointInput", got.mode)
		}
	})

	t.Run("Esc is locked while a goal create is in-flight", func(t *testing.T) {
		m := model{appModel: appModel{mode: modeCreateGoal, createGoal: newCreateGoalForm()}}
		m.appModel.createGoal.creating = true
		got := mustModel(t, mustTeaModel(handleEscapeKey(m))).appModel
		if got.mode != modeCreateGoal {
			t.Errorf("Esc during create should be locked: mode = %d, want modeCreateGoal", got.mode)
		}
	})

	t.Run("goal detail over a search closes the modal and keeps the search", func(t *testing.T) {
		m := model{appModel: appModel{searchActive: true, searchQuery: "weight", modalGoal: &Goal{Slug: "weight"}, mode: modeGoalDetail}}
		got := mustModel(t, mustTeaModel(handleEscapeKey(m))).appModel
		if got.mode != modeBrowse {
			t.Errorf("Esc on modal: mode = %d, want modeBrowse", got.mode)
		}
		if !got.searchActive || got.searchQuery != "weight" {
			t.Error("Esc on modal should keep the search layer intact")
		}
	})

	t.Run("search then exits on the next Esc", func(t *testing.T) {
		m := model{appModel: appModel{searchActive: true, searchQuery: "weight", mode: modeBrowse}}
		got := mustModel(t, mustTeaModel(handleEscapeKey(m))).appModel
		if got.searchActive {
			t.Error("Esc in browse+search should exit search")
		}
	})

	t.Run("browse with no search quits", func(t *testing.T) {
		m := model{appModel: appModel{mode: modeBrowse}}
		_, cmd := handleEscapeKey(m)
		if cmd == nil {
			t.Error("Esc in plain browse should return a quit command")
		}
	})
}

// mustTeaModel extracts the tea.Model from a handler's (tea.Model, tea.Cmd)
// return so it can be chained into mustModel.
func mustTeaModel(tm tea.Model, _ tea.Cmd) tea.Model { return tm }

// TestModeHandlerTransitions covers the handlers that drive mode transitions
// (search/create entry, in-modal goal navigation, and opening the detail modal
// from the grid) — the thin layer between key presses and the transition
// methods that TestModeTransitions exercises directly.
func TestModeHandlerTransitions(t *testing.T) {
	t.Run("'/' enters the search layer from browse", func(t *testing.T) {
		m := model{appModel: appModel{mode: modeBrowse}}
		got := mustModel(t, mustTeaModel(handleEnterSearch(m))).appModel
		if !got.searchActive || got.searchQuery != "" {
			t.Errorf("handleEnterSearch: searchActive=%v query=%q, want true/empty", got.searchActive, got.searchQuery)
		}
		if got.mode != modeBrowse {
			t.Errorf("handleEnterSearch should leave mode as browse, got %d", got.mode)
		}
	})

	t.Run("'/' is a no-op while search already active", func(t *testing.T) {
		m := model{appModel: appModel{mode: modeBrowse, searchActive: true, searchQuery: "keep"}}
		got := mustModel(t, mustTeaModel(handleEnterSearch(m))).appModel
		if got.searchQuery != "keep" {
			t.Errorf("handleEnterSearch should not reset an active query, got %q", got.searchQuery)
		}
	})

	t.Run("'n' opens the create-goal form from browse", func(t *testing.T) {
		m := model{appModel: appModel{mode: modeBrowse}}
		got := mustModel(t, mustTeaModel(handleCreateGoal(m))).appModel
		if got.mode != modeCreateGoal {
			t.Errorf("handleCreateGoal: mode=%d, want modeCreateGoal", got.mode)
		}
	})

	t.Run("'n' is a no-op while search is active", func(t *testing.T) {
		m := model{appModel: appModel{mode: modeBrowse, searchActive: true}}
		got := mustModel(t, mustTeaModel(handleCreateGoal(m))).appModel
		if got.mode != modeBrowse {
			t.Errorf("handleCreateGoal during search should be a no-op, mode=%d", got.mode)
		}
	})

	t.Run("Enter opens the detail modal and syncs cursor to the original list", func(t *testing.T) {
		goals := []Goal{{Slug: "a"}, {Slug: "b"}, {Slug: "c"}}
		// Search filters to just "b" so displayGoals[0] maps to original index 1.
		m := model{appModel: appModel{
			goals:        goals,
			mode:         modeBrowse,
			searchActive: true,
			searchQuery:  "b",
			cursor:       0,
			client:       &FakeClient{},
		}}
		updated, cmd := handleEnterKey(m)
		got := mustModel(t, updated).appModel
		if got.mode != modeGoalDetail {
			t.Errorf("Enter: mode=%d, want modeGoalDetail", got.mode)
		}
		if got.modalGoal == nil || got.modalGoal.Slug != "b" {
			t.Errorf("Enter should open goal 'b', got %v", got.modalGoal)
		}
		if got.cursor != 1 {
			t.Errorf("Enter should sync cursor to original index 1, got %d", got.cursor)
		}
		if cmd == nil {
			t.Error("Enter should return a loadGoalDetails command")
		}
	})

	t.Run("left/right navigate goals within the detail modal", func(t *testing.T) {
		goals := []Goal{{Slug: "a"}, {Slug: "b"}, {Slug: "c"}}
		base := appModel{goals: goals, mode: modeGoalDetail, cursor: 1, modalGoal: &goals[1], client: &FakeClient{}}

		left := mustModel(t, mustTeaModel(handleNavigationLeft(model{appModel: base}))).appModel
		if left.cursor != 0 || left.modalGoal == nil || left.modalGoal.Slug != "a" {
			t.Errorf("left nav: cursor=%d goal=%v, want 0/'a'", left.cursor, left.modalGoal)
		}

		right := mustModel(t, mustTeaModel(handleNavigationRight(model{appModel: base}))).appModel
		if right.cursor != 2 || right.modalGoal == nil || right.modalGoal.Slug != "c" {
			t.Errorf("right nav: cursor=%d goal=%v, want 2/'c'", right.cursor, right.modalGoal)
		}
	})
}
