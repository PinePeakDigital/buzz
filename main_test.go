package main

import (
	"os"
	"strings"
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

// TestParseFormatFlag covers the global --format extraction: default, both flag
// spellings, flag removal from args, and error cases (missing/invalid value).
func TestParseFormatFlag(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantFormat string
		wantArgs   []string
		wantErr    bool
	}{
		{"no flag defaults to table", []string{"buzz", "list"}, "table", []string{"buzz", "list"}, false},
		{"--format json (space)", []string{"buzz", "--format", "json", "list"}, "json", []string{"buzz", "list"}, false},
		{"--format=csv (equals)", []string{"buzz", "list", "--format=csv"}, "csv", []string{"buzz", "list"}, false},
		{"invalid value errors", []string{"buzz", "--format", "yaml", "list"}, "", nil, true},
		{"missing value errors", []string{"buzz", "list", "--format"}, "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, filtered, err := parseFormatFlag(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if format != tt.wantFormat {
				t.Errorf("format = %q, want %q", format, tt.wantFormat)
			}
			if len(filtered) != len(tt.wantArgs) {
				t.Fatalf("filtered args = %v, want %v", filtered, tt.wantArgs)
			}
			for i, a := range tt.wantArgs {
				if filtered[i] != a {
					t.Errorf("filtered[%d] = %q, want %q", i, filtered[i], a)
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

// TestGoalByEndOfTomorrowAtBaremin verifies that due-today goals get their baremin
// bumped by one day's worth of rate, while goals due tomorrow (or later) are
// returned unchanged.
func TestGoalByEndOfTomorrowAtBaremin(t *testing.T) {
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
			expected: "+2",
		},
		{
			name:     "due today with rate 7/week bumps +0 to +1",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+0 today", Rate: f(7), Runits: "w"},
			expected: "+1",
		},
		{
			name:     "due today with negative baremin still adds rate",
			goal:     Goal{Losedate: todayDeadline, Baremin: "-2 today", Rate: f(1), Runits: "d"},
			expected: "-1",
		},
		{
			// Goals already due tomorrow pass through with their time-window
			// suffix stripped — every row in the tomorrow view shares the
			// same horizon, so " in 1 day" is just noise.
			name:     "due tomorrow strips trailing window suffix",
			goal:     Goal{Losedate: tomorrowDeadline, Baremin: "+3 in 1 day", Rate: f(1), Runits: "d"},
			expected: "+3",
		},
		{
			name:     "due today with nil rate falls back, suffix stripped",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: nil, Runits: "d"},
			expected: "+1",
		},
		{
			// Overdue goals keep their original (overdue) losedate so the
			// OVERDUE indicator stays visible, but their Baremin window
			// suffix is still stripped.
			name:     "overdue strips trailing window suffix",
			goal:     Goal{Losedate: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC).Unix(), Baremin: "+1 today", Rate: f(1), Runits: "d"},
			expected: "+1",
		},
		{
			// Real-world scenario from a "clean" hours-valued goal: 25 minutes
			// due today, rate 1 hour/day. Tomorrow needs today's 25 minutes
			// plus another hour = 1:25.
			name:     "due today with HH:MM baremin and rate 1/day bumps by 1 hour",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25 in 8 hours", Rate: f(1), Runits: "d"},
			expected: "+01:25",
		},
		{
			name:     "due today with HH:MM baremin and rate 7/week bumps by 1 hour",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:30 today", Rate: f(7), Runits: "w"},
			expected: "+01:30",
		},
		{
			name:     "due today with HH:MM baremin and fractional rate rounds minutes",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25 today", Rate: f(0.5), Runits: "d"},
			expected: "+00:55",
		},
		{
			// Real-world "clean" goal: Beeminder returns HH:MM:SS for some
			// hour-valued goals (`+00:25:00 today` style). The bumped output
			// should preserve the HH:MM:SS format the input used.
			name:     "due today with HH:MM:SS baremin bumps and preserves format",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25:00 today", Rate: f(1), Runits: "d"},
			expected: "+01:25:00",
		},
		{
			name:     "due today with HH:MM:SS baremin with seconds bumps cleanly",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+00:25:30 today", Rate: f(1), Runits: "d"},
			expected: "+01:25:30",
		},
		{
			name:     "due today with garbage baremin falls back",
			goal:     Goal{Losedate: todayDeadline, Baremin: "garbage today", Rate: f(1), Runits: "d"},
			expected: "garbage",
		},
		{
			name:     "due today with empty runits falls back",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: f(1), Runits: ""},
			expected: "+1",
		},
		{
			name:     "due today with unknown runits falls back",
			goal:     Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: f(1), Runits: "x"},
			expected: "+1",
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
			expected: "+01:25:00",
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
			expected: "+01:25:00",
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
			expected: "+01:25:00",
		},
		{
			// Real-world steps-goal scenario: API returns dueby pre-rounded
			// to the goal's Display Precision, so we should honour the
			// formatted tomorrow delta instead of doing our own float math
			// (which would print "+10788.140000000001").
			name: "due today with dueby uses Beeminder's formatted tomorrow delta",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+2283",
				Rate:     f(8505.140000000001),
				Runits:   "d",
				Dueby: map[string]DuebyEntry{
					"20250115": {FormattedDelta: "+2283"},
					"20250116": {FormattedDelta: "+10789"},
					"20250117": {FormattedDelta: "+10789"},
				},
			},
			expected: "+10789",
		},
		{
			// Timey goals also get pre-formatted dueby entries; honour them
			// over our own colon-format reconstruction.
			name: "due today with timey dueby honours Beeminder's tomorrow formatting",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+00:25 today",
				Rate:     f(1),
				Runits:   "d",
				Dueby: map[string]DuebyEntry{
					"20250115": {FormattedDelta: "+00:25"},
					"20250116": {FormattedDelta: "+01:25"},
				},
			},
			expected: "+01:25",
		},
		{
			// Dueby with only today's entry can't tell us tomorrow's value —
			// fall back to the existing slope-based bump.
			name: "due today with single-entry dueby falls back to slope bump",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+1 today",
				Rate:     f(1),
				Runits:   "d",
				Dueby: map[string]DuebyEntry{
					"20250115": {FormattedDelta: "+1"},
				},
			},
			expected: "+2",
		},
		{
			// Dueby present but tomorrow's entry has an empty FormattedDelta
			// (defensive — shouldn't happen in practice). Fall back so we
			// still produce a useful display string.
			name: "due today with empty tomorrow FormattedDelta falls back",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+1 today",
				Rate:     f(1),
				Runits:   "d",
				Dueby: map[string]DuebyEntry{
					"20250115": {FormattedDelta: "+1"},
					"20250116": {FormattedDelta: ""},
				},
			},
			expected: "+2",
		},
		{
			// Dueby keyed only by past daystamps (defensive — Beeminder
			// normally starts at today). The deadline-aware lookup misses,
			// so we fall back instead of mistakenly using a stale entry.
			name: "due today with only-past dueby keys falls back",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+1 today",
				Rate:     f(1),
				Runits:   "d",
				Dueby: map[string]DuebyEntry{
					"20250113": {FormattedDelta: "+99"},
					"20250114": {FormattedDelta: "+99"},
				},
			},
			expected: "+2",
		},
		{
			// Dueby includes a past daystamp alongside today and tomorrow.
			// We must still pick tomorrow's entry (20250116 = "+5"), not the
			// earlier ones — i.e. don't index by sort order.
			name: "due today with past+today+tomorrow dueby still picks tomorrow",
			goal: Goal{
				Losedate: todayDeadline,
				Baremin:  "+1 today",
				Rate:     f(1),
				Runits:   "d",
				Dueby: map[string]DuebyEntry{
					"20250114": {FormattedDelta: "+99"},
					"20250115": {FormattedDelta: "+1"},
					"20250116": {FormattedDelta: "+5"},
					"20250117": {FormattedDelta: "+9"},
				},
			},
			expected: "+5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goalByEndOfTomorrowAt(tt.goal, now).baremin
			if got != tt.expected {
				t.Errorf("goalByEndOfTomorrowAt().baremin = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestTomorrowDaystampFor verifies that the daystamp lookup honours each
// goal's `deadline` shift. A positive deadline (e.g. +3h cutoff) keeps
// early-morning runs on yesterday's daystamp; a negative deadline (e.g.
// -3h, 9pm cutoff) pushes late-evening runs onto tomorrow's daystamp.
func TestTomorrowDaystampFor(t *testing.T) {
	tests := []struct {
		name     string
		deadline int
		now      time.Time
		expected string
	}{
		{
			name:     "midnight cutoff, mid-afternoon",
			deadline: 0,
			now:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
			expected: "20250116",
		},
		{
			name:     "3am cutoff, 1am run is still 'yesterday' so tomorrow is the calendar day",
			deadline: 3 * 3600,
			now:      time.Date(2025, 1, 15, 1, 0, 0, 0, time.UTC),
			expected: "20250115",
		},
		{
			name:     "9pm cutoff, 10pm run already on next Beeminder day",
			deadline: -3 * 3600,
			now:      time.Date(2025, 1, 15, 22, 0, 0, 0, time.UTC),
			expected: "20250117",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Goal{Deadline: tt.deadline}
			got := tomorrowDaystampFor(g, tt.now)
			if got != tt.expected {
				t.Errorf("tomorrowDaystampFor(deadline=%d, now=%v) = %q, want %q", tt.deadline, tt.now, got, tt.expected)
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

// TestGoalByEndOfTomorrowAtLosedate verifies that due-today goals get their
// displayed deadline advanced by one calendar day in the tomorrow view (in
// the caller's local zone, so DST transitions stay aligned), while goals
// already due tomorrow or later keep their own losedate.
func TestGoalByEndOfTomorrowAtLosedate(t *testing.T) {
	now := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
	todayDeadline := time.Date(2025, 1, 15, 17, 59, 0, 0, time.UTC).Unix()
	tomorrowDeadline := time.Date(2025, 1, 16, 17, 59, 0, 0, time.UTC).Unix()
	dayAfterTomorrowDeadline := time.Date(2025, 1, 17, 17, 59, 0, 0, time.UTC).Unix()

	tests := []struct {
		name     string
		goal     Goal
		expected int64
	}{
		{
			// User-reported real-world case: a clean goal due today at 5:59 PM
			// should show tomorrow's 5:59 PM as the deadline in `buzz tomorrow`
			// since the displayed baremin covers tomorrow. Outside DST
			// transitions this is the same as +86400.
			name:     "due today bumps deadline by one calendar day",
			goal:     Goal{Losedate: todayDeadline},
			expected: time.Date(2025, 1, 16, 17, 59, 0, 0, time.UTC).Unix(),
		},
		{
			name:     "due tomorrow keeps own losedate",
			goal:     Goal{Losedate: tomorrowDeadline},
			expected: tomorrowDeadline,
		},
		{
			name:     "due later keeps own losedate",
			goal:     Goal{Losedate: dayAfterTomorrowDeadline},
			expected: dayAfterTomorrowDeadline,
		},
		{
			// Overdue goals keep their losedate so the OVERDUE indicator
			// remains visible — bumping it would silently move the deadline
			// into the future and hide the fact that the goal has derailed.
			name:     "overdue keeps own losedate",
			goal:     Goal{Losedate: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC).Unix()},
			expected: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC).Unix(),
		},
		{
			// Edge case: losedate exactly equals now — treat as still
			// "due later today" so bumping happens. Anything strictly less
			// than now is overdue.
			name:     "losedate at exactly now still bumps",
			goal:     Goal{Losedate: now.Unix()},
			expected: time.Unix(now.Unix(), 0).In(now.Location()).AddDate(0, 0, 1).Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goalByEndOfTomorrowAt(tt.goal, now).losedate
			if got != tt.expected {
				t.Errorf("goalByEndOfTomorrowAt().losedate = %d, want %d (diff %d seconds)",
					got, tt.expected, got-tt.expected)
			}
		})
	}

	// DST boundary: in America/New_York the spring-forward jump means
	// 5:59 PM the next calendar day is only 23 hours away in absolute terms,
	// not 24. Using AddDate(0,0,1) preserves the wall-clock time, so the
	// returned losedate is +82800 seconds rather than +86400.
	t.Run("DST spring-forward preserves wall-clock", func(t *testing.T) {
		ny, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Skipf("America/New_York not available: %v", err)
		}
		// 5:59 PM the day before spring-forward (2025-03-09).
		losedate := time.Date(2025, 3, 8, 17, 59, 0, 0, ny)
		nowDST := time.Date(2025, 3, 8, 14, 0, 0, 0, ny)
		got := goalByEndOfTomorrowAt(Goal{Losedate: losedate.Unix()}, nowDST).losedate
		want := time.Date(2025, 3, 9, 17, 59, 0, 0, ny).Unix()
		if got != want {
			t.Errorf("DST goalByEndOfTomorrowAt().losedate = %d, want %d (diff %d seconds)",
				got, want, got-want)
		}
		// Sanity: a naive +86400 would land at the wrong wall-clock hour
		// (6:59 PM instead of 5:59 PM after the spring-forward).
		if losedate.Unix()+86400 == want {
			t.Errorf("DST test would also pass with naive +86400 — losedate fixture isn't actually crossing the DST boundary")
		}
	})
}

// TestGoalByEndOfTomorrowAtPairsBareminAndLosedate pins the invariant the whole
// refactor exists to guarantee: baremin and losedate are vended from a single
// due-today gate evaluation, so they move together. Either both reflect the
// one-day bump (due-later-today) or neither does (due tomorrow-or-later, or
// overdue). Reading both fields off one call makes a re-split of the gate —
// bumping one field but not the other — fail here.
func TestGoalByEndOfTomorrowAtPairsBareminAndLosedate(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	now := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
	todayDeadline := time.Date(2025, 1, 15, 23, 0, 0, 0, time.UTC).Unix()
	tomorrowDeadline := time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC).Unix()
	overdue := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC).Unix() // before now

	// Due later today: BOTH fields bump. Baremin +1 → +2, losedate advances one
	// calendar day in the local zone.
	t.Run("due later today bumps both", func(t *testing.T) {
		g := Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: f(1), Runits: "d"}
		view := goalByEndOfTomorrowAt(g, now)
		wantLosedate := time.Unix(todayDeadline, 0).In(now.Location()).AddDate(0, 0, 1).Unix()
		if view.baremin != "+2" {
			t.Errorf("baremin = %q, want %q", view.baremin, "+2")
		}
		if view.losedate != wantLosedate {
			t.Errorf("losedate = %d, want %d (advanced one day)", view.losedate, wantLosedate)
		}
	})

	// Due tomorrow already: NEITHER field bumps. Baremin only loses its window
	// suffix; losedate is unchanged.
	t.Run("due tomorrow bumps neither", func(t *testing.T) {
		g := Goal{Losedate: tomorrowDeadline, Baremin: "+3 in 1 day", Rate: f(1), Runits: "d"}
		view := goalByEndOfTomorrowAt(g, now)
		if view.baremin != "+3" {
			t.Errorf("baremin = %q, want %q (suffix stripped, not bumped)", view.baremin, "+3")
		}
		if view.losedate != tomorrowDeadline {
			t.Errorf("losedate = %d, want %d (unchanged)", view.losedate, tomorrowDeadline)
		}
	})

	// Overdue: NEITHER field bumps. The losedate stays in the past so the
	// OVERDUE indicator survives; baremin only loses its window suffix.
	t.Run("overdue bumps neither", func(t *testing.T) {
		g := Goal{Losedate: overdue, Baremin: "+1 today", Rate: f(1), Runits: "d"}
		view := goalByEndOfTomorrowAt(g, now)
		if view.baremin != "+1" {
			t.Errorf("baremin = %q, want %q (suffix stripped, not bumped)", view.baremin, "+1")
		}
		if view.losedate != overdue {
			t.Errorf("losedate = %d, want %d (unchanged, stays overdue)", view.losedate, overdue)
		}
	})
}

