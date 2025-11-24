package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleReviewCommand launches an interactive review of all goals
func handleReviewCommand() {
	// Load config
	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)

	// Fetch goals
	goals, err := client.FetchGoals(context.Background())
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
	goals         []Goal
	detailedGoals map[string]*Goal // Cache of goals with full details (datapoints, road, etc.)
	config        *Config
	current       int    // current goal index
	width         int    // terminal width
	height        int    // terminal height
	err           string // error message to display
	loading       bool   // whether we're currently loading goal details
}

// initialReviewModel creates a new review model
func initialReviewModel(goals []Goal, config *Config) reviewModel {
	return reviewModel{
		goals:         goals,
		detailedGoals: make(map[string]*Goal),
		config:        config,
		current:       0,
	}
}

// goalDetailsFetchedMsg is sent when goal details are fetched
type goalDetailsFetchedMsg struct {
	slug string
	goal *Goal
	err  error
}

// fetchGoalDetailsCmd fetches full details for a goal
func fetchGoalDetailsCmd(config *Config, slug string) tea.Cmd {
	return func() tea.Msg {
		goal, err := FetchGoalWithDatapoints(config, slug)
		return goalDetailsFetchedMsg{
			slug: slug,
			goal: goal,
			err:  err,
		}
	}
}

func (m reviewModel) Init() tea.Cmd {
	// Fetch details for the first goal
	if len(m.goals) > 0 {
		return fetchGoalDetailsCmd(m.config, m.goals[0].Slug)
	}
	return nil
}

func (m reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case goalDetailsFetchedMsg:
		// Goal details have been fetched
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Sprintf("Failed to load goal details: %v", msg.err)
		} else {
			m.detailedGoals[msg.slug] = msg.goal
			m.err = ""
		}
		return m, nil

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
				m.err = ""
				// Fetch details if not already cached
				slug := m.goals[m.current].Slug
				if _, ok := m.detailedGoals[slug]; !ok {
					m.loading = true
					return m, fetchGoalDetailsCmd(m.config, slug)
				}
			}
			return m, nil

		case "left", "h", "p", "k":
			// Previous goal
			if m.current > 0 {
				m.current--
				m.err = ""
				// Fetch details if not already cached
				slug := m.goals[m.current].Slug
				if _, ok := m.detailedGoals[slug]; !ok {
					m.loading = true
					return m, fetchGoalDetailsCmd(m.config, slug)
				}
			}
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

	// Use detailed goal if available
	detailedGoal, hasDetails := m.detailedGoals[goal.Slug]
	if hasDetails {
		goal = *detailedGoal
	}

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

	// Loading indicator
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Padding(0, 2)
		view += loadingStyle.Render("Loading goal details...") + "\n\n"
	}

	// Goal details section
	detailStyle := lipgloss.NewStyle().
		Padding(0, 2)

	details := formatGoalDetails(&goal, m.config)

	view += detailStyle.Render(details) + "\n"

	// Display goal chart if detailed data is available
	if hasDetails {
		chart := renderGoalChart(goal, m.width)
		if chart != "" {
			view += chart
		}
	}

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

	// Display current rate (n / unit)
	if goal.Rate != nil && goal.Runits != "" {
		rateStr := formatRate(*goal.Rate, goal.Runits, goal.Gunits)
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

	return details
}

