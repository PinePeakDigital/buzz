package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// handleSearchInput handles text input in search mode
func handleSearchInput(m model, msg tea.KeyMsg) (model, bool) {
	if m.appModel.searchMode && !m.appModel.showModal {
		// Allow printable Unicode characters in search
		if len(msg.Runes) == 1 && unicode.IsPrint(msg.Runes[0]) {
			m.appModel.searchQuery += string(msg.Runes)
			// Reset cursor and scroll when search query changes
			m.appModel.cursor = 0
			m.appModel.scrollRow = 0
			m.appModel.hasNavigated = false
			return m, true
		}
	}
	return m, false
}

// isAlphanumericOrDash checks if character is alphanumeric, dash, or underscore
func isAlphanumericOrDash(char string) bool {
	if len(char) != 1 {
		return false
	}
	c := char[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_'
}

// isLetter checks if character is a letter
func isLetter(char string) bool {
	if len(char) != 1 {
		return false
	}
	c := char[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isNumericOrNull checks if character is numeric or part of "null"
// currentValue is the current field value before adding the new character
func isNumericOrNull(char string, currentValue string) bool {
	if len(char) != 1 {
		return false
	}
	c := char[0]

	// Allow digits
	if c >= '0' && c <= '9' {
		return true
	}

	// Check if adding this character would form a valid prefix of "null"
	newValue := currentValue + char
	return strings.HasPrefix("null", newValue)
}

// isNumericWithDecimal checks if character is numeric, decimal, negative, or part of "null"
// currentValue is the current field value before adding the new character
func isNumericWithDecimal(char string, currentValue string) bool {
	if len(char) != 1 {
		return false
	}
	c := char[0]

	// Allow digits, decimal point, and negative sign
	if (c >= '0' && c <= '9') || c == '.' || c == '-' {
		return true
	}

	// Check if adding this character would form a valid prefix of "null"
	newValue := currentValue + char
	return strings.HasPrefix("null", newValue)
}

// handleNumericDecimalInput handles input for fields that accept numeric with decimal values
func handleNumericDecimalInput(m model, char string, fieldPtr *string) (model, bool) {
	if isNumericWithDecimal(char, *fieldPtr) {
		*fieldPtr += char
		return m, true
	}
	return m, false
}

// handleCreateModalInput handles text input in create goal modal
func handleCreateModalInput(m model, msg tea.KeyMsg) (model, bool) {
	if !m.appModel.showCreateModal || m.appModel.creatingGoal {
		return m, false
	}
	if len(msg.Runes) != 1 {
		return m, false
	}

	char := string(msg.Runes)
	r := msg.Runes[0]

	switch m.appModel.createFocus {
	case 0: // Slug - allow alphanumeric and dashes/underscores
		if isAlphanumericOrDash(char) {
			m.appModel.createSlug += char
			return m, true
		}
	case 1: // Title - allow all printable Unicode characters
		if unicode.IsPrint(r) {
			m.appModel.createTitle += char
			return m, true
		}
	case 2: // Goal type - allow letters
		if isLetter(char) {
			m.appModel.createGoalType += char
			return m, true
		}
	case 3: // Gunits - allow all printable Unicode characters
		if unicode.IsPrint(r) {
			m.appModel.createGunits += char
			return m, true
		}
	case 4: // Goaldate - allow digits or "null"
		if isNumericOrNull(char, m.appModel.createGoaldate) {
			m.appModel.createGoaldate += char
			return m, true
		}
	case 5: // Goalval - allow digits, decimal point, negative sign, or "null"
		return handleNumericDecimalInput(m, char, &m.appModel.createGoalval)
	case 6: // Rate - allow digits, decimal point, negative sign, or "null"
		return handleNumericDecimalInput(m, char, &m.appModel.createRate)
	}
	return m, false
}

// handleDatapointInput handles text input in datapoint input mode
func handleDatapointInput(m model, msg tea.KeyMsg) (model, bool) {
	// Handle text input in input mode
	// This ensures that single-character command keys (like 't', 'r', 'd', etc.)
	// can still be typed in comment fields
	if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
		if len(msg.Runes) == 1 {
			char := string(msg.Runes)
			r := msg.Runes[0]
			switch m.appModel.inputFocus {
			case 0: // Date field - allow digits and dashes
				if (char >= "0" && char <= "9") || char == "-" {
					m.appModel.inputDate += char
					return m, true
				}
			case 1: // Value field - allow digits, decimal point, and negative sign
				if (char >= "0" && char <= "9") || char == "." || char == "-" {
					m.appModel.inputValue += char
					return m, true
				}
			case 2: // Comment field - allow all printable Unicode characters
				if unicode.IsPrint(r) {
					m.appModel.inputComment += char
					return m, true
				}
			}
		}
	}
	return m, false
}

// handleKeyPress processes keyboard input and returns updated model and command
func handleKeyPress(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle text input in search mode FIRST
	if updatedModel, handled := handleSearchInput(m, msg); handled {
		return updatedModel, nil
	}

	// Handle text input in create goal modal
	if updatedModel, handled := handleCreateModalInput(m, msg); handled {
		return updatedModel, nil
	}

	// Handle text input in datapoint input mode
	if updatedModel, handled := handleDatapointInput(m, msg); handled {
		return updatedModel, nil
	}

	// Cool, what was the actual key pressed?
	switch msg.String() {

	// These keys should exit the program.
	case "ctrl+c", "q":
		return m, tea.Quit

	// Escape key closes search mode, modal, or quits
	case "esc":
		return handleEscapeKey(m)

	// Enter input mode with 'a' (only when modal is open but not in input mode and not submitting)
	case "a":
		return handleAddDatapoint(m)

	// Tab navigation between input fields (only in input mode and not submitting)
	case "tab":
		return handleTabKey(m, false)

	// Shift+Tab navigation in input mode (reverse)
	case "shift+tab":
		return handleTabKey(m, true)

	// Backspace handling in search mode or input mode
	case "backspace":
		return handleBackspace(m)

	// Submit form with Enter in input mode
	case "enter":
		return handleEnterKey(m)

	// Navigation keys - spatial movement through grid (only when modal is closed)
	case "up", "k":
		return handleNavigationUp(m)

	case "down", "j":
		return handleNavigationDown(m)

	case "left", "h":
		return handleNavigationLeft(m)

	case "right", "l":
		return handleNavigationRight(m)

	// Scroll up with Page Up or 'u' (only when modal is closed)
	case "pgup", "u":
		return handleScrollUp(m)

	// Scroll down with Page Down or 'd' (only when modal is closed)
	case "pgdown", "d":
		return handleScrollDown(m)

	// Manual refresh with 'r' (only when modal is closed)
	case "r":
		return handleRefresh(m)

	// Toggle auto-refresh with 't' (only when modal is closed)
	case "t":
		return handleToggleRefresh(m)

	// Enter search mode with '/' (only when modal is closed and not already in search mode)
	case "/":
		return handleEnterSearch(m)

	// Open create goal modal with 'n' for new (only when no modal is open)
	case "n":
		return handleCreateGoal(m)
	}

	return m, nil
}

// handleEscapeKey handles the Escape key press
func handleEscapeKey(m model) (tea.Model, tea.Cmd) {
	if m.appModel.searchMode {
		// Exit search mode
		m.appModel.searchMode = false
		m.appModel.searchQuery = ""
		m.appModel.cursor = 0
		m.appModel.scrollRow = 0
		m.appModel.hasNavigated = false
	} else if m.appModel.showCreateModal {
		// Close create goal modal
		m.appModel.showCreateModal = false
		m.appModel.createError = ""
	} else if m.appModel.showModal {
		if m.appModel.inputMode {
			// Exit input mode
			m.appModel.inputMode = false
			m.appModel.inputFocus = 0
			m.appModel.inputError = ""
		} else {
			// Close modal
			m.appModel.showModal = false
			m.appModel.modalGoal = nil
		}
	} else {
		return m, tea.Quit
	}
	return m, nil
}

// handleAddDatapoint enters input mode for adding a datapoint
func handleAddDatapoint(m model) (tea.Model, tea.Cmd) {
	if m.appModel.showModal && !m.appModel.inputMode && !m.appModel.submitting {
		m.appModel.inputMode = true
		m.appModel.inputFocus = 0
		m.appModel.inputError = "" // Clear any previous errors
		// Set default values
		m.appModel.inputDate = time.Now().Format("2006-01-02")
		m.appModel.inputComment = "Added via buzz"

		// Try to get the last datapoint value, default to "1" if it fails
		if lastValue, err := GetLastDatapointValue(m.appModel.config, m.appModel.modalGoal.Slug); err == nil && lastValue != 0 {
			m.appModel.inputValue = fmt.Sprintf("%.1f", lastValue)
		} else {
			m.appModel.inputValue = "1"
		}
	}
	return m, nil
}

// handleTabKey handles Tab and Shift+Tab navigation
func handleTabKey(m model, reverse bool) (tea.Model, tea.Cmd) {
	if m.appModel.showCreateModal && !m.appModel.creatingGoal {
		if reverse {
			m.appModel.createFocus = (m.appModel.createFocus + 6) % 7 // +6 is same as -1 in mod 7
		} else {
			m.appModel.createFocus = (m.appModel.createFocus + 1) % 7
		}
	} else if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
		if reverse {
			m.appModel.inputFocus = (m.appModel.inputFocus + 2) % 3 // +2 is same as -1 in mod 3
		} else {
			m.appModel.inputFocus = (m.appModel.inputFocus + 1) % 3
		}
	}
	return m, nil
}

// handleBackspace handles Backspace key
func handleBackspace(m model) (tea.Model, tea.Cmd) {
	if m.appModel.showCreateModal && !m.appModel.creatingGoal {
		switch m.appModel.createFocus {
		case 0: // Slug
			if len(m.appModel.createSlug) > 0 {
				m.appModel.createSlug = m.appModel.createSlug[:len(m.appModel.createSlug)-1]
			}
		case 1: // Title
			if len(m.appModel.createTitle) > 0 {
				m.appModel.createTitle = m.appModel.createTitle[:len(m.appModel.createTitle)-1]
			}
		case 2: // Goal type
			if len(m.appModel.createGoalType) > 0 {
				m.appModel.createGoalType = m.appModel.createGoalType[:len(m.appModel.createGoalType)-1]
			}
		case 3: // Gunits
			if len(m.appModel.createGunits) > 0 {
				m.appModel.createGunits = m.appModel.createGunits[:len(m.appModel.createGunits)-1]
			}
		case 4: // Goaldate
			if len(m.appModel.createGoaldate) > 0 {
				m.appModel.createGoaldate = m.appModel.createGoaldate[:len(m.appModel.createGoaldate)-1]
			}
		case 5: // Goalval
			if len(m.appModel.createGoalval) > 0 {
				m.appModel.createGoalval = m.appModel.createGoalval[:len(m.appModel.createGoalval)-1]
			}
		case 6: // Rate
			if len(m.appModel.createRate) > 0 {
				m.appModel.createRate = m.appModel.createRate[:len(m.appModel.createRate)-1]
			}
		}
	} else if m.appModel.searchMode && !m.appModel.showModal {
		// Remove last character from search query
		if len(m.appModel.searchQuery) > 0 {
			m.appModel.searchQuery = m.appModel.searchQuery[:len(m.appModel.searchQuery)-1]
			// Reset cursor and scroll when search query changes
			m.appModel.cursor = 0
			m.appModel.scrollRow = 0
			m.appModel.hasNavigated = false
		}
	} else if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
		switch m.appModel.inputFocus {
		case 0: // Date field
			if len(m.appModel.inputDate) > 0 {
				m.appModel.inputDate = m.appModel.inputDate[:len(m.appModel.inputDate)-1]
			}
		case 1: // Value field
			if len(m.appModel.inputValue) > 0 {
				m.appModel.inputValue = m.appModel.inputValue[:len(m.appModel.inputValue)-1]
			}
		case 2: // Comment field
			if len(m.appModel.inputComment) > 0 {
				m.appModel.inputComment = m.appModel.inputComment[:len(m.appModel.inputComment)-1]
			}
		}
	}
	return m, nil
}

