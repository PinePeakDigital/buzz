package main

import (
	"testing"
)

func TestSortGoalsBySlug(t *testing.T) {
	goals := []Goal{
		{Slug: "zebra", Title: "Zebra Goal"},
		{Slug: "apple", Title: "Apple Goal"},
		{Slug: "mango", Title: "Mango Goal"},
		{Slug: "banana", Title: "Banana Goal"},
	}

	SortGoalsBySlug(goals)

	expected := []string{"apple", "banana", "mango", "zebra"}
	for i, goal := range goals {
		if goal.Slug != expected[i] {
			t.Errorf("Expected slug %s at index %d, got %s", expected[i], i, goal.Slug)
		}
	}
}

func TestReviewModelNavigation(t *testing.T) {
	goals := []Goal{
		{Slug: "goal1", Title: "First Goal"},
		{Slug: "goal2", Title: "Second Goal"},
		{Slug: "goal3", Title: "Third Goal"},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)

	// Test initial state
	if m.current != 0 {
		t.Errorf("Expected initial current to be 0, got %d", m.current)
	}

	if len(m.goals) != 3 {
		t.Errorf("Expected 3 goals, got %d", len(m.goals))
	}
}

func TestReviewModelInit(t *testing.T) {
	goals := []Goal{
		{Slug: "test", Title: "Test Goal"},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	cmd := m.Init()

	if cmd != nil {
		t.Error("Expected Init() to return nil")
	}
}

func TestReviewModelView(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "Test Goal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view contains expected content
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Check for goal counter
	expectedCounter := "Goal 1 of 1"
	if !containsString(view, expectedCounter) {
		t.Errorf("Expected view to contain '%s'", expectedCounter)
	}

	// Check for goal slug
	if !containsString(view, "testgoal") {
		t.Error("Expected view to contain goal slug")
	}
}

func TestReviewModelEmptyGoals(t *testing.T) {
	goals := []Goal{}
	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	expectedMessage := "No goals to review"
	if !containsString(view, expectedMessage) {
		t.Errorf("Expected view to contain '%s'", expectedMessage)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
