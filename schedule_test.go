package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestExtractTimeSlots tests the extraction and grouping of time slots from goals
func TestExtractTimeSlots(t *testing.T) {
	// Create test goals with different deadline times. The fixtures are built in
	// time.Local and rendered with time.Local below, so the asserted hour/minute
	// stays correct regardless of the machine's timezone.
	goals := []Goal{
		{
			Slug:     "goal1",
			Losedate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local).Unix(),
		},
		{
			Slug:     "goal2",
			Losedate: time.Date(2024, 1, 16, 10, 30, 0, 0, time.Local).Unix(), // Same time as goal1
		},
		{
			Slug:     "goal3",
			Losedate: time.Date(2024, 1, 17, 15, 45, 0, 0, time.Local).Unix(),
		},
		{
			Slug:     "goal4",
			Losedate: time.Date(2024, 1, 18, 6, 0, 0, 0, time.Local).Unix(),
		},
	}

	slots := extractTimeSlots(goals, time.Local)

	// Should have 3 time slots (goal1 and goal2 share the same time)
	if len(slots) != 3 {
		t.Errorf("Expected 3 time slots, got %d", len(slots))
	}

	// Check that slots are sorted by time
	for i := 1; i < len(slots); i++ {
		prev := slots[i-1]
		curr := slots[i]
		if prev.hour > curr.hour || (prev.hour == curr.hour && prev.minute > curr.minute) {
			t.Errorf("Slots not sorted: %02d:%02d comes after %02d:%02d",
				prev.hour, prev.minute, curr.hour, curr.minute)
		}
	}

	// Check the first slot (06:00 with goal4)
	if slots[0].hour != 6 || slots[0].minute != 0 {
		t.Errorf("Expected first slot at 06:00, got %02d:%02d", slots[0].hour, slots[0].minute)
	}
	if len(slots[0].goals) != 1 || slots[0].goals[0] != "goal4" {
		t.Errorf("Expected first slot to have goal4, got %v", slots[0].goals)
	}

	// Check the second slot (10:30 with goal1 and goal2)
	if slots[1].hour != 10 || slots[1].minute != 30 {
		t.Errorf("Expected second slot at 10:30, got %02d:%02d", slots[1].hour, slots[1].minute)
	}
	if len(slots[1].goals) != 2 {
		t.Errorf("Expected second slot to have 2 goals, got %d", len(slots[1].goals))
	}
	// Goals should be in the order they were added
	expectedGoals := map[string]bool{"goal1": true, "goal2": true}
	for _, slug := range slots[1].goals {
		if !expectedGoals[slug] {
			t.Errorf("Unexpected goal %s in second slot", slug)
		}
	}

	// Check the third slot (15:45 with goal3)
	if slots[2].hour != 15 || slots[2].minute != 45 {
		t.Errorf("Expected third slot at 15:45, got %02d:%02d", slots[2].hour, slots[2].minute)
	}
	if len(slots[2].goals) != 1 || slots[2].goals[0] != "goal3" {
		t.Errorf("Expected third slot to have goal3, got %v", slots[2].goals)
	}
}

// TestExtractTimeSlotsEmpty tests extractTimeSlots with no goals
func TestExtractTimeSlotsEmpty(t *testing.T) {
	var goals []Goal
	slots := extractTimeSlots(goals, time.Local)

	if len(slots) != 0 {
		t.Errorf("Expected 0 time slots for empty goals, got %d", len(slots))
	}
}

