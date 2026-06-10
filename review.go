package main

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

	// Fetch just the goal list (one request) so the TUI opens immediately. Each
	// goal's datapoints and road are loaded lazily on demand as the user views
	// it (see fetchGoalDetailsCmd), instead of fetching every goal up front —
	// which took ~50s for accounts with many goals.
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

	// Long-lived context cancelled when the TUI exits, so in-flight lazy detail
	// fetches don't outlive the program (per the client.go context contract).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch the interactive review TUI
	model := initialReviewModel(goals, config)
	model.ctx = ctx
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
		os.Exit(1)
	}
}

// reviewModel holds the state for the review command
type reviewModel struct {
	goals    []Goal
	details  map[string]*Goal    // lazily-fetched full goals (datapoints, road, …) keyed by slug
	inFlight map[string]struct{} // slugs with a detail fetch currently in flight (dedup)
	loading  bool                // a detail fetch for the current goal is in flight
	ctx      context.Context     // cancelled when the TUI exits; cancels in-flight fetches
	config   *Config
	current  int    // current goal index
	width    int    // terminal width
	height   int    // terminal height
	err      string // error message to display
}

// initialReviewModel creates a new review model. The first goal's details fetch
// is dispatched by Init; because Init can't persist model state (it returns only
// a Cmd), the constructor pre-marks that goal as in-flight and loading here.
func initialReviewModel(goals []Goal, config *Config) reviewModel {
	m := reviewModel{
		goals:    goals,
		details:  make(map[string]*Goal),
		inFlight: make(map[string]struct{}),
		ctx:      context.Background(), // overridden with a cancellable ctx by handleReviewCommand
		config:   config,
		current:  0,
		loading:  len(goals) > 0,
	}
	if len(goals) > 0 {
		m.inFlight[goals[0].Slug] = struct{}{}
	}
	return m
}

// goalDetailsMsg carries the result of a lazy per-goal details fetch.
type goalDetailsMsg struct {
	slug string
	goal *Goal
	err  error
}

// fetchGoalDetailsCmd fetches one goal's full details (datapoints + road) in the
// background so the TUI opens immediately and navigation stays responsive. The
// context lets the fetch be cancelled when the user quits.
func fetchGoalDetailsCmd(ctx context.Context, config *Config, slug string) tea.Cmd {
	return func() tea.Msg {
		goal, err := NewHTTPClient(config).FetchGoalWithDatapoints(ctx, slug)
		return goalDetailsMsg{slug: slug, goal: goal, err: err}
	}
}

// ensureDetails returns a command to fetch the current goal's details if they
// aren't already cached or in flight, updating the loading flag accordingly.
// Deduping on inFlight stops rapid navigation (away and back before a fetch
// resolves) from firing a second request for the same goal.
func (m *reviewModel) ensureDetails() tea.Cmd {
	if len(m.goals) == 0 {
		m.loading = false
		return nil
	}
	slug := m.goals[m.current].Slug
	if _, ok := m.details[slug]; ok {
		m.loading = false
		return nil
	}
	m.loading = true
	if _, ok := m.inFlight[slug]; ok {
		return nil // already fetching this goal; just keep showing the spinner
	}
	m.inFlight[slug] = struct{}{}
	return fetchGoalDetailsCmd(m.ctx, m.config, slug)
}

func (m reviewModel) Init() tea.Cmd {
	// The constructor already marked goals[0] in-flight; just dispatch its fetch.
	if len(m.goals) == 0 {
		return nil
	}
	return fetchGoalDetailsCmd(m.ctx, m.config, m.goals[0].Slug)
}

