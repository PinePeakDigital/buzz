package main

import (
	"fmt"
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

func TestFormatRate(t *testing.T) {
	tests := []struct {
		rate     float64
		runits   string
		expected string
	}{
		{1.0, "d", "1/day"},
		{2.5, "w", "2.5/week"},
		{7.0, "d", "7/day"},
		{0.5, "w", "0.5/week"},
		{10.0, "h", "10/hour"},
		{1.0, "m", "1/month"},
		{3.0, "y", "3/year"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("rate=%v,runits=%s", tt.rate, tt.runits), func(t *testing.T) {
			result := formatRate(tt.rate, tt.runits)
			if result != tt.expected {
				t.Errorf("formatRate(%v, %s) = %s; want %s", tt.rate, tt.runits, result, tt.expected)
			}
		})
	}
}

func TestReviewModelViewWithRate(t *testing.T) {
	rate := 2.0
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "Test Goal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
			Rate:     &rate,
			Runits:   "d",
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view contains the rate
	expectedRate := "Rate:        2/day"
	if !strings.Contains(view, expectedRate) {
		t.Errorf("Expected view to contain '%s', but got:\n%s", expectedRate, view)
	}
}

func TestReviewModelViewWithoutRate(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "Test Goal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
			Rate:     nil, // No rate
			Runits:   "",
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view doesn't contain "Rate:" when rate is nil
	if strings.Contains(view, "Rate:") {
		t.Errorf("Expected view to not contain 'Rate:' when rate is nil, but got:\n%s", view)
	}
}

func TestReviewModelViewWithAutoratchet(t *testing.T) {
	autoratchet := 7.0
	goals := []Goal{
		{
			Slug:        "testgoal",
			Title:       "Test Goal",
			Safebuf:     5,
			Pledge:      10.0,
			Losedate:    1234567890,
			Limsum:      "+1 in 2 days",
			Baremin:     "+2 in 1 day",
			Autoratchet: &autoratchet,
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view contains the autoratchet value
	expectedAutoratchet := "Autoratchet: 7"
	if !strings.Contains(view, expectedAutoratchet) {
		t.Errorf("Expected view to contain '%s' when autoratchet is set, but got:\n%s", expectedAutoratchet, view)
	}
}

func TestReviewModelViewWithoutAutoratchet(t *testing.T) {
	goals := []Goal{
		{
			Slug:        "testgoal",
			Title:       "Test Goal",
			Safebuf:     5,
			Pledge:      10.0,
			Losedate:    1234567890,
			Limsum:      "+1 in 2 days",
			Baremin:     "+2 in 1 day",
			Autoratchet: nil, // No autoratchet (disabled)
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view doesn't contain "Autoratchet:" when autoratchet is nil
	if strings.Contains(view, "Autoratchet:") {
		t.Errorf("Expected view to not contain 'Autoratchet:' when autoratchet is nil, but got:\n%s", view)
	}
}