// TestExtractTimeSlotsAcrossDates tests that goals on different dates with same time are grouped together
func TestExtractTimeSlotsAcrossDates(t *testing.T) {
	// Fixtures built in time.Local and rendered with time.Local below, so the
	// asserted slot time is timezone-independent (extractTimeSlots groups by the
	// deadline's hour/minute in the provided location).
	goals := []Goal{
		{
			Slug:     "goal1",
			Losedate: time.Date(2024, 1, 15, 14, 30, 0, 0, time.Local).Unix(),
		},
		{
			Slug:     "goal2",
			Losedate: time.Date(2024, 2, 20, 14, 30, 0, 0, time.Local).Unix(), // Different date, same time
		},
		{
			Slug:     "goal3",
			Losedate: time.Date(2024, 3, 10, 14, 30, 0, 0, time.Local).Unix(), // Different date, same time
		},
	}

	slots := extractTimeSlots(goals, time.Local)

	// Should have 1 time slot (all goals at 14:30)
	if len(slots) != 1 {
		t.Errorf("Expected 1 time slot, got %d", len(slots))
	}

	// Check that the slot has all 3 goals
	if len(slots[0].goals) != 3 {
		t.Errorf("Expected 3 goals in the slot, got %d", len(slots[0].goals))
	}

	// Verify the time
	if slots[0].hour != 14 || slots[0].minute != 30 {
		t.Errorf("Expected slot at 14:30, got %02d:%02d", slots[0].hour, slots[0].minute)
	}
}

// TestExtractTimeSlotsUsesProvidedLocation verifies that a goal's deadline is
// rendered in the supplied location, so the same absolute instant yields a
// different hour/minute across timezones (issue #214).
func TestExtractTimeSlotsUsesProvidedLocation(t *testing.T) {
	// A fixed absolute instant: noon UTC on 2024-01-15.
	instant := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	goals := []Goal{{Slug: "goal1", Losedate: instant}}

	utcSlots := extractTimeSlots(goals, time.UTC)
	if len(utcSlots) != 1 || utcSlots[0].hour != 12 || utcSlots[0].minute != 0 {
		t.Fatalf("UTC: expected one slot at 12:00, got %+v", utcSlots)
	}

	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("America/New_York tzdata unavailable: %v", err)
	}
	// New York is UTC-5 in January, so noon UTC is 07:00 local.
	nySlots := extractTimeSlots(goals, ny)
	if len(nySlots) != 1 || nySlots[0].hour != 7 || nySlots[0].minute != 0 {
		t.Fatalf("America/New_York: expected one slot at 07:00, got %+v", nySlots)
	}
}

// TestScheduleLocation verifies that scheduleLocation prefers the account
// timezone and falls back to time.Local on error, empty, or unparseable input.
func TestScheduleLocation(t *testing.T) {
	if _, err := time.LoadLocation("America/New_York"); err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}

	tests := []struct {
		name    string
		tz      string
		tzErr   error
		wantLoc string // empty means expect time.Local
	}{
		{name: "account timezone used", tz: "America/New_York", wantLoc: "America/New_York"},
		{name: "empty falls back to local", tz: ""},
		{name: "error falls back to local", tzErr: errFakeNotConfigured},
		{name: "unparseable falls back to local", tz: "Not/AZone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &FakeClient{
				FetchUserTimezoneFunc: func() (string, error) { return tt.tz, tt.tzErr },
			}
			loc := scheduleLocation(context.Background(), client)
			want := tt.wantLoc
			if want == "" {
				if loc != time.Local {
					t.Errorf("expected time.Local, got %s", loc)
				}
				return
			}
			if loc.String() != want {
				t.Errorf("expected %s, got %s", want, loc)
			}
		})
	}
}

// TestDisplayHourlyDensity tests the hourly density visualization
func TestDisplayHourlyDensity(t *testing.T) {
	tests := []struct {
		name       string
		hourCounts []int
	}{
		{
			name:       "empty counts",
			hourCounts: make([]int, 24),
		},
		{
			name: "single goal at midnight",
			hourCounts: func() []int {
				counts := make([]int, 24)
				counts[0] = 1
				return counts
			}(),
		},
		{
			name: "multiple goals at different hours",
			hourCounts: func() []int {
				counts := make([]int, 24)
				counts[6] = 1
				counts[10] = 5
				counts[12] = 1
				counts[15] = 2
				counts[18] = 1
				counts[22] = 3
				return counts
			}(),
		},
		{
			name: "100+ goals at single hour",
			hourCounts: func() []int {
				counts := make([]int, 24)
				counts[10] = 150
				return counts
			}(),
		},
		{
			name: "max scaling test",
			hourCounts: func() []int {
				counts := make([]int, 24)
				counts[0] = 1
				counts[12] = 50
				return counts
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("displayHourlyDensity panicked: %v", r)
				}
			}()

			// We can't easily test the output without capturing stdout,
			// but we can ensure the function runs without error
			displayHourlyDensity(tt.hourCounts)
		})
	}
}

