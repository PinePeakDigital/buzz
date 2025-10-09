package main

import (
	"fmt"
	"strings"
)

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
	// - 18 chars for content (inner width)
	// - 2 chars for left/right borders
	// - 2 chars for horizontal padding
	// Total: ~22 chars per cell
	const minCellWidth = 22
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

// formatGoalFirstLine formats the first line of a goal cell with slug and stakes
// Format: "slug         $5" (exactly 18 characters)
func formatGoalFirstLine(slug string, pledge float64) string {
	const width = 18
	
	// Format the pledge part (e.g., "$5" or "$10")
	pledgeStr := fmt.Sprintf("$%.0f", pledge)
	
	// Calculate space available for slug (need at least 1 space between slug and pledge)
	availableForSlug := width - len(pledgeStr) - 1
	
	if availableForSlug < 1 {
		// If pledge is too long, just show ellipsis and pledge
		return "..." + strings.Repeat(" ", width-3-len(pledgeStr)) + pledgeStr
	}
	
	// Truncate slug if necessary
	var slugPart string
	if len(slug) <= availableForSlug {
		slugPart = slug
	} else {
		// Need to truncate slug with ellipsis
		if availableForSlug < 3 {
			slugPart = strings.Repeat(".", min(availableForSlug, 3))
		} else {
			slugPart = slug[:availableForSlug-3] + "..."
		}
	}
	
	// Calculate spaces needed to fill the width
	spacesNeeded := width - len(slugPart) - len(pledgeStr)
	if spacesNeeded < 0 {
		spacesNeeded = 0
	}
	
	return slugPart + strings.Repeat(" ", spacesNeeded) + pledgeStr
}

// formatGoalSecondLine formats the second line of a goal cell with delta value and timeframe
// Format: "deltaValue in timeframe" (exactly 18 characters)
func formatGoalSecondLine(deltaValue string, timeframe string) string {
	const width = 18
	
	// Build the full string
	fullStr := deltaValue + " in " + timeframe
	
	if len(fullStr) <= width {
		// Pad with spaces to reach exact width
		return fullStr + strings.Repeat(" ", width-len(fullStr))
	}
	
	// Need to truncate with ellipsis
	return fullStr[:width-3] + "..."
}

// wrapText wraps text to fit within the specified width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var lines []string
	var currentLine strings.Builder

	for i, word := range words {
		// If this is the first word, add it directly
		if i == 0 {
			currentLine.WriteString(word)
			continue
		}

		// Check if adding the next word would exceed the width
		if currentLine.Len()+1+len(word) > width {
			// Start a new line
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		} else {
			// Add word to current line
			currentLine.WriteString(" " + word)
		}
	}

	// Add the last line if it has content
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// fuzzyMatch returns true if the pattern matches the text using fuzzy search
// Pattern characters must appear in order in the text (case-insensitive)
func fuzzyMatch(pattern, text string) bool {
	if pattern == "" {
		return true
	}

	// Convert to lowercase for case-insensitive matching
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	patternIdx := 0
	for _, char := range text {
		if patternIdx < len(pattern) && char == rune(pattern[patternIdx]) {
			patternIdx++
		}
		if patternIdx == len(pattern) {
			return true
		}
	}

	return patternIdx == len(pattern)
}