// renderGoalChart renders an ASCII chart showing goal progress with datapoints and road
func renderGoalChart(goal Goal, width int) string {
	// Return empty if no datapoints
	if len(goal.Datapoints) == 0 {
		return ""
	}

	// Parse timeframe from tmin/tmax or default to last 30 days
	var startTime, endTime time.Time
	now := time.Now()

	if goal.Tmin != "" && goal.Tmax != "" {
		var err error
		startTime, err = time.Parse("2006-01-02", goal.Tmin)
		if err != nil {
			// Fallback to last 30 days
			startTime = now.AddDate(0, 0, -30)
		}
		endTime, err = time.Parse("2006-01-02", goal.Tmax)
		if err != nil {
			// Fallback to today
			endTime = now
		}
	} else {
		// Default to last 30 days
		startTime = now.AddDate(0, 0, -30)
		endTime = now
	}

	// Filter datapoints within timeframe
	var filteredDatapoints []Datapoint
	for _, dp := range goal.Datapoints {
		dpTime := time.Unix(dp.Timestamp, 0)
		if !dpTime.Before(startTime) && !dpTime.After(endTime) {
			filteredDatapoints = append(filteredDatapoints, dp)
		}
	}

	// Return empty if no datapoints in timeframe
	if len(filteredDatapoints) == 0 {
		return ""
	}

	// Sort datapoints by timestamp using sort.Slice
	sortedDatapoints := make([]Datapoint, len(filteredDatapoints))
	copy(sortedDatapoints, filteredDatapoints)
	sort.Slice(sortedDatapoints, func(i, j int) bool {
		return sortedDatapoints[i].Timestamp < sortedDatapoints[j].Timestamp
	})

	// Process datapoints based on cumulative setting
	processedDatapoints := make([]struct {
		timestamp int64
		value     float64
	}, len(sortedDatapoints))

	if goal.Kyoom {
		// Cumulative: sum values progressively
		sum := 0.0
		for i, dp := range sortedDatapoints {
			sum += dp.Value
			processedDatapoints[i].timestamp = dp.Timestamp
			processedDatapoints[i].value = sum
		}
	} else {
		// Non-cumulative: use actual values
		for i, dp := range sortedDatapoints {
			processedDatapoints[i].timestamp = dp.Timestamp
			processedDatapoints[i].value = dp.Value
		}
	}

	// Chart dimensions
	chartHeight := 10
	chartWidth := width - 8 // Leave room for padding and axis labels
	if chartWidth < 40 {
		chartWidth = 40
	}
	if chartWidth > 80 {
		chartWidth = 80
	}

	// Calculate road values for each column of the chart
	roadValues := getRoadValuesForTimeframe(goal, startTime, endTime, chartWidth)

	// Find min and max values for scaling
	minVal := processedDatapoints[0].value
	maxVal := processedDatapoints[0].value
	for _, dp := range processedDatapoints {
		if dp.value < minVal {
			minVal = dp.value
		}
		if dp.value > maxVal {
			maxVal = dp.value
		}
	}
	for _, rv := range roadValues {
		if rv < minVal {
			minVal = rv
		}
		if rv > maxVal {
			maxVal = rv
		}
	}

	// Add some padding to the range
	valueRange := maxVal - minVal
	if valueRange == 0 {
		valueRange = 1
	}
	minVal -= valueRange * 0.1
	maxVal += valueRange * 0.1

	// Build the chart
	var chart strings.Builder

	// Header with goal type and timeframe
	chart.WriteString("\n")
	chartStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Padding(0, 2)

	goalType := "Do More"
	if goal.Yaw == -1 {
		goalType = "Do Less"
	}
	cumulativeStr := ""
	if goal.Kyoom {
		cumulativeStr = " (Cumulative)"
	}

	header := fmt.Sprintf("Goal Progress Chart - %s%s", goalType, cumulativeStr)
	chart.WriteString(chartStyle.Render(header) + "\n")

	timeframeInfo := fmt.Sprintf("Timeframe: %s to %s", startTime.Format("Jan 2"), endTime.Format("Jan 2, 2006"))
	chart.WriteString(chartStyle.Render(timeframeInfo) + "\n\n")

	// Draw the chart row by row (top to bottom)
	for row := chartHeight - 1; row >= 0; row-- {
		// Calculate the value at this row
		rowValue := minVal + (maxVal-minVal)*(float64(row)/float64(chartHeight-1))

		// Y-axis label
		chart.WriteString(fmt.Sprintf("%6.1f │", rowValue))

		// Draw the row
		for col := 0; col < chartWidth; col++ {
			// Get the road value for this column (road values are calculated per column)
			roadVal := roadValues[col]

			// Calculate which datapoint this column represents
			dpIndex := (col * len(processedDatapoints)) / chartWidth
			if dpIndex >= len(processedDatapoints) {
				dpIndex = len(processedDatapoints) - 1
			}
			dp := processedDatapoints[dpIndex]

			// Calculate normalized positions (0.0 to 1.0)
			dpPos := (dp.value - minVal) / (maxVal - minVal)
			roadPos := (roadVal - minVal) / (maxVal - minVal)
			rowPos := float64(row) / float64(chartHeight-1)

			// Tolerance for "close enough"
			tolerance := 1.0 / float64(chartHeight*2)

			// Determine what to draw
			dpClose := dpPos >= rowPos-tolerance && dpPos <= rowPos+tolerance
			roadClose := roadPos >= rowPos-tolerance && roadPos <= rowPos+tolerance

			if dpClose && roadClose {
				// Both datapoint and road at this position
				chart.WriteString("█")
			} else if dpClose {
				// Just datapoint - check if on good or bad side
				goodSide := false
				if goal.Yaw == 1 {
					// Good side is above road
					goodSide = dp.value >= roadVal
				} else {
					// Good side is below road
					goodSide = dp.value <= roadVal
				}
				if goodSide {
					chart.WriteString("●")
				} else {
					chart.WriteString("○")
				}
			} else if roadClose {
				// Just road
				chart.WriteString("─")
			} else {
				chart.WriteString(" ")
			}
		}
		chart.WriteString("\n")
	}

	// X-axis
	chart.WriteString("       └" + strings.Repeat("─", chartWidth) + "\n")

	// Legend
	legendStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	legend := "Legend: ● = on good side  ○ = on bad side  ─ = yellow brick road  █ = on target"
	chart.WriteString(legendStyle.Render(legend) + "\n")

	return chart.String()
}

