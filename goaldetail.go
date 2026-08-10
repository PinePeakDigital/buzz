package main

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// This file holds the goal-detail formatting library: the pure functions that
// render a goal's detail block (rate, limsum, deadline, pledge, forecast,
// recent datapoints, archive banner). Shared by `buzz view`, `buzz review`, and
// the TUI grid — it lives on its own rather than in review.go so "how is a goal
// detail formatted?" has one home regardless of which command you're reading.

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

	// The most recent `count` datapoints, newest first.
	recent := make([]Datapoint, 0, count)
	for i := len(datapoints) - 1; i >= len(datapoints)-count; i-- {
		recent = append(recent, datapoints[i])
	}

	dates, values, maxValueLen := formatDatapointRows(recent)

	output := "\nRecent datapoints:\n"
	for i, dp := range recent {
		if dp.Comment != "" {
			output += fmt.Sprintf("  %s   %-*s   %s\n", dates[i], maxValueLen, values[i], dp.Comment)
		} else {
			output += fmt.Sprintf("  %s   %s\n", dates[i], values[i])
		}
	}

	return output
}

// formatArchiveBanner returns a one-line warning banner when the goal is
// scheduled for archive (Beeminder sets archivedate to the Unix time it will be
// archived), or "" otherwise. Shared by every full goal view so the warning
// shows up the same way everywhere. now gates on a still-upcoming date so an
// already-archived goal (reachable by slug) doesn't show a past-dated warning.
func formatArchiveBanner(goal *Goal, now time.Time) string {
	if !goal.ScheduledForArchive(now) {
		return ""
	}
	when := time.Unix(goal.Archivedate, 0).Format("Mon Jan 2, 2006")
	banner := fmt.Sprintf("⚠ Scheduled for archive on %s", when)
	return lipgloss.NewStyle().Bold(true).Foreground(archiveColor).Render(banner) + "\n"
}

// formatGoalDetails formats the goal details in a consistent way for both view
// and review commands. now is the reference clock for the 7-day forecast; pass
// time.Now() in production.
func formatGoalDetails(goal *Goal, config *Config, now time.Time) string {
	details := formatArchiveBanner(goal, now)

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
