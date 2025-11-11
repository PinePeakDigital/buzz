package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// reviewModel holds the state for the review command
type reviewModel struct {
	goals   []Goal
	config  *Config
	current int    // current goal index
	width   int    // terminal width
	height  int    // terminal height
	err     string // error message to display
}

// initialReviewModel creates a new review model
func initialReviewModel(goals []Goal, config *Config) reviewModel {
	return reviewModel{
		goals:   goals,
		config:  config,
		current: 0,
	}
}

func (m reviewModel) Init() tea.Cmd {
	return nil
}

func (m reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "right", "l", "n", "j":
			// Next goal
			if m.current < len(m.goals)-1 {
				m.current++
			}
			m.err = ""
			return m, nil

		case "left", "h", "p", "k":
			// Previous goal
			if m.current > 0 {
				m.current--
			}
			m.err = ""
			return m, nil

		case "o", "enter":
			// Open current goal in browser
			if m.current < len(m.goals) {
				goal := m.goals[m.current]
				if err := openBrowser(m.config, goal.Slug); err != nil {
					m.err = fmt.Sprintf("Failed to open browser: %v", err)
				} else {
					m.err = "" // Clear any previous error
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m reviewModel) View() string {
	if len(m.goals) == 0 {
		return "No goals to review.\n\nPress q to quit."
	}

	goal := m.goals[m.current]

	// Create the goal details view
	var view string

	// Title section with counter and status indicator
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1)

	counterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	// Add colored status indicator based on buffer
	color := GetBufferColor(goal.Safebuf)
	var statusColor lipgloss.Color
	var statusSymbol string
	switch color {
	case "red":
		statusColor = lipgloss.Color("9")
		statusSymbol = "●"
	case "orange":
		statusColor = lipgloss.Color("214")
		statusSymbol = "●"
	case "blue":
		statusColor = lipgloss.Color("12")
		statusSymbol = "●"
	case "green":
		statusColor = lipgloss.Color("10")
		statusSymbol = "●"
	default:
		statusColor = lipgloss.Color("241")
		statusSymbol = "●"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(statusColor).
		Padding(0, 1, 0, 0)

	view += statusStyle.Render(statusSymbol) + titleStyle.Render(fmt.Sprintf("Goal: %s", goal.Slug)) + "\n"
	view += counterStyle.Render(fmt.Sprintf("Goal %d of %d", m.current+1, len(m.goals))) + "\n\n"

	// Goal details section
	detailStyle := lipgloss.NewStyle().
		Padding(0, 2)

	details := ""
	details += fmt.Sprintf("Title:       %s\n", goal.Title)
	details += fmt.Sprintf("Limsum:      %s\n", goal.Limsum)
	deadlineTime := time.Unix(goal.Losedate, 0)
	details += fmt.Sprintf("Deadline:    %s\n", deadlineTime.Format("Mon Jan 2, 2006 at 3:04 PM MST"))
	details += fmt.Sprintf("Due time:    %s\n", formatDueTime(goal.Deadline))
	details += fmt.Sprintf("Pledge:      $%.2f\n", goal.Pledge)

	// Display current rate (n / unit)
	if goal.Rate != nil && goal.Runits != "" {
		rateStr := formatRate(*goal.Rate, goal.Runits, goal.Gunits)
		details += fmt.Sprintf("Rate:        %s\n", rateStr)
	}

	details += fmt.Sprintf("Autodata:    %s\n", goal.Autodata)

	// Display autoratchet only if set (not nil)
	if goal.Autoratchet != nil {
		details += fmt.Sprintf("Autoratchet: %.0f\n", *goal.Autoratchet)
	}

	view += detailStyle.Render(details) + "\n"

	// Error message section (if any)
	if m.err != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Padding(0, 2)
		view += errorStyle.Render(fmt.Sprintf("⚠ %s", m.err)) + "\n"
	}

	// Help section
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(1, 2)

	help := "Navigation: ← → (or h l, or j k, or p n)  |  Open in browser: o or Enter  |  Quit: q or Esc"
	view += helpStyle.Render(help)

	return view
}

// openBrowser opens the goal page in the default browser
func openBrowser(config *Config, goalSlug string) error {
	baseURL := getBaseURL(config)
	goalURL := fmt.Sprintf("%s/%s/%s", baseURL, url.PathEscape(config.Username), url.PathEscape(goalSlug))

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", goalURL)
	case "linux":
		cmd = exec.Command("xdg-open", goalURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", goalURL)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// formatRate formats the rate with the appropriate time unit and goal units
func formatRate(rate float64, runits, gunits string) string {
	unitName := ""
	switch runits {
	case "y":
		unitName = "year"
	case "m":
		unitName = "month"
	case "w":
		unitName = "week"
	case "d":
		unitName = "day"
	case "h":
		unitName = "hour"
	default:
		unitName = runits
	}

	if gunits != "" {
		return fmt.Sprintf("%g %s / %s", rate, gunits, unitName)
	}
	return fmt.Sprintf("%g/%s", rate, unitName)
}

// formatDueTime formats the deadline offset (seconds from midnight) as a time string
// Negative offset means before midnight, positive means after midnight
func formatDueTime(deadlineOffset int) string {
	// Calculate hours and minutes from seconds
	hours := deadlineOffset / 3600
	minutes := (deadlineOffset % 3600) / 60

	// Handle negative offsets (before midnight)
	if deadlineOffset < 0 {
		hours = 24 + hours // Convert to hours before midnight
		if minutes != 0 {
			minutes = 60 + minutes
			hours--
		}
	}

	// Create a time at the specified hour and minute
	t := time.Date(0, 1, 1, hours, minutes, 0, 0, time.UTC)
	return t.Format("3:04 PM")
}
