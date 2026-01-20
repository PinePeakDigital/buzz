package main

import (
	"testing"
	"time"
)

// TestExtractTimeSlots tests the extraction and grouping of time slots from goals
func TestExtractTimeSlots(t *testing.T) {
	// Create test goals with different deadline times
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	goals := []Goal{
		{
			Slug:     "goal1",
			Losedate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix(),
		},
		{
			Slug:     "goal2",
			Losedate: time.Date(2024, 1, 16, 10, 30, 0, 0, time.UTC).Unix(), // Same time as goal1
		},
		{
			Slug:     "goal3",
			Losedate: time.Date(2024, 1, 17, 15, 45, 0, 0, time.UTC).Unix(),
		},
		{
			Slug:     "goal4",
			Losedate: time.Date(2024, 1, 18, 6, 0, 0, 0, time.UTC).Unix(),
		},
	}

	slots := extractTimeSlots(goals)

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

	_ = now // Silence unused variable warning
}

// TestExtractTimeSlotsEmpty tests extractTimeSlots with no goals
func TestExtractTimeSlotsEmpty(t *testing.T) {
	var goals []Goal
	slots := extractTimeSlots(goals)

	if len(slots) != 0 {
		t.Errorf("Expected 0 time slots for empty goals, got %d", len(slots))
	}
}

// TestExtractTimeSlotsAcrossDates tests that goals on different dates with same time are grouped together
func TestExtractTimeSlotsAcrossDates(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "goal1",
			Losedate: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC).Unix(),
		},
		{
			Slug:     "goal2",
			Losedate: time.Date(2024, 2, 20, 14, 30, 0, 0, time.UTC).Unix(), // Different date, same time
		},
		{
			Slug:     "goal3",
			Losedate: time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC).Unix(), // Different date, same time
		},
	}

	slots := extractTimeSlots(goals)

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
