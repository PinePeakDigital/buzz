package main

import (
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
