package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
)

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

	// For cumulative goals, we need to calculate the running sum from ALL datapoints
	// before filtering, so that datapoints within the timeframe show their true cumulative value
	var processedDatapoints []struct {
		timestamp int64
		value     float64
	}

	if goal.Kyoom {
		// Cumulative goal: first sort ALL datapoints, calculate running sum, then filter
		allDatapoints := make([]Datapoint, len(goal.Datapoints))
		copy(allDatapoints, goal.Datapoints)
		sort.Slice(allDatapoints, func(i, j int) bool {
			return allDatapoints[i].Timestamp < allDatapoints[j].Timestamp
		})

		// Calculate running sum and keep only those in timeframe
		// Also track the cumulative value at the start of the timeframe
		sum := 0.0
		startSum := 0.0
		for _, dp := range allDatapoints {
			dpTime := time.Unix(dp.Timestamp, 0)

			// Track the sum just before the timeframe starts
			if dpTime.Before(startTime) {
				sum += dp.Value
				startSum = sum
			} else if !dpTime.After(endTime) {
				// Datapoint is within timeframe
				sum += dp.Value
				processedDatapoints = append(processedDatapoints, struct {
					timestamp int64
					value     float64
				}{
					timestamp: dp.Timestamp,
					value:     sum,
				})
			}
			// Datapoints after endTime are ignored
		}

		// Always add a starting point at the beginning of the timeframe
		// with the cumulative sum up to that point (handles case where first
		// datapoint is partway through the timeframe)
		if len(processedDatapoints) > 0 || startSum != 0 {
			// Insert at the beginning
			startPoint := struct {
				timestamp int64
				value     float64
			}{
				timestamp: startTime.Unix(),
				value:     startSum,
			}
			// Prepend the start point
			newProcessed := make([]struct {
				timestamp int64
				value     float64
			}, 0, len(processedDatapoints)+1)
			newProcessed = append(newProcessed, startPoint)
			newProcessed = append(newProcessed, processedDatapoints...)
			processedDatapoints = newProcessed
		}

		// If we still have no datapoints, there's nothing to show
		if len(processedDatapoints) == 0 {
			return ""
		}
	} else {
		// Non-cumulative: filter datapoints within timeframe first
		var filteredDatapoints []Datapoint
		for _, dp := range goal.Datapoints {
			dpTime := time.Unix(dp.Timestamp, 0)
			if !dpTime.Before(startTime) && !dpTime.After(endTime) {
				filteredDatapoints = append(filteredDatapoints, dp)
			}
		}

		// Sort filtered datapoints by timestamp
		sort.Slice(filteredDatapoints, func(i, j int) bool {
			return filteredDatapoints[i].Timestamp < filteredDatapoints[j].Timestamp
		})

		// Use actual values
		for _, dp := range filteredDatapoints {
			processedDatapoints = append(processedDatapoints, struct {
				timestamp int64
				value     float64
			}{
				timestamp: dp.Timestamp,
				value:     dp.Value,
			})
		}
	}

	// Return empty if no datapoints in timeframe
	if len(processedDatapoints) == 0 {
		return ""
	}

	// Chart dimensions
	chartHeight := 10
	chartWidth := width - 10 // Leave room for padding and axis labels
	if chartWidth < 40 {
		chartWidth = 40
	}
	if chartWidth > 80 {
		chartWidth = 80
	}

	// Calculate road values for the timeframe (one per chart column)
	roadValues := getRoadValuesForTimeframe(goal, startTime, endTime, chartWidth)

	// Create datapoint values array aligned to chart columns
	// Map each datapoint to its appropriate column based on timestamp
	datapointValues := make([]float64, chartWidth)
	hasDatapoint := make([]bool, chartWidth)
	duration := endTime.Sub(startTime)

	for _, dp := range processedDatapoints {
		dpTime := time.Unix(dp.timestamp, 0)
		// Calculate which column this datapoint belongs to
		progress := dpTime.Sub(startTime).Seconds() / duration.Seconds()
		col := int(progress * float64(chartWidth-1))
		if col < 0 {
			col = 0
		}
		if col >= chartWidth {
			col = chartWidth - 1
		}
		// Since processedDatapoints is sorted by timestamp, later iterations
		// will overwrite earlier ones for the same column (which is correct)
		datapointValues[col] = dp.value
		hasDatapoint[col] = true
	}

	// Interpolate between datapoints for a smoother line
	// First pass: find first and last datapoint positions
	firstDP, lastDP := -1, -1
	for i := 0; i < chartWidth; i++ {
		if hasDatapoint[i] {
			if firstDP == -1 {
				firstDP = i
			}
			lastDP = i
		}
	}

	// If we have datapoints, interpolate
	if firstDP >= 0 {
		// Fill before first datapoint with first value
		for i := 0; i < firstDP; i++ {
			datapointValues[i] = datapointValues[firstDP]
		}

		// Fill after last datapoint with last value
		for i := lastDP + 1; i < chartWidth; i++ {
			datapointValues[i] = datapointValues[lastDP]
		}

		// Interpolate between datapoints
		prevDP := firstDP
		for i := firstDP + 1; i <= lastDP; i++ {
			if hasDatapoint[i] {
				// Linear interpolation from prevDP to i
				if i > prevDP+1 {
					startVal := datapointValues[prevDP]
					endVal := datapointValues[i]
					for j := prevDP + 1; j < i; j++ {
						ratio := float64(j-prevDP) / float64(i-prevDP)
						datapointValues[j] = startVal + ratio*(endVal-startVal)
					}
				}
				prevDP = i
			}
		}
	}

	// Build the chart header
	var chart strings.Builder
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

	// Use asciigraph to render both lines
	// Create a combined view showing both datapoints and road
	graphData := [][]float64{datapointValues, roadValues}

	graphOutput := asciigraph.PlotMany(graphData,
		asciigraph.Height(chartHeight),
		asciigraph.Width(chartWidth),
		asciigraph.SeriesColors(asciigraph.Blue, asciigraph.Red),
		asciigraph.Caption("Blue: datapoints, Red: bright red line"),
	)

	chart.WriteString(graphOutput)
	chart.WriteString("\n")

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

