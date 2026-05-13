package main

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Deadline-related predicates and formatters. All work in terms of a goal's
// `losedate` (Unix seconds) plus an explicit "now" so the caller can inject
// deterministic time for tests. The bare entry points without `At` delegate
// to the `At` variants with `time.Now()` for production use.

// IsDueToday checks if a goal is due today (on or before midnight tonight)
func IsDueToday(losedate int64) bool {
	return IsDueTodayAt(losedate, time.Now())
}

// IsDueTodayAt checks if a goal is due today relative to a given time
func IsDueTodayAt(losedate int64, now time.Time) bool {
	goalTime := time.Unix(losedate, 0)

	// Get start of tomorrow (midnight tonight)
	startOfTomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	// Goal is due today if it's due before the start of tomorrow
	// This includes overdue goals and goals due later today
	return goalTime.Before(startOfTomorrow)
}

// IsDueTomorrow checks if a goal is due tomorrow (between midnight tonight and midnight tomorrow)
func IsDueTomorrow(losedate int64) bool {
	return IsDueTomorrowAt(losedate, time.Now())
}

// IsDueTomorrowAt checks if a goal is due tomorrow relative to a given time
func IsDueTomorrowAt(losedate int64, now time.Time) bool {
	goalTime := time.Unix(losedate, 0)

	// Get start of tomorrow (midnight tonight)
	startOfTomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	// Get start of day after tomorrow
	startOfDayAfterTomorrow := time.Date(now.Year(), now.Month(), now.Day()+2, 0, 0, 0, 0, now.Location())

	// Goal is due tomorrow if it's on or after midnight tonight but before the day after tomorrow
	return !goalTime.Before(startOfTomorrow) && goalTime.Before(startOfDayAfterTomorrow)
}

// IsDueWithin checks if a goal is due within the specified duration from now
func IsDueWithin(losedate int64, duration time.Duration) bool {
	return IsDueWithinAt(losedate, duration, time.Now())
}

// IsDueWithinAt checks if a goal is due within the specified duration from the given time
func IsDueWithinAt(losedate int64, duration time.Duration, now time.Time) bool {
	goalTime := time.Unix(losedate, 0)
	cutoffTime := now.Add(duration)

	// Goal is due within the duration if it's not after the cutoff time
	return !goalTime.After(cutoffTime)
}

// ParseDuration parses a duration string (e.g., "10m", "1h", "5d", "1w") and returns time.Duration
// Supported formats: Nm (minutes), Nh (hours), Nd (days), Nw (weeks) where N is a number
// Returns the duration and true on success, 0 and false on error
func ParseDuration(durationStr string) (time.Duration, bool) {
	if len(durationStr) < 2 {
		// Need at least one character for number and one for unit
		return 0, false
	}

	// Get the unit (last character)
	unit := durationStr[len(durationStr)-1]

	// Get the numeric part
	numStr := durationStr[:len(durationStr)-1]

	// Parse the number. strconv.ParseFloat rejects empty input and trailing
	// garbage; the explicit NaN/Inf check below catches "NaNh" / "Infh" which
	// would otherwise convert to a 0-duration via Go's NaN→int64 fallback and
	// produce a misleading ok=true return.
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, false
	}
	if math.IsNaN(num) || math.IsInf(num, 0) {
		return 0, false
	}

	// Reject negative durations, which don't make sense for "due within" semantics
	if num < 0 {
		return 0, false
	}

	// Convert to duration based on unit
	var duration time.Duration
	switch unit {
	case 'm', 'M':
		duration = time.Duration(num * float64(time.Minute))
	case 'h', 'H':
		duration = time.Duration(num * float64(time.Hour))
	case 'd', 'D':
		duration = time.Duration(num * 24 * float64(time.Hour))
	case 'w', 'W':
		duration = time.Duration(num * 7 * 24 * float64(time.Hour))
	default:
		return 0, false
	}

	// Check for overflow: time.Duration is int64 nanoseconds
	// Maximum duration is ~290 years (math.MaxInt64 nanoseconds)
	// If the result is negative, it overflowed
	if duration < 0 {
		return 0, false
	}

	return duration, true
}

