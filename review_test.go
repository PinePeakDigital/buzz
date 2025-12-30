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
		gunits   string
		expected string
	}{
		{1.0, "d", "", "1/day"},
		{2.5, "w", "", "2.5/week"},
		{7.0, "d", "", "7/day"},
		{0.5, "w", "", "0.5/week"},
		{10.0, "h", "", "10/hour"},
		{1.0, "m", "", "1/month"},
		{3.0, "y", "", "3/year"},
		{5.0, "d", "pushups", "5 pushups / day"},
		{2.0, "w", "hours", "2 hours / week"},
		{1.0, "d", "pages", "1 pages / day"},
		{3.5, "d", "workouts", "3.5 workouts / day"},
		// Large rates that should not use scientific notation
		{9800.0, "d", "", "9800/day"},
		{12345.0, "w", "", "12345/week"},
		{100000.0, "y", "", "100000/year"},
		{9800.0, "d", "steps", "9800 steps / day"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("rate=%v,runits=%s,gunits=%s", tt.rate, tt.runits, tt.gunits), func(t *testing.T) {
			result := formatRate(tt.rate, tt.runits, tt.gunits)
			if result != tt.expected {
				t.Errorf("formatRate(%v, %s, %s) = %s; want %s", tt.rate, tt.runits, tt.gunits, result, tt.expected)
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

	// Check that the view contains the rate (without gunits)
	expectedRate := "Rate:        2/day"
	if !strings.Contains(view, expectedRate) {
		t.Errorf("Expected view to contain '%s', but got:\n%s", expectedRate, view)
	}
}

func TestReviewModelViewWithRateAndGunits(t *testing.T) {
	rate := 5.0
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
			Gunits:   "pushups",
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view contains the rate with gunits
	expectedRate := "Rate:        5 pushups / day"
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

func TestFormatDueTime(t *testing.T) {
	tests := []struct {
		name           string
		deadlineOffset int
		expected       string
	}{
		{"midnight", 0, "12:00 AM"},
		{"3am", 3 * 3600, "3:00 AM"},
		{"9am", 9 * 3600, "9:00 AM"},
		{"noon", 12 * 3600, "12:00 PM"},
		{"3pm", 15 * 3600, "3:00 PM"},
		{"6pm", 18 * 3600, "6:00 PM"},
		{"9pm", 21 * 3600, "9:00 PM"},
		{"11:30pm", 23*3600 + 30*60, "11:30 PM"},
		{"6am (with minutes)", 6*3600 + 30*60, "6:30 AM"},
		{"before midnight (-1 hour)", -1 * 3600, "11:00 PM"},
		{"before midnight (-30 min)", -30 * 60, "11:30 PM"},
		{"before midnight (-6 hours)", -6 * 3600, "6:00 PM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDueTime(tt.deadlineOffset)
			if result != tt.expected {
				t.Errorf("formatDueTime(%d) = %s; want %s", tt.deadlineOffset, result, tt.expected)
			}
		})
	}
}

func TestReviewModelViewWithDueTime(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "Test Goal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
			Deadline: 15 * 3600, // 3pm
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view contains the due time
	if !strings.Contains(view, "Due time:") {
		t.Error("Expected view to contain 'Due time:' label")
	}

	if !strings.Contains(view, "3:00 PM") {
		t.Errorf("Expected view to contain '3:00 PM', but got:\n%s", view)
	}
}

func TestReviewModelViewWithFineprint(t *testing.T) {
	goals := []Goal{
		{
			Slug:      "testgoal",
			Title:     "Test Goal",
			Fineprint: "This is a test description",
			Safebuf:   5,
			Pledge:    10.0,
			Losedate:  1234567890,
			Limsum:    "+1 in 2 days",
			Baremin:   "+2 in 1 day",
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view contains the fine print
	expectedFineprint := "Fine print:  This is a test description"
	if !strings.Contains(view, expectedFineprint) {
		t.Errorf("Expected view to contain '%s', but got:\n%s", expectedFineprint, view)
	}
}

func TestReviewModelViewWithoutFineprint(t *testing.T) {
	goals := []Goal{
		{
			Slug:      "testgoal",
			Title:     "Test Goal",
			Fineprint: "", // Empty fine print
			Safebuf:   5,
			Pledge:    10.0,
			Losedate:  1234567890,
			Limsum:    "+1 in 2 days",
			Baremin:   "+2 in 1 day",
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view doesn't contain "Fine print:" when fineprint is empty
	if strings.Contains(view, "Fine print:") {
		t.Errorf("Expected view to not contain 'Fine print:' when fineprint is empty, but got:\n%s", view)
	}
}

func TestReviewModelViewWithURL(t *testing.T) {
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

	// Check that the view contains the URL
	if !strings.Contains(view, "URL:") {
		t.Error("Expected view to contain 'URL:' label")
	}

	expectedURL := "https://www.beeminder.com/testuser/testgoal"
	if !strings.Contains(view, expectedURL) {
		t.Errorf("Expected view to contain '%s', but got:\n%s", expectedURL, view)
	}
}

// TestFineprintOrderInOutput verifies that fineprint appears after URL in the output
func TestFineprintOrderInOutput(t *testing.T) {
	goal := Goal{
		Slug:      "testgoal",
		Title:     "Test Goal",
		Fineprint: "This is the fine print",
		Limsum:    "+1 in 2 days",
		Losedate:  1234567890,
		Deadline:  0,
		Pledge:    10.0,
		Autodata:  "manual",
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	output := formatGoalDetails(&goal, config)

	// Find positions of URL and Fine print in the output
	urlIndex := strings.Index(output, "URL:")
	fineprintIndex := strings.Index(output, "Fine print:")

	if urlIndex == -1 {
		t.Error("URL not found in output")
	}

	if fineprintIndex == -1 {
		t.Error("Fine print not found in output")
	}

	// Verify that Fine print comes after URL
	if fineprintIndex <= urlIndex {
		t.Errorf("Expected Fine print to come after URL, but Fine print is at position %d and URL is at position %d", fineprintIndex, urlIndex)
	}
}
