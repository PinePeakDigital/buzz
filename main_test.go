package main

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestParseTimeToDeadlineOffset(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{
			name:     "3:00 PM in 12-hour format",
			input:    "3:00 PM",
			expected: -32400, // 15*3600 - 24*3600
		},
		{
			name:     "11:30 AM in 12-hour format",
			input:    "11:30 AM",
			expected: -45000, // (11*3600 + 30*60) - 24*3600
		},
		{
			name:     "midnight 12:00 AM",
			input:    "12:00 AM",
			expected: 0, // midnight = 0
		},
		{
			name:     "6:00 AM",
			input:    "6:00 AM",
			expected: 21600, // 6*3600
		},
		{
			name:    "6:30 AM rejected",
			input:   "6:30 AM",
			wantErr: true,
		},
		{
			name:     "7:00 AM wraps negative",
			input:    "7:00 AM",
			expected: -61200, // 7*3600 - 24*3600
		},
		{
			name:     "15:00 in 24-hour format",
			input:    "15:00",
			expected: -32400,
		},
		{
			name:     "23:30 in 24-hour format",
			input:    "23:30",
			expected: -1800, // (23*3600 + 30*60) - 24*3600
		},
		{
			name:     "3:00 am lowercase",
			input:    "3:00 am",
			expected: 10800, // 3*3600
		},
		{
			name:    "invalid time",
			input:   "not-a-time",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeToDeadlineOffset(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTimeToDeadlineOffset(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTimeToDeadlineOffset(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parseTimeToDeadlineOffset(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNoColorFlag tests that the --no-color flag is properly parsed
func TestNoColorFlag(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectNoColor bool
		expectedArgs  []string
	}{
		{
			name:          "no flag",
			args:          []string{"buzz", "next"},
			expectNoColor: false,
			expectedArgs:  []string{"buzz", "next"},
		},
		{
			name:          "with --no-color before command",
			args:          []string{"buzz", "--no-color", "next"},
			expectNoColor: true,
			expectedArgs:  []string{"buzz", "next"},
		},
		{
			name:          "with --no-color after command",
			args:          []string{"buzz", "next", "--no-color"},
			expectNoColor: true,
			expectedArgs:  []string{"buzz", "next"},
		},
		{
			name:          "with --no-color and multiple args",
			args:          []string{"buzz", "--no-color", "add", "mygoal", "5"},
			expectNoColor: true,
			expectedArgs:  []string{"buzz", "add", "mygoal", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args and color profile
			origArgs := os.Args
			origProfile := lipgloss.ColorProfile()
			defer func() {
				os.Args = origArgs
				lipgloss.SetColorProfile(origProfile)
			}()

			// Set test args
			os.Args = tt.args

			// Process the --no-color flag using the shared function
			noColor, filteredArgs := parseNoColorFlag(os.Args)
			os.Args = filteredArgs

			if noColor {
				lipgloss.SetColorProfile(termenv.Ascii)
			}

			// Verify results
			if noColor != tt.expectNoColor {
				t.Errorf("Expected noColor=%v, got noColor=%v", tt.expectNoColor, noColor)
			}

			if len(os.Args) != len(tt.expectedArgs) {
				t.Errorf("Expected args length %d, got %d", len(tt.expectedArgs), len(os.Args))
			}

			for i, arg := range tt.expectedArgs {
				if i >= len(os.Args) || os.Args[i] != arg {
					t.Errorf("Expected arg[%d]=%q, got %q", i, arg, os.Args[i])
				}
			}

			// Verify color profile
			if tt.expectNoColor {
				if lipgloss.ColorProfile() != termenv.Ascii {
					t.Errorf("Expected Ascii color profile when --no-color is set, got %v", lipgloss.ColorProfile())
				}
			}
		})
	}
}

// TestDueFiltersSkipEndValueReached verifies that the today and tomorrow filters
// exclude goals whose end value has already been reached — those goals can show
// a negative baremin and shouldn't be surfaced as due.
func TestDueFiltersSkipEndValueReached(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	// Fixed reference time so the test is deterministic across midnight boundaries.
	now := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
	todayDeadline := time.Date(2025, 1, 15, 23, 0, 0, 0, time.UTC).Unix()
	tomorrowDeadline := time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC).Unix()

	tests := []struct {
		name           string
		goal           Goal
		todayExpect    bool
		tomorrowExpect bool
	}{
		{
			// Due-today goals are also surfaced in the tomorrow view: if the
			// user does nothing today the goal carries into tomorrow.
			name:           "do-more goal due today, not yet reached",
			goal:           Goal{Losedate: todayDeadline, Dir: 1, Curval: f(50), Goalval: f(100)},
			todayExpect:    true,
			tomorrowExpect: true,
		},
		{
			name:           "do-more goal due today, end value reached",
			goal:           Goal{Losedate: todayDeadline, Dir: 1, Curval: f(120), Goalval: f(100)},
			todayExpect:    false,
			tomorrowExpect: false,
		},
		{
			name:           "do-more goal due tomorrow, not yet reached",
			goal:           Goal{Losedate: tomorrowDeadline, Dir: 1, Curval: f(50), Goalval: f(100)},
			todayExpect:    false,
			tomorrowExpect: true,
		},
		{
			name:           "do-more goal due tomorrow, end value reached",
			goal:           Goal{Losedate: tomorrowDeadline, Dir: 1, Curval: f(120), Goalval: f(100)},
			todayExpect:    false,
			tomorrowExpect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDueTodayFilterAt(tt.goal, now); got != tt.todayExpect {
				t.Errorf("isDueTodayFilterAt = %v, want %v", got, tt.todayExpect)
			}
			if got := isDueTomorrowFilterAt(tt.goal, now); got != tt.tomorrowExpect {
				t.Errorf("isDueTomorrowFilterAt = %v, want %v", got, tt.tomorrowExpect)
			}
		})
	}
}

// TestRatePerDay verifies rate conversion across the runits Beeminder reports.
func TestRatePerDay(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		runits   string
		expected float64
	}{
		{"per day stays the same", 2, "d", 2},
		{"per week divides by 7", 7, "w", 1},
		{"per month divides by 30", 30, "m", 1},
		{"per year divides by 365", 365, "y", 1},
		{"per hour multiplies by 24", 0.5, "h", 12},
		{"unknown unit passes through", 3, "x", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ratePerDay(tt.rate, tt.runits)
			if got != tt.expected {
				t.Errorf("ratePerDay(%v, %q) = %v, want %v", tt.rate, tt.runits, got, tt.expected)
			}
		})
	}
}

// TestBareminByEndOfTomorrowAt verifies that due-today goals get their baremin
// bumped by one day's worth of rate, while goals due tomorrow (or later) are
// returned unchanged.
func TestBareminByEndOfTomorrowAt(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	now := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
	todayDeadline := time.Date(2025, 1, 15, 23, 0, 0, 0, time.UTC).Unix()
	tomorrowDeadline := time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC).Unix()

	tests := []struct {
		name     string
		goal     Goal
		expected string
	}{
		{
			name:     "due today with rate 1/day bumps +1 to +2",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+1 in 0 days", Rate: f(1), Runits: "d"},
			expected: "+2 in 1 day",
		},
		{
			name:     "due today with rate 7/week bumps +0 to +1",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+0 today", Rate: f(7), Runits: "w"},
			expected: "+1 in 1 day",
		},
		{
			name:     "due today with negative baremin still adds rate",
			goal:     Goal{Losedate: todayDeadline, Baremin: "-2 today", Rate: f(1), Runits: "d"},
			expected: "-1 in 1 day",
		},
		{
			name:     "due tomorrow is unchanged",
			goal:     Goal{Losedate: tomorrowDeadline, Baremin: "+3 in 1 day", Rate: f(1), Runits: "d"},
			expected: "+3 in 1 day",
		},
		{
			name:     "due today with nil rate is unchanged",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: nil, Runits: "d"},
			expected: "+1 today",
		},
		{
			// Real-world scenario from a "clean" hours-valued goal: 25 minutes
			// due today, rate 1 hour/day. Tomorrow needs today's 25 minutes
			// plus another hour = 1:25.
			name:     "due today with HH:MM baremin and rate 1/day bumps by 1 hour",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25 in 8 hours", Rate: f(1), Runits: "d"},
			expected: "+01:25 in 1 day",
		},
		{
			name:     "due today with HH:MM baremin and rate 7/week bumps by 1 hour",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:30 today", Rate: f(7), Runits: "w"},
			expected: "+01:30 in 1 day",
		},
		{
			name:     "due today with HH:MM baremin and fractional rate rounds minutes",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25 today", Rate: f(0.5), Runits: "d"},
			expected: "+00:55 in 1 day",
		},
		{
			// Real-world "clean" goal: Beeminder returns HH:MM:SS for some
			// hour-valued goals (`+00:25:00 today` style). The bumped output
			// should preserve the HH:MM:SS format the input used.
			name:     "due today with HH:MM:SS baremin bumps and preserves format",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25:00 today", Rate: f(1), Runits: "d"},
			expected: "+01:25:00 in 1 day",
		},
		{
			name:     "due today with HH:MM:SS baremin with seconds bumps cleanly",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25:30 today", Rate: f(1), Runits: "d"},
			expected: "+01:25:30 in 1 day",
		},
		{
			name:     "due today with garbage baremin falls back",
			goal:     Goal{Losedate: todayDeadline, Baremin: "garbage today", Rate: f(1), Runits: "d"},
			expected: "garbage today",
		},
		{
			name:     "due today with empty runits falls back",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: f(1), Runits: ""},
			expected: "+1 today",
		},
		{
			name:     "due today with unknown runits falls back",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: f(1), Runits: "x"},
			expected: "+1 today",
		},
		{
			// Real-world "clean" scenario: g.Rate reports the end-of-graph
			// rate (0.1 h/day) but the current segment is 1 h/day. The bump
			// must use the current segment, not g.Rate, so today's 25 min →
			// tomorrow's 1:25:00, not 0:31:00.
			name: "due today with piecewise roadall uses current segment, not end-of-graph rate",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+00:25:00 today",
				Rate:     f(0.1),
				Runits:   "d",
				Roadall: piecewiseRoadall(
					// Start anchor: 2024-12-01 at value 0
					time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC).Unix(), 0,
					// First segment ends 2025-02-01 at rate 1 h/day
					float64(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC).Unix()), 1.0,
					// Second segment runs to 2025-12-31 at 0.1 h/day
					float64(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC).Unix()), 0.1,
				),
			},
			expected: "+01:25:00 in 1 day",
		},
		{
			// The goal is in its slower segment; use that slope.
			name: "due today picks slope from the later piecewise segment",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+00:25:00 today",
				Rate:     f(1),
				Runits:   "d",
				Roadall: piecewiseRoadall(
					time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC).Unix(), 0,
					// First segment ends 2025-01-10 (before todayDeadline) at 0.1 h/day
					float64(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC).Unix()), 0.1,
					// Second segment ends 2025-12-31 (after todayDeadline) at 1 h/day
					float64(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC).Unix()), 1.0,
				),
			},
			expected: "+01:25:00 in 1 day",
		},
		{
			// Short roadall (just the start anchor) — fall back to g.Rate.
			name: "due today with short roadall falls back to g.Rate",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+00:25:00 today",
				Rate:     f(1),
				Runits:   "d",
				Roadall:  piecewiseRoadall(time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC).Unix(), 0),
			},
			expected: "+01:25:00 in 1 day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bareminByEndOfTomorrowAt(tt.goal, now)
			if got != tt.expected {
				t.Errorf("bareminByEndOfTomorrowAt = %q, want %q", got, tt.expected)
			}
		})
	}
}

