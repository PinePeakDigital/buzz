package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// handleSearchInput handles text input in search mode
func handleSearchInput(m model, msg tea.KeyMsg) (model, bool) {
	if m.appModel.searchActive && m.appModel.mode == modeBrowse {
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

// handleCreateModalInput handles text input in create goal modal
func handleCreateModalInput(m model, msg tea.KeyMsg) (model, bool) {
	if m.appModel.mode != modeCreateGoal || m.appModel.createGoal.creating {
		return m, false
	}
	if len(msg.Runes) != 1 {
		return m, false
	}
	handled := m.appModel.createGoal.handleRune(msg.Runes[0])
	return m, handled
}

// handleDatapointInput handles text input in datapoint input mode
func handleDatapointInput(m model, msg tea.KeyMsg) (model, bool) {
	// Handle text input in input mode
	// This ensures that single-character command keys (like 't', 'r', 'd', etc.)
	// can still be typed in comment fields
	if m.appModel.mode == modeDatapointInput && !m.appModel.datapoint.submitting {
		if len(msg.Runes) == 1 {
			handled := m.appModel.datapoint.handleRune(msg.Runes[0])
			return m, handled
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

// handleEscapeKey handles the Escape key press as a "back out one level" ladder.
// Foreground modes are unwound before the search layer, so Esc on a goal-detail
// modal opened over a search closes the modal while keeping the search.
func handleEscapeKey(m model) (tea.Model, tea.Cmd) {
	switch {
	case m.appModel.mode == modeDatapointInput:
		// Cancel datapoint entry, back to goal detail
		m.appModel.exitDatapointInput()
	case m.appModel.mode == modeCreateGoal:
		// Close create goal form
		m.appModel.closeCreateGoal()
	case m.appModel.mode == modeGoalDetail:
		// Close goal detail modal (search, if any, stays active underneath)
		m.appModel.closeModal()
	case m.appModel.searchActive:
		// Exit the search filter layer
		m.appModel.exitSearch()
	default:
		return m, tea.Quit
	}
	return m, nil
}

// handleAddDatapoint enters input mode for adding a datapoint
func handleAddDatapoint(m model) (tea.Model, tea.Cmd) {
	if m.appModel.mode == modeGoalDetail {
		// Try to get the last datapoint value, default to "1" if it fails
		defaultValue := "1"
		if lastValue, err := m.appModel.client.GetLastDatapointValue(m.appModel.ctx, m.appModel.modalGoal.Slug); err == nil && lastValue != 0 {
			defaultValue = fmt.Sprintf("%.1f", lastValue)
		}
		m.appModel.startDatapointInput(newDatapointForm(defaultValue))
	}
	return m, nil
}

// handleTabKey handles Tab and Shift+Tab navigation
func handleTabKey(m model, reverse bool) (tea.Model, tea.Cmd) {
	if m.appModel.mode == modeCreateGoal && !m.appModel.createGoal.creating {
		m.appModel.createGoal.tab(reverse)
	} else if m.appModel.mode == modeDatapointInput && !m.appModel.datapoint.submitting {
		m.appModel.datapoint.tab(reverse)
	}
	return m, nil
}

// handleBackspace handles Backspace key
func handleBackspace(m model) (tea.Model, tea.Cmd) {
	if m.appModel.mode == modeCreateGoal && !m.appModel.createGoal.creating {
		m.appModel.createGoal.backspace()
	} else if m.appModel.searchActive && m.appModel.mode == modeBrowse {
		// Remove last character from search query. Trim a whole rune rather
		// than a byte so backspacing a multibyte character (search accepts any
		// printable Unicode) leaves valid UTF-8.
		if len(m.appModel.searchQuery) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.appModel.searchQuery)
			m.appModel.searchQuery = m.appModel.searchQuery[:len(m.appModel.searchQuery)-size]
			// Reset cursor and scroll when search query changes
			m.appModel.cursor = 0
			m.appModel.scrollRow = 0
			m.appModel.hasNavigated = false
		}
	} else if m.appModel.mode == modeDatapointInput && !m.appModel.datapoint.submitting {
		m.appModel.datapoint.backspace()
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

	// Parse and validate date. Interpret the calendar date in local time so the
	// comparison below against the local time.Now() is timezone-consistent
	// (parsing without a location would assume UTC and shift the boundary).
	date, err := time.ParseInLocation("2006-01-02", inputDate, time.Local)
	if err != nil {
		return "Invalid date format (use YYYY-MM-DD)"
	}

	// Validate that date is not in the future beyond today
	if date.After(time.Now().AddDate(0, 0, 1)) {
		return "Date cannot be more than 1 day in the future"
	}

	// Parse and validate value (must be a valid, finite number). ParseFloat
	// accepts "NaN"/"Inf"/"+Inf"/"-Inf"/"Infinity"/"+Infinity"/"-Infinity", so
	// reject non-finite results explicitly.
	if v, err := strconv.ParseFloat(inputValue, 64); err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
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

// isValidFloat checks if a string is a valid, finite float. ParseFloat accepts
// "NaN"/"Inf"/"+Inf"/"-Inf"/"Infinity"/"+Infinity"/"-Infinity", so reject
// non-finite results explicitly.
func isValidFloat(s string) bool {
	if s == "" || s == "null" {
		return false
	}
	v, err := strconv.ParseFloat(s, 64)
	return err == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
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
	if m.appModel.mode == modeCreateGoal && !m.appModel.createGoal.creating {
		// Clear previous error
		m.appModel.createGoal.err = ""

		// Validate input fields
		if errMsg := m.appModel.createGoal.validate(); errMsg != "" {
			m.appModel.createGoal.err = errMsg
			return m, nil
		}

		// Set creating state and submit goal creation asynchronously
		m.appModel.createGoal.creating = true
		return m, createGoalCmd(m.appModel.ctx, m.appModel.client, m.appModel.createGoal.slug(), m.appModel.createGoal.title(),
			m.appModel.createGoal.goalType(), m.appModel.createGoal.gunits(), m.appModel.createGoal.goaldate(),
			m.appModel.createGoal.goalval(), m.appModel.createGoal.rate())
	} else if m.appModel.mode == modeDatapointInput && !m.appModel.datapoint.submitting {
		// Clear previous error
		m.appModel.datapoint.err = ""

		// Validate input fields
		if errMsg := m.appModel.datapoint.validate(); errMsg != "" {
			m.appModel.datapoint.err = errMsg
			return m, nil
		}

		// Parse date to get timestamp. Interpret the entered calendar date in
		// local time (matching validateDatapointInput) so the datapoint lands on
		// the day the user intended rather than being shifted by the UTC offset.
		date, _ := time.ParseInLocation("2006-01-02", m.appModel.datapoint.date(), time.Local)
		timestamp := fmt.Sprintf("%d", date.Unix())

		// Set submitting state and submit datapoint asynchronously
		m.appModel.datapoint.submitting = true
		return m, submitDatapointCmd(m.appModel.ctx, m.appModel.client, m.appModel.modalGoal.Slug,
			timestamp, m.appModel.datapoint.value(), m.appModel.datapoint.comment())
	} else if m.appModel.mode == modeBrowse {
		// Show goal details modal (existing functionality)
		displayGoals := m.appModel.getDisplayGoals()
		if len(displayGoals) > 0 && m.appModel.cursor < len(displayGoals) {
			selected := &displayGoals[m.appModel.cursor]
			m.appModel.openGoalDetail(selected)

			// Update cursor to point to the goal in the original goals list
			// This is necessary for left/right navigation in modal
			for i, goal := range m.appModel.goals {
				if goal.Slug == selected.Slug {
					m.appModel.cursor = i
					break
				}
			}

			// Load detailed goal information including datapoints
			return m, loadGoalDetailsCmd(m.appModel.ctx, m.appModel.client, m.appModel.modalGoal.Slug)
		}
	}
	return m, nil
}

// handleNavigationUp handles up arrow/k key
func handleNavigationUp(m model) (tea.Model, tea.Cmd) {
	if m.appModel.mode == modeBrowse {
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
	if m.appModel.mode == modeBrowse {
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
	if m.appModel.mode == modeGoalDetail && len(m.appModel.goals) > 0 {
		// Navigate to previous goal in modal view
		if m.appModel.cursor > 0 {
			m.appModel.cursor--
			m.appModel.openGoalDetail(&m.appModel.goals[m.appModel.cursor])
			// Load detailed goal information including datapoints
			return m, loadGoalDetailsCmd(m.appModel.ctx, m.appModel.client, m.appModel.modalGoal.Slug)
		}
	} else if m.appModel.mode == modeBrowse {
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
	if m.appModel.mode == modeGoalDetail && len(m.appModel.goals) > 0 {
		// Navigate to next goal in modal view
		if m.appModel.cursor < len(m.appModel.goals)-1 {
			m.appModel.cursor++
			m.appModel.openGoalDetail(&m.appModel.goals[m.appModel.cursor])
			// Load detailed goal information including datapoints
			return m, loadGoalDetailsCmd(m.appModel.ctx, m.appModel.client, m.appModel.modalGoal.Slug)
		}
	} else if m.appModel.mode == modeBrowse {
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
	if m.appModel.mode == modeBrowse && m.appModel.scrollRow > 0 {
		m.appModel.scrollRow--
	}
	return m, nil
}

// handleScrollDown handles page down/d key
func handleScrollDown(m model) (tea.Model, tea.Cmd) {
	if m.appModel.mode == modeBrowse {
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
	if m.appModel.mode == modeBrowse {
		m.appModel.loading = true
		return m, loadGoalsCmd(m.appModel.ctx, m.appModel.client)
	}
	return m, nil
}

// handleToggleRefresh handles the 't' key for toggling auto-refresh
func handleToggleRefresh(m model) (tea.Model, tea.Cmd) {
	if m.appModel.mode == modeBrowse {
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
	if m.appModel.mode == modeBrowse && !m.appModel.searchActive {
		m.appModel.enterSearch()
	}
	return m, nil
}

// handleCreateGoal handles the 'n' key for creating a new goal
func handleCreateGoal(m model) (tea.Model, tea.Cmd) {
	if m.appModel.mode == modeBrowse && !m.appModel.searchActive {
		m.appModel.openCreateGoal()
	}
	return m, nil
}

// handleMouseClick handles mouse click events on the grid
func handleMouseClick(m model, msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	displayGoals := m.appModel.getDisplayGoals()
	if len(displayGoals) == 0 {
		return m, nil
	}

	// Calculate which goal was clicked based on coordinates
	// Header is 2 lines (title + empty line), so content starts at line 2 (0-indexed)
	headerHeight := 2
	clickRow := msg.Y - headerHeight

	// Each grid cell is 4 lines high (3 lines content + 1 line spacing)
	cellHeight := 4
	if clickRow < 0 {
		// Clicked on header area
		return m, nil
	}
	gridRow := clickRow / cellHeight

	// Calculate column based on terminal width
	cols := calculateColumns(m.appModel.width)
	if cols < 1 {
		cols = 1
	}
	// Approximate cell width
	cellWidth := m.appModel.width / cols
	if cellWidth < 1 {
		cellWidth = 1
	}
	gridCol := msg.X / cellWidth

	// Calculate the goal index accounting for scroll position
	goalIndex := (m.appModel.scrollRow+gridRow)*cols + gridCol

	// Validate the index is within bounds
	if goalIndex >= 0 && goalIndex < len(displayGoals) {
		m.appModel.hasNavigated = true
		m.appModel.lastNavigationTime = time.Now()

		// Open the modal immediately (same as pressing Enter)
		m.appModel.openGoalDetail(&displayGoals[goalIndex])

		// Update cursor to point to goal in original list (for left/right navigation)
		for i, goal := range m.appModel.goals {
			if goal.Slug == displayGoals[goalIndex].Slug {
				m.appModel.cursor = i
				break
			}
		}

		// Load detailed goal information
		return m, tea.Batch(
			loadGoalDetailsCmd(m.appModel.ctx, m.appModel.client, m.appModel.modalGoal.Slug),
			navigationTimeoutCmd(navigationTimeout),
		)
	}

	return m, nil
}
