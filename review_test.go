package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestReviewModelNavigationForward(t *testing.T) {
	goals := []Goal{
		{Slug: "goal1", Title: "First Goal"},
		{Slug: "goal2", Title: "Second Goal"},
		{Slug: "goal3", Title: "Third Goal"},
	}
	config := &Config{Username: "testuser", AuthToken: "testtoken"}
	m := initialReviewModel(goals, config)

	// Test moving forward from first goal
	if m.current != 0 {
		t.Errorf("Expected initial current to be 0, got %d", m.current)
	}

	// Simulate pressing right arrow
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updatedModel.(reviewModel)

	if m.current != 1 {
		t.Errorf("Expected current to be 1 after right key, got %d", m.current)
	}

	// Move forward again
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updatedModel.(reviewModel)

	if m.current != 2 {
		t.Errorf("Expected current to be 2 after second right key, got %d", m.current)
	}

	// Test boundary - should not go past last goal
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updatedModel.(reviewModel)

	if m.current != 2 {
		t.Errorf("Expected current to stay at 2 at boundary, got %d", m.current)
	}
}

func TestReviewModelNavigationBackward(t *testing.T) {
	goals := []Goal{
		{Slug: "goal1", Title: "First Goal"},
		{Slug: "goal2", Title: "Second Goal"},
		{Slug: "goal3", Title: "Third Goal"},
	}
	config := &Config{Username: "testuser", AuthToken: "testtoken"}
	m := initialReviewModel(goals, config)
	m.current = 2 // Start at last goal

	// Simulate pressing left arrow
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updatedModel.(reviewModel)

	if m.current != 1 {
		t.Errorf("Expected current to be 1 after left key, got %d", m.current)
	}

	// Move backward again
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updatedModel.(reviewModel)

	if m.current != 0 {
		t.Errorf("Expected current to be 0 after second left key, got %d", m.current)
	}

	// Test boundary - should not go below 0
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updatedModel.(reviewModel)

	if m.current != 0 {
		t.Errorf("Expected current to stay at 0 at boundary, got %d", m.current)
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
	if !strings.Contains(view, expectedCounter) {
		t.Errorf("Expected view to contain '%s'", expectedCounter)
	}

	// Check for goal slug
	if !strings.Contains(view, "testgoal") {
		t.Error("Expected view to contain goal slug")
	}

	// Check for deadline display
	if !strings.Contains(view, "Deadline:") {
		t.Error("Expected view to contain 'Deadline:' label")
	}
}

func TestReviewModelViewDeadlineFormat(t *testing.T) {
	// Use a known timestamp: 1234567890
	// This is Feb 13, 2009 in local timezone (exact time depends on system timezone)
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

	// Check that deadline is formatted correctly
	// Time displayed is in local system timezone, so we only check date components
	// that should be consistent across timezones
	if !strings.Contains(view, "2009") {
		t.Error("Expected view to contain year '2009' from deadline")
	}

	if !strings.Contains(view, "Feb") {
		t.Error("Expected view to contain month 'Feb' from deadline")
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
	if !strings.Contains(view, expectedMessage) {
		t.Errorf("Expected view to contain '%s'", expectedMessage)
	}
}