// piecewiseRoadall builds a roadall matrix from a start anchor (t, v) followed
// by zero or more (t, r) rate-segment pairs. Mirrors the rate-only form
// Beeminder typically emits.
func piecewiseRoadall(startT int64, startV float64, segments ...float64) [][]*float64 {
	if len(segments)%2 != 0 {
		panic("piecewiseRoadall: segments must be (t, rate) pairs")
	}
	t := float64(startT)
	v := startV
	rows := [][]*float64{{&t, &v, nil}}
	for i := 0; i < len(segments); i += 2 {
		segT := segments[i]
		segR := segments[i+1]
		rows = append(rows, []*float64{&segT, nil, &segR})
	}
	return rows
}

// TestRoadallSlopePerDayAt verifies the segment-resolving helper that powers
// piecewise-aware baremin bumping.
func TestRoadallSlopePerDayAt(t *testing.T) {
	target := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
	startT := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC).Unix()
	segEnd1 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC).Unix()
	segEnd2 := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC).Unix()

	tests := []struct {
		name     string
		goal     Goal
		expected float64
		ok       bool
	}{
		{
			name: "empty roadall returns false",
			goal: Goal{Runits: "d"},
			ok:   false,
		},
		{
			name: "start-anchor only returns false",
			goal: Goal{Runits: "d", Roadall: piecewiseRoadall(startT, 0)},
			ok:   false,
		},
		{
			name:     "first segment (target before segEnd1) uses segment 1 rate",
			goal:     Goal{Runits: "d", Roadall: piecewiseRoadall(startT, 0, float64(segEnd1), 1.0, float64(segEnd2), 0.1)},
			expected: 1.0,
			ok:       true,
		},
		{
			name:     "second segment (target after segEnd1, before segEnd2) uses segment 2 rate",
			goal:     Goal{Runits: "d", Roadall: piecewiseRoadall(startT, 0, float64(target.Unix()-1), 0.1, float64(segEnd2), 1.0)},
			expected: 1.0,
			ok:       true,
		},
		{
			name:     "weekly runits converts to per-day",
			goal:     Goal{Runits: "w", Roadall: piecewiseRoadall(startT, 0, float64(segEnd1), 7.0)},
			expected: 1.0,
			ok:       true,
		},
		{
			name: "value-only segment computes slope from Δv/Δt",
			goal: Goal{
				Runits: "d",
				Roadall: [][]*float64{
					floatPtrRow(float64(startT), 0, math.NaN()),
					// Segment ends at segEnd1 with value 30 (≈ 0.477 / day over 63 days)
					floatPtrRow(float64(segEnd1), 30, math.NaN()),
				},
			},
			expected: 30.0 / float64(segEnd1-startT) * 86400.0,
			ok:       true,
		},
		{
			name: "target past last segment returns false",
			goal: Goal{
				Runits: "d",
				// Segment ending before target — target is "past goal end".
				Roadall: piecewiseRoadall(startT, 0, float64(target.Unix()-86400), 1.0),
			},
			ok: false,
		},
		{
			// A malformed boundary row (missing time) makes the road
			// ambiguous — must fail fast rather than silently jumping to the
			// next segment, which would pick the wrong slope.
			name: "malformed boundary row fails fast",
			goal: Goal{
				Runits: "d",
				Roadall: [][]*float64{
					floatPtrRow(float64(startT), 0, math.NaN()),
					floatPtrRow(math.NaN(), math.NaN(), 0.1),                 // malformed: no time
					floatPtrRow(float64(target.Unix()+86400), math.NaN(), 1), // would otherwise be selected
				},
			},
			ok: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t2 *testing.T) {
			got, ok := roadallSlopePerDayAt(tt.goal, target)
			if ok != tt.ok {
				t2.Fatalf("roadallSlopePerDayAt ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if math.Abs(got-tt.expected) > 1e-9 {
				t2.Errorf("roadallSlopePerDayAt = %v, want %v", got, tt.expected)
			}
		})
	}
}

