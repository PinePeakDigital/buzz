package main

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleReviewCommand launches an interactive review of all goals
func handleReviewCommand() {
	// Load config
	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)

	// Fetch goals with their recent datapoints
	goals, err := client.FetchGoalsWithDatapoints(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch goals: %s\n", redactError(err))
		os.Exit(1)
	}

	if len(goals) == 0 {
		fmt.Println("No goals found.")
		return
	}

	// Sort goals alphabetically by slug as specified
	SortGoalsBySlug(goals)

	// Launch the interactive review TUI
	p := tea.NewProgram(initialReviewModel(goals, config), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
		os.Exit(1)
	}
}

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

	// Colored status indicator. Uses bright-palette variants (9/214/12/10/241)
	// rather than the main urgency colours so the dot stands out next to the
	// title text, which already uses the main palette.
	var statusColor lipgloss.Color
	statusSymbol := "●"
	switch UrgencyFor(goal.Safebuf) {
	case UrgencyOverdue:
		statusColor = lipgloss.Color("9")
	case UrgencyDueToday:
		statusColor = lipgloss.Color("214")
	case UrgencyDueTomorrow:
		statusColor = lipgloss.Color("12")
	case UrgencyThisWeek:
		statusColor = lipgloss.Color("10")
	default:
		statusColor = lipgloss.Color("241")
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(statusColor).
		Padding(0, 1, 0, 0)

	view += statusStyle.Render(statusSymbol) + titleStyle.Render(fmt.Sprintf("Goal: %s", goal.Slug)) + "\n"
	view += counterStyle.Render(fmt.Sprintf("Goal %d of %d", m.current+1, len(m.goals))) + "\n\n"

	// Goal details section
	detailStyle := lipgloss.NewStyle().
		Padding(0, 2)

	details := formatGoalDetails(&goal, m.config)

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

	value := formatRateValue(rate)
	if gunits != "" {
		return fmt.Sprintf("%s %s / %s", value, gunits, unitName)
	}
	return fmt.Sprintf("%s/%s", value, unitName)
}

// rateDisplayDecimals caps how many decimal places a rate is shown with. The
// Beeminder API returns rates at full float precision (e.g.
// 0.21317778888888886), which is noise to a human reading `buzz view`.
const rateDisplayDecimals = 4

// formatRateValue renders a rate as a clean decimal string: rounded to
// rateDisplayDecimals places, with trailing zeros trimmed and no scientific
// notation (so large whole-number rates like 100000 stay readable).
func formatRateValue(rate float64) string {
	scale := math.Pow(10, rateDisplayDecimals)
	rounded := math.Round(rate*scale) / scale
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// formatRecentDatapoints formats up to 5 of the most recent datapoints for
// display, most recent first, in aligned date / value / comment columns.
// The Beeminder API returns datapoints oldest-first, so the most recent ones
// are at the end of the slice.
func formatRecentDatapoints(datapoints []Datapoint) string {
	if len(datapoints) == 0 {
		return ""
	}

	count := 5
	if len(datapoints) < count {
		count = len(datapoints)
	}

	type row struct {
		date    string
		value   string
		comment string
	}

	rows := make([]row, 0, count)
	maxValueLen := 0
	for i := len(datapoints) - 1; i >= len(datapoints)-count; i-- {
		dp := datapoints[i]
		var dateStr string
		if len(dp.Daystamp) == 8 {
			// Daystamp avoids timezone drift: "20241217" -> "2024-12-17".
			dateStr = dp.Daystamp[:4] + "-" + dp.Daystamp[4:6] + "-" + dp.Daystamp[6:8]
		} else {
			dateStr = time.Unix(dp.Timestamp, 0).UTC().Format("2006-01-02")
		}
		valueStr := fmt.Sprintf("%.6g", dp.Value)
		if len(valueStr) > maxValueLen {
			maxValueLen = len(valueStr)
		}
		rows = append(rows, row{date: dateStr, value: valueStr, comment: dp.Comment})
	}

	output := "\nRecent datapoints:\n"
	for _, r := range rows {
		if r.comment != "" {
			output += fmt.Sprintf("  %s   %-*s   %s\n", r.date, maxValueLen, r.value, r.comment)
		} else {
			output += fmt.Sprintf("  %s   %s\n", r.date, r.value)
		}
	}

	return output
}

// formatGoalDetails formats the goal details in a consistent way for both view and review commands
func formatGoalDetails(goal *Goal, config *Config) string {
	var details string

	// Display title only if not empty
	if goal.Title != "" {
		details += fmt.Sprintf("Title:       %s\n", goal.Title)
	}

	// Display limsum with color coding based on urgency
	style := UrgencyFor(goal.Safebuf).TextStyle()
	coloredLimsum := style.Render(goal.Limsum)
	details += fmt.Sprintf("Limsum:      %s\n", coloredLimsum)

	// Display deadline (formatted timestamp) with same color coding
	deadlineTime := time.Unix(goal.Losedate, 0)
	deadlineStr := deadlineTime.Format("Mon Jan 2, 2006 at 3:04 PM MST")
	coloredDeadline := style.Render(deadlineStr)
	details += fmt.Sprintf("Deadline:    %s\n", coloredDeadline)

	// Display due time (time of day)
	details += fmt.Sprintf("Due time:    %s\n", formatDueTime(goal.Deadline))

	pledgeDisplay := fmt.Sprintf("$%.2f", goal.Pledge)
	if goal.PledgeCap != nil && *goal.PledgeCap > 0 && *goal.PledgeCap != goal.Pledge {
		pledgeDisplay = fmt.Sprintf("$%.2f / $%.2f", goal.Pledge, *goal.PledgeCap)
	}
	details += fmt.Sprintf("Pledge:      %s\n", pledgeDisplay)

	// Display rate (n / unit). When the current rate differs from the end
	// rate (a non-flat road), show both so the user sees what they're held to
	// today versus where the goal is heading.
	if goal.Rate != nil && goal.Runits != "" {
		rateStr := formatRate(*goal.Rate, goal.Runits, goal.Gunits)
		if cur := goal.CurrentRate(); cur != nil && *cur != *goal.Rate {
			rateStr = fmt.Sprintf("%s (current), %s (end)",
				formatRate(*cur, goal.Runits, goal.Gunits),
				formatRateValue(*goal.Rate))
		}
		details += fmt.Sprintf("Rate:        %s\n", rateStr)
	}

	// Display autodata only if not empty
	if goal.Autodata != "" {
		details += fmt.Sprintf("Autodata:    %s\n", goal.Autodata)
	}

	// Display autoratchet only if set (not nil)
	if goal.Autoratchet != nil {
		details += fmt.Sprintf("Autoratchet: %.0f\n", *goal.Autoratchet)
	}

	// Generate and display goal URL
	baseURL := getBaseURL(config)
	goalURL := fmt.Sprintf("%s/%s/%s", baseURL, url.PathEscape(config.Username), url.PathEscape(goal.Slug))
	details += fmt.Sprintf("URL:         %s\n", goalURL)

	// Display fine print if it exists (at the end)
	if goal.Fineprint != "" {
		details += fmt.Sprintf("Fine print:  %s\n", goal.Fineprint)
	}

	// Display recent datapoints if available
	if len(goal.Datapoints) > 0 {
		details += formatRecentDatapoints(goal.Datapoints)
	}

	return details
}
