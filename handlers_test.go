package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// mockKeyMsg creates a mock KeyMsg for testing
func mockKeyMsg(runes []rune) tea.KeyMsg {
	return tea.KeyMsg{
		Runes: runes,
	}
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

// TestHandleNumericDecimalInput tests the handleNumericDecimalInput helper function
func TestHandleNumericDecimalInput(t *testing.T) {
	tests := []struct {
		name          string
		char          string
		initialValue  string
		expectedValue string
		expectedOk    bool
	}{
		{"valid digit", "5", "10", "105", true},
		{"valid decimal", ".", "10", "10.", true},
		{"valid negative", "-", "", "-", true},
		{"valid null char n", "n", "", "n", true},
		{"valid null char u", "u", "n", "nu", true},
		{"valid null char l", "l", "nu", "nul", true},
		{"invalid letter", "a", "10", "10", false},
		{"invalid space", " ", "10", "10", false},
		{"invalid special", "@", "10", "10", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test model
			m := model{
				appModel: appModel{},
			}
			field := tt.initialValue

			// Call the function
			resultModel, ok := handleNumericDecimalInput(m, tt.char, &field)

			// Verify the result
			if ok != tt.expectedOk {
				t.Errorf("handleNumericDecimalInput(%q) returned ok=%v, want %v", tt.char, ok, tt.expectedOk)
			}
			if field != tt.expectedValue {
				t.Errorf("handleNumericDecimalInput(%q) resulted in field=%q, want %q", tt.char, field, tt.expectedValue)
			}
			// Verify model is returned unchanged
			if resultModel.appModel.createGoalval != "" {
				t.Errorf("handleNumericDecimalInput should not modify model")
			}
		})
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
					searchMode: true,
					showModal:  false,
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
		{"Title with ASCII", 1, []rune{'a'}, true, func(a appModel) string { return a.createTitle }, "Title"},
		{"Title with accented char", 1, []rune{'é'}, true, func(a appModel) string { return a.createTitle }, "Title"},
		{"Title with Chinese", 1, []rune{'中'}, true, func(a appModel) string { return a.createTitle }, "Title"},
		{"Title with emoji", 1, []rune{'😀'}, true, func(a appModel) string { return a.createTitle }, "Title"},
		{"Title with Greek", 1, []rune{'Ω'}, true, func(a appModel) string { return a.createTitle }, "Title"},
		{"Gunits with ASCII", 3, []rune{'x'}, true, func(a appModel) string { return a.createGunits }, "Gunits"},
		{"Gunits with accented char", 3, []rune{'ñ'}, true, func(a appModel) string { return a.createGunits }, "Gunits"},
		{"Gunits with Cyrillic", 3, []rune{'Д'}, true, func(a appModel) string { return a.createGunits }, "Gunits"},
		{"Gunits with emoji", 3, []rune{'📊'}, true, func(a appModel) string { return a.createGunits }, "Gunits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock model with create modal open
			m := model{
				appModel: appModel{
					showCreateModal: true,
					creatingGoal:    false,
					createFocus:     tt.focus,
					createTitle:     "",
					createGunits:    "",
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
			// Create a mock model in input mode with comment field focused
			m := model{
				appModel: appModel{
					showModal:    true,
					inputMode:    true,
					submitting:   false,
					inputFocus:   2, // Comment field
					inputComment: "",
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
				if updatedModel.appModel.inputComment != expectedComment {
					t.Errorf("inputComment = %q, want %q", updatedModel.appModel.inputComment, expectedComment)
				}
			}
		})
	}
}