// floatPtrRow builds a [t, v, r] row. NaN signals "this field is nil" so we
// can describe value-only or rate-only rows inline in the table.
func floatPtrRow(t, v, r float64) []*float64 {
	row := []*float64{nil, nil, nil}
	if !math.IsNaN(t) {
		row[0] = &t
	}
	if !math.IsNaN(v) {
		row[1] = &v
	}
	if !math.IsNaN(r) {
		row[2] = &r
	}
	return row
}

// TestParseTimeValue verifies the colon-separated time parser used by
// bareminByEndOfTomorrowAt for HH:MM and HH:MM:SS baremin values.
func TestParseTimeValue(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSeconds int
		expectedHasSec  bool
		ok              bool
	}{
		{"HH:MM zero", "0:00", 0, false, true},
		{"HH:MM twenty-five minutes", "00:25", 25 * 60, false, true},
		{"HH:MM one and a half hours", "1:30", 90 * 60, false, true},
		{"HH:MM single digit hours", "9:15", 9*3600 + 15*60, false, true},
		{"HH:MM double digit hours", "12:30", 12*3600 + 30*60, false, true},
		{"HH:MM negative", "-0:15", -15 * 60, false, true},
		{"HH:MM:SS zero", "0:00:00", 0, true, true},
		{"HH:MM:SS twenty-five minutes", "00:25:00", 25 * 60, true, true},
		{"HH:MM:SS with seconds", "01:25:30", 1*3600 + 25*60 + 30, true, true},
		{"HH:MM:SS negative", "-0:00:15", -15, true, true},
		{"four parts is rejected", "1:30:00:00", 0, false, false},
		{"missing minutes is rejected", "1", 0, false, false},
		{"non-numeric is rejected", "ab:cd", 0, false, false},
		{"out-of-range minutes is rejected", "1:75", 0, false, false},
		{"out-of-range seconds is rejected", "1:30:75", 0, false, false},
		{"negative minutes field is rejected", "1:-05", 0, false, false},
		{"negative seconds field is rejected", "1:30:-05", 0, false, false},
		{"double-negative is rejected", "--1:30", 0, false, false},
		{"minutes at boundary (60) is rejected", "1:60", 0, false, false},
		{"seconds at boundary (60) is rejected", "1:30:60", 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSeconds, gotHasSec, ok := parseTimeValue(tt.input)
			if ok != tt.ok {
				t.Fatalf("parseTimeValue(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if !ok {
				return
			}
			if gotSeconds != tt.expectedSeconds {
				t.Errorf("parseTimeValue(%q) seconds = %d, want %d", tt.input, gotSeconds, tt.expectedSeconds)
			}
			if gotHasSec != tt.expectedHasSec {
				t.Errorf("parseTimeValue(%q) includeSeconds = %v, want %v", tt.input, gotHasSec, tt.expectedHasSec)
			}
		})
	}
}

