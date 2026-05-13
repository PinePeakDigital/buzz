package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Wall-clock time-string conversions. Three formats live in the buzz codebase:
//
//   - HH:MM[:SS] baremin time values like "+00:25" or "1:30:45" — parsed via
//     parseTimeValue / re-formatted via formatTimeValue, used by the tomorrow-
//     view baremin math.
//   - The same HH:MM[:SS] format converted to decimal hours (isTimeFormat /
//     timeToDecimalHours), used by `buzz add 1:30` to submit hour-valued
//     datapoints.
//   - Wall-clock-of-day strings ("3:00 PM" or "15:00") and Beeminder's
//     seconds-from-midnight deadline offset, converted by
//     parseTimeToDeadlineOffset / formatDueTime, used by `buzz deadline`.
//
// The three concepts are kept separate because their input shapes and error
// semantics differ; co-locating them in one file makes the parsing surface
// findable in one place rather than spread across main.go, utils.go, review.go.

// parseTimeValue parses a "[-]HH:MM" or "[-]HH:MM:SS" string into total
// seconds. The second return value reports whether the input included a
// seconds component, so callers can preserve the original format on output.
// Returns ok=false for anything that isn't exactly two or three colon-separated
// non-negative integer fields with minutes and seconds < 60.
func parseTimeValue(s string) (totalSeconds int, includeSeconds bool, ok bool) {
	sign := 1
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = strings.TrimPrefix(s, "-")
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, false, false
	}
	hh, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false, false
	}
	mm, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false, false
	}
	ss := 0
	if len(parts) == 3 {
		ss, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, false, false
		}
	}
	// Reject malformed inputs (e.g. "1:75", "1:-05:00", "--1:30"). The leading
	// sign has already been stripped, so any negative field — or minutes/
	// seconds outside [0, 60) — means the string was malformed.
	if hh < 0 || mm < 0 || mm >= 60 || ss < 0 || ss >= 60 {
		return 0, false, false
	}
	return sign * (hh*3600 + mm*60 + ss), len(parts) == 3, true
}

// formatTimeValue formats a signed second count back into Beeminder's
// colon-separated baremin style. When includeSeconds is true the output is
// HH:MM:SS, otherwise HH:MM. Hours, minutes, and seconds are zero-padded and a
// leading "+" is included for non-negative values. When dropping seconds, the
// value is rounded to the nearest minute first so a fractional-minute bump
// from the rate conversion doesn't silently undercount by up to 59 seconds.
func formatTimeValue(totalSeconds int, includeSeconds bool) string {
	if !includeSeconds {
		totalSeconds = int(math.Round(float64(totalSeconds)/60.0)) * 60
	}
	sign := "+"
	if totalSeconds < 0 {
		sign = "-"
		totalSeconds = -totalSeconds
	}
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	if includeSeconds {
		return fmt.Sprintf("%s%02d:%02d:%02d", sign, h, m, totalSeconds%60)
	}
	return fmt.Sprintf("%s%02d:%02d", sign, h, m)
}

// isTimeFormat checks if a string is in time format (HH:MM or HH:MM:SS)
// Returns true for formats like "1:30", "00:05", "2:45:30", etc.
func isTimeFormat(s string) bool {
	s = strings.TrimPrefix(s, "+")
	s = strings.TrimPrefix(s, "-")
	return strings.Contains(s, ":")
}

// timeToDecimalHours converts a time string (HH:MM or HH:MM:SS) to decimal hours
// Examples: "1:30" -> 1.5, "00:05" -> 0.083333, "2:45:30" -> 2.758333
// Returns the decimal hours and true if successful, 0 and false if the format is invalid
func timeToDecimalHours(timeStr string) (float64, bool) {
	// Handle negative times
	isNegative := false
	if strings.HasPrefix(timeStr, "-") {
		isNegative = true
		timeStr = strings.TrimPrefix(timeStr, "-")
	}
	// Remove leading + if present
	timeStr = strings.TrimPrefix(timeStr, "+")

	// Split by colon
	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}

	// Parse hours as integer. Using Atoi rather than ParseFloat rejects
	// decimal hours ("1.5:30"), NaN, and Inf in one shot — matching how
	// minutes and seconds are parsed below.
	hoursInt, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}

	// Parse minutes as integer. Atoi rejects "30.0"-style decimals that
	// ParseFloat would accept (since "30.0" == float64(int(30.0))).
	minutesInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}

	// Parse seconds if present, also integer-only.
	secondsInt := 0
	if len(parts) == 3 {
		secondsInt, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, false
		}
	}

	// Validate ranges. Hours has already been parsed as int above; the sign
	// was stripped before splitting, so negative values here would mean an
	// explicit "-30" segment.
	if hoursInt < 0 || minutesInt < 0 || minutesInt >= 60 || secondsInt < 0 || secondsInt >= 60 {
		return 0, false
	}

	// Convert to decimal hours
	decimalHours := float64(hoursInt) + float64(minutesInt)/60.0 + float64(secondsInt)/3600.0

	if isNegative {
		decimalHours = -decimalHours
	}

	return decimalHours, true
}

// parseTimeToDeadlineOffset parses a time string (e.g., "3:00 PM", "15:00") into
// a deadline offset in seconds from midnight, as used by the Beeminder API.
func parseTimeToDeadlineOffset(timeStr string) (int, error) {
	trimmed := strings.TrimSpace(timeStr)

	// Try 12-hour format first (e.g., "3:00 PM", "11:30 AM")
	t, err := time.Parse("3:04 PM", strings.ToUpper(trimmed))
	if err != nil {
		// Try 24-hour format (e.g., "15:00", "23:30")
		t, err = time.Parse("15:04", trimmed)
	}
	if err != nil {
		return 0, fmt.Errorf("invalid time format %q (expected e.g. \"3:00 PM\" or \"15:00\")", timeStr)
	}

	hour := t.Hour()
	minute := t.Minute()

	// Beeminder does not allow deadlines between 6:01 AM and 6:59 AM.
	// Allowed ranges: 12:00 AM–6:00 AM (nightowl) and 7:00 AM–11:59 PM (earlybird).
	// https://help.beeminder.com/article/14-deadline#allowed
	if hour == 6 && minute > 0 {
		return 0, fmt.Errorf("deadlines between 6:01 AM and 6:59 AM are not allowed by Beeminder")
	}

	offset := hour*3600 + minute*60

	// Beeminder deadline offsets use seconds from midnight.
	// Range: -61200 (7:00 AM) to 21600 (6:00 AM)
	// Times from 7:00-23:59 are negative offsets (before midnight)
	// Times from 0:00-6:00 are positive offsets (after midnight)
	// https://forum.beeminder.com/t/api-deadline/10666
	if hour > 6 {
		offset = offset - 24*3600
	}

	return offset, nil
}

// formatDueTime formats the deadline offset (seconds from midnight) as a time
// string. Negative offset means before midnight, positive means after midnight.
//
// The offset is normalized into the [0, 86400) range before formatting so a
// second-precision negative input like -3599 (which is 59:59 before midnight,
// i.e. 23:00:01) rounds the same way as its positive counterpart 82801 instead
// of drifting by a minute from hand-rolled hour/minute arithmetic.
func formatDueTime(deadlineOffset int) string {
	const secondsPerDay = 24 * 60 * 60
	normalized := ((deadlineOffset % secondsPerDay) + secondsPerDay) % secondsPerDay
	t := time.Unix(int64(normalized), 0).UTC()
	return t.Format("3:04 PM")
}