// validateDatapointInput validates datapoint input fields and returns error message if invalid
func validateDatapointInput(inputDate, inputValue string) string {
	if inputDate == "" {
		return "Date cannot be empty"
	}

	if inputValue == "" {
		return "Value cannot be empty"
	}

	// Parse and validate date
	date, err := time.Parse("2006-01-02", inputDate)
	if err != nil {
		return "Invalid date format (use YYYY-MM-DD)"
	}

	// Validate that date is not in the future beyond today
	if date.After(time.Now().AddDate(0, 0, 1)) {
		return "Date cannot be more than 1 day in the future"
	}

	// Parse and validate value (must be a valid number)
	if _, err := strconv.ParseFloat(inputValue, 64); err != nil {
		return "Value must be a valid number"
	}

	return ""
}

// isValidInteger checks if a string is a valid integer (for epoch timestamps)
func isValidInteger(s string) bool {
	if s == "" || s == "null" {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

// isValidFloat checks if a string is a valid float
func isValidFloat(s string) bool {
	if s == "" || s == "null" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// validateCreateGoalInput validates create goal input fields and returns error message if invalid
func validateCreateGoalInput(slug, title, goalType, gunits, goaldate, goalval, rate string) string {
	if slug == "" {
		return "Slug cannot be empty"
	}

	if title == "" {
		return "Title cannot be empty"
	}

	if goalType == "" {
		return "Goal type cannot be empty"
	}

	if gunits == "" {
		return "Goal units cannot be empty"
	}

	// Validate that exactly 2 out of 3 (goaldate, goalval, rate) are provided
	countProvided := 0

	// Validate goaldate: must be empty, "null", or a valid integer (epoch timestamp)
	if goaldate != "" && goaldate != "null" {
		if !isValidInteger(goaldate) {
			return "Goal date must be a valid epoch timestamp or 'null'"
		}
		countProvided++
	}

	// Validate goalval: must be empty, "null", or a valid number
	if goalval != "" && goalval != "null" {
		if !isValidFloat(goalval) {
			return "Goal value must be a valid number or 'null'"
		}
		countProvided++
	}

	// Validate rate: must be empty, "null", or a valid number
	if rate != "" && rate != "null" {
		if !isValidFloat(rate) {
			return "Rate must be a valid number or 'null'"
		}
		countProvided++
	}

	if countProvided != 2 {
		return "Exactly 2 out of 3 (goaldate, goalval, rate) must be provided"
	}

	return ""
}

// handleEnterKey handles Enter key press
func handleEnterKey(m model) (tea.Model, tea.Cmd) {
	if m.appModel.showCreateModal && !m.appModel.creatingGoal {
		// Clear previous error
		m.appModel.createError = ""

		// Validate input fields
		if errMsg := validateCreateGoalInput(m.appModel.createSlug, m.appModel.createTitle,
			m.appModel.createGoalType, m.appModel.createGunits, m.appModel.createGoaldate,
			m.appModel.createGoalval, m.appModel.createRate); errMsg != "" {
			m.appModel.createError = errMsg
			return m, nil
		}

		// Set creating state and submit goal creation asynchronously
		m.appModel.creatingGoal = true
		return m, createGoalCmd(m.appModel.config, m.appModel.createSlug, m.appModel.createTitle,
			m.appModel.createGoalType, m.appModel.createGunits, m.appModel.createGoaldate,
			m.appModel.createGoalval, m.appModel.createRate)
	} else if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
		// Clear previous error
		m.appModel.inputError = ""

		// Validate input fields
		if errMsg := validateDatapointInput(m.appModel.inputDate, m.appModel.inputValue); errMsg != "" {
			m.appModel.inputError = errMsg
			return m, nil
		}

		// Parse date to get timestamp
		date, _ := time.Parse("2006-01-02", m.appModel.inputDate)
		timestamp := fmt.Sprintf("%d", date.Unix())

		// Set submitting state and submit datapoint asynchronously
		m.appModel.submitting = true
		return m, submitDatapointCmd(m.appModel.config, m.appModel.modalGoal.Slug,
			timestamp, m.appModel.inputValue, m.appModel.inputComment)
	} else if !m.appModel.showModal {
		// Show goal details modal (existing functionality)
		displayGoals := m.appModel.getDisplayGoals()
		if len(displayGoals) > 0 && m.appModel.cursor < len(displayGoals) {
			m.appModel.showModal = true
			m.appModel.modalGoal = &displayGoals[m.appModel.cursor]

			// Update cursor to point to the goal in the original goals list
			// This is necessary for left/right navigation in modal
			for i, goal := range m.appModel.goals {
				if goal.Slug == displayGoals[m.appModel.cursor].Slug {
					m.appModel.cursor = i
					break
				}
			}

			// Load detailed goal information including datapoints
			return m, loadGoalDetailsCmd(m.appModel.config, m.appModel.modalGoal.Slug)
		}
	}
	return m, nil
}

// handleNavigationUp handles up arrow/k key
func handleNavigationUp(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal {
		displayGoals := m.appModel.getDisplayGoals()
		if len(displayGoals) > 0 {
			m.appModel.hasNavigated = true
			m.appModel.lastNavigationTime = time.Now()
			cols := calculateColumns(m.appModel.width)
			newCursor := m.appModel.cursor - cols
			if newCursor >= 0 {
				m.appModel.cursor = newCursor
			}
			// Keep selection visible after navigation
			updateScrollForCursor(&m, len(displayGoals))
			return m, navigationTimeoutCmd(navigationTimeout)
		}
	}
	return m, nil
}

// handleNavigationDown handles down arrow/j key
func handleNavigationDown(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal {
		displayGoals := m.appModel.getDisplayGoals()
		if len(displayGoals) > 0 {
			m.appModel.hasNavigated = true
			m.appModel.lastNavigationTime = time.Now()
			cols := calculateColumns(m.appModel.width)
			newCursor := m.appModel.cursor + cols
			if newCursor < len(displayGoals) {
				m.appModel.cursor = newCursor
			}
			// Keep selection visible after navigation
			updateScrollForCursor(&m, len(displayGoals))
			return m, navigationTimeoutCmd(navigationTimeout)
		}
	}
	return m, nil
}

// handleNavigationLeft handles left arrow/h key
func handleNavigationLeft(m model) (tea.Model, tea.Cmd) {
	if m.appModel.showModal && !m.appModel.inputMode && !m.appModel.submitting && len(m.appModel.goals) > 0 {
		// Navigate to previous goal in modal view
		if m.appModel.cursor > 0 {
			m.appModel.cursor--
			m.appModel.modalGoal = &m.appModel.goals[m.appModel.cursor]
			// Load detailed goal information including datapoints
			return m, loadGoalDetailsCmd(m.appModel.config, m.appModel.modalGoal.Slug)
		}
	} else if !m.appModel.showModal && !m.appModel.showCreateModal {
		displayGoals := m.appModel.getDisplayGoals()
		if len(displayGoals) > 0 {
			m.appModel.hasNavigated = true
			m.appModel.lastNavigationTime = time.Now()
			cols := calculateColumns(m.appModel.width)
			currentCol := m.appModel.cursor % cols
			if currentCol > 0 {
				m.appModel.cursor--
			}
			// Keep selection visible after navigation (future-proof if rows change)
			updateScrollForCursor(&m, len(displayGoals))
			return m, navigationTimeoutCmd(navigationTimeout)
		}
	}
	return m, nil
}

// handleNavigationRight handles right arrow/l key
func handleNavigationRight(m model) (tea.Model, tea.Cmd) {
	if m.appModel.showModal && !m.appModel.inputMode && !m.appModel.submitting && len(m.appModel.goals) > 0 {
		// Navigate to next goal in modal view
		if m.appModel.cursor < len(m.appModel.goals)-1 {
			m.appModel.cursor++
			m.appModel.modalGoal = &m.appModel.goals[m.appModel.cursor]
			// Load detailed goal information including datapoints
			return m, loadGoalDetailsCmd(m.appModel.config, m.appModel.modalGoal.Slug)
		}
	} else if !m.appModel.showModal && !m.appModel.showCreateModal {
		displayGoals := m.appModel.getDisplayGoals()
		if len(displayGoals) > 0 {
			m.appModel.hasNavigated = true
			m.appModel.lastNavigationTime = time.Now()
			cols := calculateColumns(m.appModel.width)
			currentCol := m.appModel.cursor % cols
			if currentCol < cols-1 && m.appModel.cursor+1 < len(displayGoals) {
				m.appModel.cursor++
			}
			// Keep selection visible after navigation (future-proof if rows change)
			updateScrollForCursor(&m, len(displayGoals))
			return m, navigationTimeoutCmd(navigationTimeout)
		}
	}
	return m, nil
}

// handleScrollUp handles page up/u key
func handleScrollUp(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal && m.appModel.scrollRow > 0 {
		m.appModel.scrollRow--
	}
	return m, nil
}

// handleScrollDown handles page down/d key
func handleScrollDown(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal {
		displayGoals := m.appModel.getDisplayGoals()
		cols := calculateColumns(m.appModel.width)
		totalRows := (len(displayGoals) + cols - 1) / cols
		maxVisibleRows := max(1, (m.appModel.height-4)/4) // Rough estimate of rows that fit
		if m.appModel.scrollRow < totalRows-maxVisibleRows {
			m.appModel.scrollRow++
		}
	}
	return m, nil
}

// handleRefresh handles the 'r' key for manual refresh
func handleRefresh(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal {
		m.appModel.loading = true
		return m, loadGoalsCmd(m.appModel.config)
	}
	return m, nil
}

// handleToggleRefresh handles the 't' key for toggling auto-refresh
func handleToggleRefresh(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal {
		m.appModel.refreshActive = !m.appModel.refreshActive
		if m.appModel.refreshActive {
			// If we just enabled auto-refresh, start the timer
			return m, refreshTickCmd()
		}
	}
	return m, nil
}

// handleEnterSearch handles the '/' key for entering search mode
func handleEnterSearch(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal && !m.appModel.searchMode {
		m.appModel.searchMode = true
		m.appModel.searchQuery = ""
	}
	return m, nil
}

// handleCreateGoal handles the 'n' key for creating a new goal
func handleCreateGoal(m model) (tea.Model, tea.Cmd) {
	if !m.appModel.showModal && !m.appModel.showCreateModal && !m.appModel.searchMode {
		m.appModel.showCreateModal = true
		m.appModel.createFocus = 0
		m.appModel.createError = ""
		// Set default values
		m.appModel.createSlug = ""
		m.appModel.createTitle = ""
		m.appModel.createGoalType = "hustler"
		m.appModel.createGunits = "units"
		m.appModel.createGoaldate = ""
		m.appModel.createGoalval = "0"
		m.appModel.createRate = "1"
	}
	return m, nil
}