func (m reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case goalDetailsMsg:
		// This fetch is no longer in flight.
		delete(m.inFlight, msg.slug)
		// Cache the result regardless of which goal is now current (the user
		// may have navigated on). Only touch loading/err for the current goal.
		isCurrent := len(m.goals) > 0 && msg.slug == m.goals[m.current].Slug
		if msg.err != nil {
			if isCurrent {
				m.loading = false
				m.err = fmt.Sprintf("Failed to load goal details: %s", redactError(msg.err))
			}
			return m, nil
		}
		m.details[msg.slug] = msg.goal
		if isCurrent {
			m.loading = false
			m.err = ""
		}
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
			return m, m.ensureDetails()

		case "left", "h", "p", "k":
			// Previous goal
			if m.current > 0 {
				m.current--
			}
			m.err = ""
			return m, m.ensureDetails()

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

	// Start from the bulk summary goal, then merge in only the fields the
	// per-goal detail fetch adds (datapoints + the chart's road/window inputs).
	// Merging rather than replacing keeps the summary fields (title, limsum,
	// deadline, …) intact even if a detail response is ever sparse.
	goal := m.goals[m.current]
	if d, ok := m.details[goal.Slug]; ok {
		goal.Datapoints = d.Datapoints
		goal.Roadall = d.Roadall
		goal.Tmin = d.Tmin
		goal.Tmax = d.Tmax
		goal.Initday = d.Initday
		goal.Kyoom = d.Kyoom
		goal.Yaw = d.Yaw
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

	// Goal details section
	detailStyle := lipgloss.NewStyle().
		Padding(0, 2)

	details := formatGoalDetails(&goal, m.config, time.Now())

	view += detailStyle.Render(details) + "\n"

	// Progress chart (datapoints vs. bright red line). Empty when the goal has
	// no datapoints or none inside the charted window.
	if chart := renderGoalChart(goal, m.width); chart != "" {
		view += chart
	}

	// Loading indicator while this goal's datapoints/chart are being fetched.
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 2)
		view += loadingStyle.Render("Loading datapoints…") + "\n"
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
	scale := math.Pow10(rateDisplayDecimals)
	rounded := math.Round(rate*scale) / scale
	if rounded == 0 {
		// Normalize -0 (a small negative rate that rounds to zero) to "0" so
		// do-less / downward-sloping goals don't render a confusing "-0".
		return "0"
	}
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

// formatGoalDetails formats the goal details in a consistent way for both view
// and review commands. now is the reference clock for the 7-day forecast; pass
// time.Now() in production.
func formatGoalDetails(goal *Goal, config *Config, now time.Time) string {
	var details string

	// Field order follows issue #229: the goal's commitment (rate, autoratchet)
	// first, then urgency (limsum, deadline, due time), then stakes (pledge),
	// then reference info (title, url). Fields the issue didn't enumerate
	// (autodata, fine print, recent datapoints) follow, preserving that order.

	// Display rate (n / unit). When the current rate differs from the end
	// rate (a non-flat road), show both so the user sees what they're held to
	// today versus where the goal is heading.
	if goal.Rate != nil && goal.Runits != "" {
		rateStr := formatRate(*goal.Rate, goal.Runits, goal.Gunits)
		if cur := goal.CurrentRate(); cur != nil && formatRateValue(*cur) != formatRateValue(*goal.Rate) {
			rateStr = fmt.Sprintf("%s (current), %s (end)",
				formatRate(*cur, goal.Runits, goal.Gunits),
				formatRateValue(*goal.Rate))
		}
		details += fmt.Sprintf("Rate:        %s\n", rateStr)
	}

	// Display autoratchet only if set (not nil)
	if goal.Autoratchet != nil {
		details += fmt.Sprintf("Autoratchet: %.0f\n", *goal.Autoratchet)
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

	// Display title only if not empty
	if goal.Title != "" {
		details += fmt.Sprintf("Title:       %s\n", goal.Title)
	}

	// Generate and display goal URL
	baseURL := getBaseURL(config)
	goalURL := fmt.Sprintf("%s/%s/%s", baseURL, url.PathEscape(config.Username), url.PathEscape(goal.Slug))
	details += fmt.Sprintf("URL:         %s\n", goalURL)

	// Display autodata only if not empty
	if goal.Autodata != "" {
		details += fmt.Sprintf("Autodata:    %s\n", goal.Autodata)
	}

	// Display fine print if it exists
	if goal.Fineprint != "" {
		details += fmt.Sprintf("Fine print:  %s\n", goal.Fineprint)
	}

	// Display the next-seven-days "amount due" forecast, when available.
	details += formatSevenDayForecastAt(goal, now)

	// Display recent datapoints if available
	if len(goal.Datapoints) > 0 {
		details += formatRecentDatapoints(goal.Datapoints)
	}

	return details
}

// formatSevenDayForecastAt renders the goal's per-day "amount due" forecast for
// the next seven days from Beeminder's dueby map. Each entry carries the delta
// (how much is needed that day to stay on the safe side of the bright line) and
// the running red-line total, both pre-formatted by Beeminder to the goal's
// display precision — the same strings the Beeminder Android app shows. Returns
// an empty string when the goal has no dueby data (e.g. a freshly created
// goal), so callers can append the result unconditionally.
//
// now is the reference clock (injected for testability). The forecast anchors
// on the goal's current daystamp (deadline-aware) rather than the dueby map's
// sort position: Beeminder's dueby can carry past daystamps as well as today's
// and future ones, so we drop anything before today and label each remaining
// day by its actual date — never by slice index — to avoid mislabelling a stale
// past day as "Today".
func formatSevenDayForecastAt(goal *Goal, now time.Time) string {
	if len(goal.Dueby) == 0 {
		return ""
	}

	today := todayDaystampFor(*goal, now)

	days := make([]string, 0, len(goal.Dueby))
	for daystamp := range goal.Dueby {
		// YYYYMMDD strings compare chronologically; drop stale past daystamps.
		if daystamp >= today {
			days = append(days, daystamp)
		}
	}
	sort.Strings(days)
	if len(days) > 7 {
		days = days[:7]
	}
	if len(days) == 0 {
		return ""
	}

	type forecastRow struct{ label, due, total string }
	rows := make([]forecastRow, 0, len(days))
	for _, daystamp := range days {
		entry := goal.Dueby[daystamp]
		rows = append(rows, forecastRow{
			label: forecastDayLabel(daystamp, today),
			due:   entry.FormattedDelta,
			total: entry.FormattedTotal,
		})
	}

	// Size each column to the widest of its header and values so the columns
	// line up regardless of count-vs-time formatting (e.g. "+1" vs "+00:05:59").
	dayW, dueW, totW := len("Day"), len("Due"), len("Total")
	for _, r := range rows {
		dayW = max(dayW, len(r.label))
		dueW = max(dueW, len(r.due))
		totW = max(totW, len(r.total))
	}

	var b strings.Builder
	b.WriteString("\n7-Day Forecast:\n")
	fmt.Fprintf(&b, "  %-*s  %-*s  %-*s\n", dayW, "Day", dueW, "Due", totW, "Total")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-*s  %-*s  %-*s\n", dayW, r.label, dueW, r.due, totW, r.total)
	}
	return b.String()
}

// forecastDayLabel returns a human label for a dueby daystamp (YYYYMMDD),
// relative to today's daystamp: "Today", "Tomorrow", or otherwise weekday +
// ordinal day-of-month, e.g. "Fri (12th)". Labelling by actual date (rather
// than position) keeps the labels correct even if the dueby map has gaps or
// stale entries. Falls back to the raw daystamp if it can't be parsed.
func forecastDayLabel(daystamp, today string) string {
	if daystamp == today {
		return "Today"
	}
	t, err := time.Parse("20060102", daystamp)
	if err != nil {
		return daystamp
	}
	if todayTime, err := time.Parse("20060102", today); err == nil {
		if daystamp == todayTime.AddDate(0, 0, 1).Format("20060102") {
			return "Tomorrow"
		}
	}
	return fmt.Sprintf("%s (%d%s)", t.Format("Mon"), t.Day(), ordinalSuffix(t.Day()))
}

// ordinalSuffix returns the English ordinal suffix ("st", "nd", "rd", "th") for
// a day-of-month, handling the 11–13 exceptions.
func ordinalSuffix(day int) string {
	if day >= 11 && day <= 13 {
		return "th"
	}
	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}