// TestFormatTimeValue verifies the colon-separated formatter matches
// Beeminder's zero-padded baremin style with a leading sign, in both HH:MM and
// HH:MM:SS variants.
func TestFormatTimeValue(t *testing.T) {
	tests := []struct {
		seconds        int
		includeSeconds bool
		expected       string
	}{
		{0, false, "+00:00"},
		{25 * 60, false, "+00:25"},
		{85 * 60, false, "+01:25"},
		{605 * 60, false, "+10:05"},
		{-15 * 60, false, "-00:15"},
		// When dropping seconds, round to the nearest minute rather than
		// truncating — 30 seconds past the minute rounds up to the next.
		{25*60 + 30, false, "+00:26"},
		{25*60 + 29, false, "+00:25"},
		{-15*60 - 30, false, "-00:16"},
		{0, true, "+00:00:00"},
		{25 * 60, true, "+00:25:00"},
		{85 * 60, true, "+01:25:00"},
		{85*60 + 30, true, "+01:25:30"},
		{-15, true, "-00:00:15"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatTimeValue(tt.seconds, tt.includeSeconds)
			if got != tt.expected {
				t.Errorf("formatTimeValue(%d, %v) = %q, want %q", tt.seconds, tt.includeSeconds, got, tt.expected)
			}
		})
	}
}
