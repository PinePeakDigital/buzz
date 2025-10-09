package main

import (
	"fmt"
	"strconv"
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
		// If pledge is too long, clamp spaces to avoid negative Repeat count
		spaces := width - 3 - len(pledgeStr)
		if spaces < 0 {
			// Fallback: truncate pledge to fit the line
			return truncateString(pledgeStr, width)
		}
		return "..." + strings.Repeat(" ", spaces) + pledgeStr
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

// isTimeFormat checks if a string is in time format (HH:MM or HH:MM:SS)
// Returns true for formats like "1:30", "00:05", "2:45:30", etc.
func isTimeFormat(s string) bool {
	s = strings.TrimPrefix(s, "+")
	s = strings.TrimPrefix(s, "-")
	return strings.Contains(s, ":")
}

// timeToDecimalHours converts a time string (HH:MM or HH:MM:SS) to decimal hours
// Examples: "1:30" -> 1.5, "00:05" -> 0.083333, "2:45:30" -> 2.758333
// Returns the decimal hours and true if successful, 0 and false if the format is invalid
func timeToDecimalHours(timeStr string) (float64, bool) {
	// Handle negative times
	isNegative := false
	if strings.HasPrefix(timeStr, "-") {
		isNegative = true
		timeStr = strings.TrimPrefix(timeStr, "-")
	}
	// Remove leading + if present
	timeStr = strings.TrimPrefix(timeStr, "+")

	// Split by colon
	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}

	// Parse hours
	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, false
	}

	// Parse minutes
	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, false
	}

	// Parse seconds if present
	seconds := 0.0
	if len(parts) == 3 {
		seconds, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, false
		}
	}

	// Convert to decimal hours
	decimalHours := hours + (minutes / 60.0) + (seconds / 3600.0)

	if isNegative {
		decimalHours = -decimalHours
	}

	return decimalHours, true
}
