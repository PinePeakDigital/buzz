package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// RenderGrid renders the goals grid based on the app model
func RenderGrid(goals []Goal, width, height, scrollRow int) string {
	if len(goals) == 0 {
		return "No goals found.\n\nPress q to quit.\n"
	}

	// The header
	s := "Beeminder Goals\n\n"

	// Get grid styles
	styles := CreateGridStyles()

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
			style, exists := styles[color]
			if !exists {
				style = styles["gray"]
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
	
	return fmt.Sprintf("\nPress q to quit%s%s\n", scrollInfo, refreshInfo)
}