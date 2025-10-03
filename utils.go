package main

import "strings"

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// calculateColumns determines the optimal number of columns based on terminal width
func calculateColumns(width int) int {
	// Each cell needs approximately:
	// - 16 chars for content (truncateString maxLen)
	// - 2 chars for left/right borders
	// - 2 chars for horizontal padding
	// Total: ~20 chars per cell
	const minCellWidth = 20
	const minCols = 1
	
	if width < minCellWidth {
		return minCols
	}
	
	cols := width / minCellWidth
	return max(minCols, cols)
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		// Pad with spaces to ensure consistent width
		return s + strings.Repeat(" ", maxLen-len(s))
	}
	return s[:maxLen-3] + "..."
}