// TestGoalByEndOfTomorrowAtFlagsMalformedRoad pins #325: a goal whose roadall is
// malformed is flagged (roadMalformed → a "(!) " prefix on the displayed baremin),
// while a valid road and an absent road ("not populated", benign) are not. The
// flag is independent of the due-today bump gate.
func TestGoalByEndOfTomorrowAtFlagsMalformedRoad(t *testing.T) {
	now := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
	t0 := float64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	t1 := float64(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC).Unix())

	malformedRoad := [][]*float64{
		roadallRow(t0, fptr(0), nil),
		roadallRow(t1, fptr(5), fptr(1)), // both value and rate set → malformed
	}
	validRoad := [][]*float64{
		roadallRow(t0, fptr(0), nil),
		roadallRow(t1, fptr(5), nil), // value row
	}

	t.Run("malformed road sets the flag and marks the baremin", func(t *testing.T) {
		g := Goal{Baremin: "+1 today", Roadall: malformedRoad, Runits: "d"}
		view := goalByEndOfTomorrowAt(g, now)
		if !view.roadMalformed {
			t.Error("expected roadMalformed = true for a malformed roadall")
		}
		if got := view.markedBaremin(); got != "(!) +1" {
			t.Errorf("markedBaremin = %q, want %q", got, "(!) +1")
		}
	})

	t.Run("valid road is not flagged", func(t *testing.T) {
		g := Goal{Baremin: "+1 today", Roadall: validRoad, Runits: "d"}
		view := goalByEndOfTomorrowAt(g, now)
		if view.roadMalformed {
			t.Error("expected roadMalformed = false for a valid roadall")
		}
		if got := view.markedBaremin(); got != "+1" {
			t.Errorf("markedBaremin = %q, want %q (no marker)", got, "+1")
		}
	})

	t.Run("absent road is benign, not flagged", func(t *testing.T) {
		// No roadall at all → parseRoad returns (nil, nil): "not populated", not
		// malformed. The issue explicitly keeps this distinct from the error case.
		g := Goal{Baremin: "+1 today"}
		view := goalByEndOfTomorrowAt(g, now)
		if view.roadMalformed {
			t.Error("expected roadMalformed = false for an absent roadall")
		}
		if got := view.markedBaremin(); got != "+1" {
			t.Errorf("markedBaremin = %q, want %q (no marker)", got, "+1")
		}
	})

	t.Run("malformed road on a bumped (due-later-today) goal", func(t *testing.T) {
		// The gate-independence case that matters most: a due-later-today goal is
		// bumped, and the bump silently falls back to g.Rate because the road
		// won't parse. The marker warns that the bumped figure is a fallback.
		// (Pins the roadMalformed flag on the bumped return branch, which is a
		// separate struct literal from the not-bumped one.)
		todayDeadline := time.Date(2025, 1, 15, 23, 0, 0, 0, time.UTC).Unix() // later today → bumped
		g := Goal{Losedate: todayDeadline, Baremin: "+1 today", Rate: fptr(1), Runits: "d", Roadall: malformedRoad}
		view := goalByEndOfTomorrowAt(g, now)
		if !view.roadMalformed {
			t.Error("expected roadMalformed = true on the bumped branch")
		}
		// Malformed road → slopePerDayAt falls back to g.Rate (1/day): +1 → +2.
		if got := view.markedBaremin(); got != "(!) +2" {
			t.Errorf("markedBaremin = %q, want %q (bumped via g.Rate fallback)", got, "(!) +2")
		}
	})
}