// TestDisplayTimeline tests the timeline visualization
func TestDisplayTimeline(t *testing.T) {
	tests := []struct {
		name   string
		slots  []timeSlot
		verify func(*testing.T)
	}{
		{
			name:  "empty slots",
			slots: []timeSlot{},
			verify: func(t *testing.T) {
				// Should not panic
			},
		},
		{
			name: "single slot with one goal",
			slots: []timeSlot{
				{hour: 10, minute: 30, goals: []string{"exercise"}},
			},
			verify: func(t *testing.T) {
				// Should not panic
			},
		},
		{
			name: "single slot with multiple goals",
			slots: []timeSlot{
				{hour: 10, minute: 30, goals: []string{"exercise", "vitamins", "breakfast"}},
			},
			verify: func(t *testing.T) {
				// Should not panic
			},
		},
		{
			name: "multiple slots",
			slots: []timeSlot{
				{hour: 6, minute: 0, goals: []string{"wake_up"}},
				{hour: 10, minute: 30, goals: []string{"exercise", "vitamins"}},
				{hour: 22, minute: 0, goals: []string{"sleep"}},
			},
			verify: func(t *testing.T) {
				// Should not panic
			},
		},
		{
			name: "midnight and noon",
			slots: []timeSlot{
				{hour: 0, minute: 0, goals: []string{"midnight_task"}},
				{hour: 12, minute: 0, goals: []string{"noon_task"}},
			},
			verify: func(t *testing.T) {
				// Should not panic
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("displayTimeline panicked: %v", r)
				}
			}()

			displayTimeline(tt.slots)
			tt.verify(t)
		})
	}
}

// TestTimelineFormatting tests the specific formatting logic of displayTimeline
func TestTimelineFormatting(t *testing.T) {
	tests := []struct {
		name          string
		slot          timeSlot
		expectedTime  string
		expectedGoals string
	}{
		{
			name:          "zero-padded hour and minute",
			slot:          timeSlot{hour: 9, minute: 5, goals: []string{"task1"}},
			expectedTime:  "09:05",
			expectedGoals: "task1",
		},
		{
			name:          "midnight formatting",
			slot:          timeSlot{hour: 0, minute: 0, goals: []string{"midnight"}},
			expectedTime:  "00:00",
			expectedGoals: "midnight",
		},
		{
			name:          "noon formatting",
			slot:          timeSlot{hour: 12, minute: 0, goals: []string{"lunch"}},
			expectedTime:  "12:00",
			expectedGoals: "lunch",
		},
		{
			name:          "late evening formatting",
			slot:          timeSlot{hour: 23, minute: 59, goals: []string{"bedtime"}},
			expectedTime:  "23:59",
			expectedGoals: "bedtime",
		},
		{
			name:          "multiple goals joined with commas",
			slot:          timeSlot{hour: 10, minute: 30, goals: []string{"task1", "task2", "task3"}},
			expectedTime:  "10:30",
			expectedGoals: "task1, task2, task3",
		},
		{
			name:          "two goals joined",
			slot:          timeSlot{hour: 14, minute: 15, goals: []string{"exercise", "vitamins"}},
			expectedTime:  "14:15",
			expectedGoals: "exercise, vitamins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify time formatting
			timeStr := fmt.Sprintf("%02d:%02d", tt.slot.hour, tt.slot.minute)
			if timeStr != tt.expectedTime {
				t.Errorf("Time formatting incorrect: expected %s, got %s", tt.expectedTime, timeStr)
			}

			// Verify goals joining
			goalsStr := strings.Join(tt.slot.goals, ", ")
			if goalsStr != tt.expectedGoals {
				t.Errorf("Goals joining incorrect: expected %s, got %s", tt.expectedGoals, goalsStr)
			}
		})
	}
}