// getRoadValuesForTimeframe calculates road values for each datapoint timestamp
func getRoadValuesForTimeframe(goal Goal, startTime, endTime time.Time, numPoints int) []float64 {
	values := make([]float64, numPoints)

	// If no roadall data, return zeros
	if len(goal.Roadall) == 0 {
		return values
	}

	// Handle edge case where numPoints is 1
	if numPoints == 1 {
		values[0] = getRoadValueAtTime(goal, startTime)
		return values
	}

	// Calculate timestamps for each point
	duration := endTime.Sub(startTime)
	for i := 0; i < numPoints; i++ {
		t := startTime.Add(time.Duration(float64(duration) * float64(i) / float64(numPoints-1)))
		values[i] = getRoadValueAtTime(goal, t)
	}

	return values
}

// getRoadValueAtTime interpolates the road value at a specific time.
//
// Beeminder's roadall is a piecewise schedule: the first row is the anchor
// (t, v set, r nil), and each subsequent row has exactly one of v/r null —
// either the value at that t, or the rate (per runits) used to get there.
// To interpolate we walk forward, materialising each row's value from the
// previous anchor and the row's rate when the row's own value is missing.
func getRoadValueAtTime(goal Goal, t time.Time) float64 {
	if len(goal.Roadall) < 2 {
		return 0
	}

	target := float64(t.Unix())

	// Resolve the anchor (row 0): must have t and v set.
	first := goal.Roadall[0]
	if len(first) < 3 || first[0] == nil || first[1] == nil {
		return 0
	}
	prevT := *first[0]
	prevV := *first[1]

	// If target is before the road starts, extrapolate backwards using the
	// first segment's slope so the chart can still draw a value.
	if target < prevT {
		slope, ok := segmentSlopePerSecond(goal, 1, prevT, prevV)
		if !ok {
			return prevV
		}
		return prevV + slope*(target-prevT)
	}

	for i := 1; i < len(goal.Roadall); i++ {
		cur := goal.Roadall[i]
		if len(cur) < 3 || cur[0] == nil {
			return prevV
		}
		curT := *cur[0]

		// Materialise this row's value.
		var curV float64
		switch {
		case cur[1] != nil:
			curV = *cur[1]
		case cur[2] != nil:
			rate := *cur[2]
			rps := ratePerDay(rate, goal.Runits) / 86400.0
			curV = prevV + rps*(curT-prevT)
		default:
			return prevV
		}

		if target <= curT {
			if curT == prevT {
				return curV
			}
			frac := (target - prevT) / (curT - prevT)
			return prevV + frac*(curV-prevV)
		}

		prevT = curT
		prevV = curV
	}

	// Past the end of the road: return the last row's materialised value.
	return prevV
}

// segmentSlopePerSecond returns the slope (gunits/second) of roadall segment
// ending at index i, given the prior anchor (prevT, prevV). Used to
// extrapolate before the start of the road.
func segmentSlopePerSecond(goal Goal, i int, prevT, prevV float64) (float64, bool) {
	if i >= len(goal.Roadall) {
		return 0, false
	}
	cur := goal.Roadall[i]
	if len(cur) < 3 || cur[0] == nil {
		return 0, false
	}
	if cur[2] != nil {
		return ratePerDay(*cur[2], goal.Runits) / 86400.0, true
	}
	if cur[1] == nil {
		return 0, false
	}
	dt := *cur[0] - prevT
	if dt == 0 {
		return 0, false
	}
	return (*cur[1] - prevV) / dt, true
}
