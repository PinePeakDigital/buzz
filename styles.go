package main

import "github.com/charmbracelet/lipgloss"

// Grid layout constants
const (
	GridMarginRight    = 0  // No horizontal margin - borders will touch
	GridMarginBottom   = 0  // No vertical margin - borders will touch
	PaddingVertical    = 0
	PaddingHorizontal  = 1
)

// CreateGridStyles returns the styled grid cell styles
func CreateGridStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"red": lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("1")).
			Foreground(lipgloss.Color("1")).
			Padding(PaddingVertical, PaddingHorizontal).
			MarginRight(GridMarginRight).
			MarginBottom(GridMarginBottom),
		
		"orange": lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("208")).
			Foreground(lipgloss.Color("208")).
			Padding(PaddingVertical, PaddingHorizontal).
			MarginRight(GridMarginRight).
			MarginBottom(GridMarginBottom),
		
		"blue": lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("4")).
			Foreground(lipgloss.Color("4")).
			Padding(PaddingVertical, PaddingHorizontal).
			MarginRight(GridMarginRight).
			MarginBottom(GridMarginBottom),
		
		"green": lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("2")).
			Foreground(lipgloss.Color("2")).
			Padding(PaddingVertical, PaddingHorizontal).
			MarginRight(GridMarginRight).
			MarginBottom(GridMarginBottom),
		
		"gray": lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("8")).
			Foreground(lipgloss.Color("8")).
			Padding(PaddingVertical, PaddingHorizontal).
			MarginRight(GridMarginRight).
			MarginBottom(GridMarginBottom),
	}
}

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