// getRoadValueAtTime interpolates the road value at a specific time
func getRoadValueAtTime(goal Goal, t time.Time) float64 {
	if len(goal.Roadall) == 0 {
		return 0
	}

	timestamp := t.Unix()

	// Parse the first segment to check if we're before it
	firstSegment := goal.Roadall[0]
	var firstSegTime int64
	switch v := firstSegment[0].(type) {
	case float64:
		firstSegTime = int64(v)
	case string:
		parsedTime, err := time.Parse("2006-01-02", v)
		if err == nil {
			firstSegTime = parsedTime.Unix()
		}
	}

	// If timestamp is before the first segment, extrapolate backwards using first segment's rate
	if timestamp < firstSegTime {
		var firstValue, firstRate float64
		if len(firstSegment) > 1 {
			switch v := firstSegment[1].(type) {
			case float64:
				firstValue = v
			}
		}
		if len(firstSegment) > 2 {
			switch v := firstSegment[2].(type) {
			case float64:
				firstRate = v
			}
		}
		// Extrapolate backwards: value at first segment - rate * days before
		daysBefore := float64(firstSegTime-timestamp) / 86400.0
		return firstValue - (firstRate * daysBefore)
	}

	// Find the road segment that contains this timestamp
	for i := 0; i < len(goal.Roadall)-1; i++ {
		segment := goal.Roadall[i]
		nextSegment := goal.Roadall[i+1]

		// Parse segment timestamps (can be float64 or string)
		var segTime, nextSegTime int64

		switch v := segment[0].(type) {
		case float64:
			segTime = int64(v)
		case string:
			// Try parsing as date string
			parsedTime, err := time.Parse("2006-01-02", v)
			if err == nil {
				segTime = parsedTime.Unix()
			}
		}

		switch v := nextSegment[0].(type) {
		case float64:
			nextSegTime = int64(v)
		case string:
			parsedTime, err := time.Parse("2006-01-02", v)
			if err == nil {
				nextSegTime = parsedTime.Unix()
			}
		}

		// Check if timestamp is within this segment
		if timestamp >= segTime && timestamp <= nextSegTime {
			// Parse values
			var segValue, rate float64

			if len(segment) > 1 {
				switch v := segment[1].(type) {
				case float64:
					segValue = v
				}
			}

			// Rate is in segment[2]
			if len(segment) > 2 {
				switch v := segment[2].(type) {
				case float64:
					rate = v
				}
			}

			// Calculate days from segment start
			days := float64(timestamp-segTime) / 86400.0

			// Interpolate value
			return segValue + (rate * days)
		}
	}

	// If past the end, use the last segment's value
	lastSegment := goal.Roadall[len(goal.Roadall)-1]
	if len(lastSegment) > 1 {
		switch v := lastSegment[1].(type) {
		case float64:
			return v
		}
	}

	return 0
}
