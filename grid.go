package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderGrid renders the goals grid based on the app model
func RenderGrid(goals []Goal, width, height, scrollRow, cursor int, hasNavigated bool) string {
	if len(goals) == 0 {
		return "No goals found.\n\nPress q to quit.\n"
	}

	// The header
	s := "Beeminder Goals\n\n"

	// Get grid styles
	styles := CreateGridStyles()
	highlightedStyles := CreateHighlightedGridStyles()

	// Calculate grid dimensions based on terminal width
	cols := calculateColumns(width)
	totalRows := (len(goals) + cols - 1) / cols
	
	// Calculate visible rows based on terminal height
	// Each cell is roughly 4 lines high (3 lines content + 1 line spacing)
	maxVisibleRows := max(1, (height-4)/4) // -4 for header and footer
	
	// Calculate which rows to display
	startRow := scrollRow
	endRow := min(totalRows, startRow+maxVisibleRows)

	// Build grid - only render visible rows
	for row := startRow; row < endRow; row++ {
		var rowCells []string
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(goals) {
				break
			}

			goal := goals[idx]

			// Get color based on buffer
			color := GetBufferColor(goal.Safebuf)
			
			// Choose style based on whether this goal is selected and user has navigated
			var style lipgloss.Style
			var exists bool
			if idx == cursor && hasNavigated {
				// Use highlighted style for selected goal (only after navigation)
				style, exists = highlightedStyles[color]
				if !exists {
					style = highlightedStyles["gray"]
				}
			} else {
				// Use normal style for non-selected goals or when not navigated yet
				style, exists = styles[color]
				if !exists {
					style = styles["gray"]
				}
			}

			// Format goal display
			display := fmt.Sprintf("%s\n$%.0f | %s",
				truncateString(goal.Slug, 16),
				goal.Pledge,
				FormatDueDate(goal.Losedate))

			cell := style.Render(display)
			rowCells = append(rowCells, cell)
		}
		s += lipgloss.JoinHorizontal(lipgloss.Top, rowCells...)
		s += "\n"
	}

	return s
}

// RenderFooter renders the footer with scroll and refresh information
func RenderFooter(goals []Goal, width, height, scrollRow int, refreshActive bool) string {
	// The footer with scroll information
	footerCols := calculateColumns(width)
	footerTotalRows := (len(goals) + footerCols - 1) / footerCols
	footerMaxVisibleRows := max(1, (height-4)/4)
	
	scrollInfo := ""
	if footerTotalRows > footerMaxVisibleRows {
		scrollInfo = fmt.Sprintf(" | Scroll: %d/%d (u/d or pgup/pgdown)", 
			scrollRow+1, max(1, footerTotalRows-footerMaxVisibleRows+1))
	}
	
	// Refresh status
	refreshStatus := "OFF"
	if refreshActive {
		refreshStatus = "ON"
	}
	refreshInfo := fmt.Sprintf(" | Auto-refresh: %s (t to toggle, r to refresh now)", refreshStatus)
	
	// Build the full footer text
	footerText := fmt.Sprintf("Press q to quit%s%s | Arrow keys to navigate, Enter for details", scrollInfo, refreshInfo)
	
	// If the footer is too wide, wrap it
	if len(footerText) > width {
		// Split into multiple lines based on available width
		lines := wrapText(footerText, width)
		return "\n" + strings.Join(lines, "\n") + "\n"
	}
	
	return fmt.Sprintf("\n%s\n", footerText)
}

// RenderModal renders a modal with detailed goal information and data input form
func RenderModal(goal *Goal, width, height int, inputDate, inputValue, inputComment string, inputFocus int, inputMode bool, inputError string, submitting bool) string {
	if goal == nil {
		return ""
	}

	modalStyle := CreateModalStyle()
	
	// Calculate modal dimensions (80% of screen width, auto height)
	modalWidth := width * 8 / 10
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	// Goal details content
	content := fmt.Sprintf("Goal Details\n\n"+
		"Slug: %s\n"+
		"Title: %s\n"+
		"Pledge: $%.2f\n"+
		"Safe Buffer: %d days\n"+
		"Due Date: %s\n"+
		"Buffer Color: %s",
		goal.Slug,
		goal.Title,
		goal.Pledge,
		goal.Safebuf,
		FormatDueDate(goal.Losedate),
		GetBufferColor(goal.Safebuf))

	// Data input form
	var formContent string
	if inputMode {
		if submitting {
			// Show submitting state
			formContent = fmt.Sprintf("\n\n--- Add Datapoint ---\nDate: %s\nValue: %s\nComment: %s\n\n%s", 
				inputDate, inputValue, inputComment,
				lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Submitting datapoint..."))
		} else {
			// Create input fields with focus highlighting
			dateField := inputDate
			valueField := inputValue
			commentField := inputComment
			
			if inputFocus == 0 {
				dateField = lipgloss.NewStyle().Background(lipgloss.Color("4")).Render(dateField)
			}
			if inputFocus == 1 {
				valueField = lipgloss.NewStyle().Background(lipgloss.Color("4")).Render(valueField)
			}
			if inputFocus == 2 {
				commentField = lipgloss.NewStyle().Background(lipgloss.Color("4")).Render(commentField)
			}
			
			errorMsg := ""
			if inputError != "" {
				errorMsg = fmt.Sprintf("\n%s", lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("Error: "+inputError))
			}
			
			formContent = fmt.Sprintf("\n\n--- Add Datapoint ---\nDate: %s\nValue: %s\nComment: %s%s\n\nTab/Shift+Tab: Navigate • Enter: Submit • Esc: Cancel", 
				dateField, valueField, commentField, errorMsg)
		}
	} else {
		formContent = "\n\nPress 'a' to add datapoint • Press ESC to close"
	}
	
	content += formContent

	// Apply width constraint to content
	styledContent := modalStyle.Width(modalWidth).Render(content)
	
	// Center the modal horizontally
	leftPadding := (width - modalWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}
	
	// Center the modal vertically (approximately)
	topPadding := height / 4
	if topPadding < 1 {
		topPadding = 1
	}
	
	// Add vertical spacing
	verticalPadding := ""
	for i := 0; i < topPadding; i++ {
		verticalPadding += "\n"
	}
	
	// Add horizontal centering
	centeredModal := ""
	for _, line := range []string{styledContent} {
		padding := ""
		for i := 0; i < leftPadding; i++ {
			padding += " "
		}
		centeredModal += padding + line
	}
	
	return verticalPadding + centeredModal
}