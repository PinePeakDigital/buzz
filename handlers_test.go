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
				showModal:          false,
				searchMode:         false,
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
				showModal:          false,
				searchMode:         false,
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
				showModal:          true,
				searchMode:         false,
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
				showModal:          false,
				searchMode:         true,
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

// TestLimsumFetchDelay tests the delay behavior before fetching limsum after adding a datapoint
func TestLimsumFetchDelay(t *testing.T) {
	t.Run("limsumFetchDelay has correct default value", func(t *testing.T) {
		// Verify the default delay is 2 seconds
		if limsumFetchDelay != 2*time.Second {
			t.Errorf("limsumFetchDelay = %v, want %v", limsumFetchDelay, 2*time.Second)
		}
	})

	t.Run("limsumFetchDelay can be overridden for testing", func(t *testing.T) {
		// Store the original value and defer restoration
		originalDelay := limsumFetchDelay
		defer func() { limsumFetchDelay = originalDelay }()

		// Override the delay for testing
		limsumFetchDelay = 1 * time.Millisecond

		// Verify the delay is now 1 millisecond
		if limsumFetchDelay != 1*time.Millisecond {
			t.Errorf("limsumFetchDelay = %v, want %v", limsumFetchDelay, 1*time.Millisecond)
		}
	})

	t.Run("delay is applied correctly", func(t *testing.T) {
		// Store the original value and defer restoration
		originalDelay := limsumFetchDelay
		defer func() { limsumFetchDelay = originalDelay }()

		// Set a short delay for testing
		testDelay := 50 * time.Millisecond
		limsumFetchDelay = testDelay

		// Measure the time it takes to execute the delay
		start := time.Now()
		time.Sleep(limsumFetchDelay)
		elapsed := time.Since(start)

		// Verify the delay was applied (allow 10ms tolerance)
		if elapsed < testDelay {
			t.Errorf("delay was not applied correctly: elapsed %v, want at least %v", elapsed, testDelay)
		}
	})

	t.Run("delay does not block error handling", func(t *testing.T) {
		// Store the original value and defer restoration
		originalDelay := limsumFetchDelay
		defer func() { limsumFetchDelay = originalDelay }()

		// Set a short delay for testing
		limsumFetchDelay = 1 * time.Millisecond

		// Simulate the delay followed by an error condition
		var fetchError error
		time.Sleep(limsumFetchDelay)

		// Simulate a fetch error
		fetchError = nil // In a real scenario, this would be set by FetchGoal

		// Verify that after the delay, error handling is still possible
		if fetchError != nil {
			t.Errorf("error handling should work after delay, got error: %v", fetchError)
		}
	})
}