// TestTomorrowMalformedLegend keeps the footnote and the cell marker in sync:
// the legend must reference the same "(!)" the marked baremin uses and name the
// bright red line, so a user seeing "(!)" can connect it to its explanation.
func TestTomorrowMalformedLegend(t *testing.T) {
	marked := tomorrowView{roadMalformed: true, baremin: "+2"}.markedBaremin()
	marker := strings.TrimSuffix(marked, "+2") // "(!) "
	if strings.TrimSpace(marker) == "" {
		t.Fatalf("expected a non-empty marker prefix, got %q", marked)
	}
	if !strings.Contains(tomorrowMalformedLegend, strings.TrimSpace(marker)) {
		t.Errorf("legend %q should reference the marker %q", tomorrowMalformedLegend, strings.TrimSpace(marker))
	}
	if !strings.Contains(tomorrowMalformedLegend, "bright red line") {
		t.Errorf("legend should name the bright red line, got %q", tomorrowMalformedLegend)
	}
}

// TestTomorrowLegendGating pins the visibility rule the footnote ships: it
// appears only when at least one displayed goal carries the "(!)" marker, and
// is suppressed otherwise (so a clean tomorrow view stays uncluttered).
func TestTomorrowLegendGating(t *testing.T) {
	viewOf := func(g Goal) tomorrowView {
		return tomorrowView{roadMalformed: g.Slug == "bad"}
	}

	if got := tomorrowLegend([]Goal{{Slug: "ok"}, {Slug: "bad"}}, viewOf); got != tomorrowMalformedLegend {
		t.Errorf("expected the legend when a shown goal is flagged, got %q", got)
	}
	if got := tomorrowLegend([]Goal{{Slug: "ok"}, {Slug: "ok2"}}, viewOf); got != "" {
		t.Errorf("expected no legend when no goal is flagged, got %q", got)
	}
	if got := tomorrowLegend(nil, viewOf); got != "" {
		t.Errorf("expected no legend for an empty goal list, got %q", got)
	}
}

