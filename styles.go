package main

import "github.com/charmbracelet/lipgloss"

// Grid layout constants
const (
	GridMarginRight   = 0 // No horizontal margin - borders will touch
	GridMarginBottom  = 0 // No vertical margin - borders will touch
	PaddingVertical   = 0
	PaddingHorizontal = 1
)

// CreateModalStyle returns the style for the goal details modal
func CreateModalStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("4")).
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("15")).
		Padding(1, 2).
		Margin(1, 2)
}

// CreateOverlayStyle returns the semi-transparent overlay style
func CreateOverlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("8")).
		Foreground(lipgloss.Color("0"))
}