// FormatDueDate formats the losedate timestamp into a readable string
func FormatDueDate(losedate int64) string {
	return FormatDueDateAt(losedate, time.Now())
}

// FormatDueDateAt formats the losedate timestamp relative to a given time
func FormatDueDateAt(losedate int64, now time.Time) string {
	t := time.Unix(losedate, 0)

	// Calculate duration until due
	duration := t.Sub(now)
	totalHours := duration.Hours()

	if totalHours < 0 {
		return "OVERDUE"
	}

	// If less than 1 day, show in hours or minutes
	if totalHours < 24 {
		if totalHours >= 1 {
			// Show in hours (rounded down)
			hours := int(totalHours)
			return fmt.Sprintf("%dh", hours)
		} else {
			// Show in minutes (rounded down)
			minutes := int(duration.Minutes())
			if minutes < 1 {
				return "0m"
			}
			return fmt.Sprintf("%dm", minutes)
		}
	}

	// Show in days with "d" suffix
	days := int(totalHours / 24)
	return fmt.Sprintf("%dd", days)
}

// FormatAbsoluteDeadline formats the losedate timestamp as an absolute date/time string
// Returns a compact format suitable for table display
func FormatAbsoluteDeadline(losedate int64) string {
	return FormatAbsoluteDeadlineAt(losedate, time.Now())
}

// FormatAbsoluteDeadlineAt formats the losedate timestamp as an absolute date/time string relative to a given time
// Returns a compact format suitable for table display
func FormatAbsoluteDeadlineAt(losedate int64, now time.Time) string {
	// Convert Unix timestamp to the same timezone as now for accurate comparisons
	t := time.Unix(losedate, 0).In(now.Location())

	// Get start of today
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Get start of tomorrow
	startOfTomorrow := startOfToday.AddDate(0, 0, 1)

	// If it's today, show time only (e.g., "3:04 PM")
	if !t.Before(startOfToday) && t.Before(startOfTomorrow) {
		return t.Format("3:04 PM")
	}

	// If it's tomorrow, show "tomorrow" + time (e.g., "tomorrow 3:04 PM")
	startOfDayAfterTomorrow := startOfTomorrow.AddDate(0, 0, 1)
	if !t.Before(startOfTomorrow) && t.Before(startOfDayAfterTomorrow) {
		return "tomorrow " + t.Format("3:04 PM")
	}

	// For other dates, show date and time (e.g., "Jan 2 3:04 PM")
	return t.Format("Jan 2 3:04 PM")
}

// isDueTodayFilterAt returns true if the goal is due today (relative to now) and
// hasn't already reached its end value. Exposed for deterministic time-based tests.
func isDueTodayFilterAt(g Goal, now time.Time) bool {
	return IsDueTodayAt(g.Losedate, now) && !IsEndValueReached(g)
}

// isDueTodayFilter returns true if the goal is due today and hasn't already reached its end value
func isDueTodayFilter(g Goal) bool {
	return isDueTodayFilterAt(g, time.Now())
}

// isDueTomorrowFilterAt returns true if the goal is due by the end of tomorrow
// (i.e. due today or tomorrow) and hasn't already reached its end value. Goals
// due today are included so the tomorrow view shows the full set of goals the
// user must address to avoid a beemergency tomorrow. Exposed for deterministic
// time-based tests.
func isDueTomorrowFilterAt(g Goal, now time.Time) bool {
	if IsEndValueReached(g) {
		return false
	}
	return IsDueTodayAt(g.Losedate, now) || IsDueTomorrowAt(g.Losedate, now)
}

// dueLaterTodayAt reports whether a goal's losedate falls between now and the
// start of tomorrow. Used as the gating predicate for the tomorrow-view
// bumping helpers: overdue goals (losedate < now) and goals due tomorrow or
// later are left untouched so their actual deadline keeps showing.
func dueLaterTodayAt(g Goal, now time.Time) bool {
	return g.Losedate >= now.Unix() && IsDueTodayAt(g.Losedate, now)
}