// TestSortGoalsByDisplayedLosedate locks in the rule that rows render in the
// order the user actually sees in the deadline column. The tomorrow view
// bumps due-today losedates by one calendar day, which can flip the relative
// order of goals if we don't resort using the same losedateFor projection.
func TestSortGoalsByDisplayedLosedate(t *testing.T) {
	now := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)
	losedateFor := func(g Goal) int64 { return goalByEndOfTomorrowAt(g, now).losedate }

	// Goal A: due today at 11 PM → displays as tomorrow 11 PM
	// Goal B: due tomorrow at 9 AM → displays as tomorrow 9 AM (earlier)
	dueToday11PM := time.Date(2025, 1, 15, 23, 0, 0, 0, time.UTC).Unix()
	dueTomorrow9AM := time.Date(2025, 1, 16, 9, 0, 0, 0, time.UTC).Unix()

	// Pre-sort by original losedate, like SortGoals would. A comes first
	// because its original losedate is earlier — but A's *displayed* losedate
	// is later, so the resort must flip them.
	goals := []Goal{
		{Slug: "a", Losedate: dueToday11PM},
		{Slug: "b", Losedate: dueTomorrow9AM},
	}

	sortGoalsByDisplayedLosedate(goals, losedateFor)

	if goals[0].Slug != "b" || goals[1].Slug != "a" {
		t.Errorf("expected [b, a] after resorting by displayed losedate, got [%s, %s]",
			goals[0].Slug, goals[1].Slug)
	}
}

// TestParseTimeValue verifies the colon-separated time parser used by
// bumpedBaremin for HH:MM and HH:MM:SS baremin values.
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
		{"explicit plus on minutes is rejected", "1:+30", 0, false, false},
		{"explicit plus on seconds is rejected", "1:30:+45", 0, false, false},
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
