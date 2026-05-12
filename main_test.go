package main

import (
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

	// Pick a losedate squarely inside "today" and one inside "tomorrow"
	// (filters use time.Now() internally, so we anchor relative to that).
	now := time.Now()
	todayDeadline := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location()).Unix()
	tomorrowDeadline := time.Date(now.Year(), now.Month(), now.Day()+1, 12, 0, 0, 0, now.Location()).Unix()

	tests := []struct {
		name           string
		goal           Goal
		todayExpect    bool
		tomorrowExpect bool
	}{
		{
			name:           "do-more goal due today, not yet reached",
			goal:           Goal{Losedate: todayDeadline, Dir: 1, Curval: f(50), Goalval: f(100)},
			todayExpect:    true,
			tomorrowExpect: false,
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
			if got := isDueTodayFilter(tt.goal); got != tt.todayExpect {
				t.Errorf("isDueTodayFilter = %v, want %v", got, tt.todayExpect)
			}
			if got := isDueTomorrowFilter(tt.goal); got != tt.tomorrowExpect {
				t.Errorf("isDueTomorrowFilter = %v, want %v", got, tt.tomorrowExpect)
			}
		})
	}
}
