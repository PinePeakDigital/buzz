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

func TestFormatGoalDetailsWithDatapoints(t *testing.T) {
	// Test that formatGoalDetails includes datapoints when present
	datapoints := []Datapoint{
		{
			ID:        "1",
			Timestamp: 1609459200, // 2021-01-01
			Daystamp:  "20210101",
			Value:     5.0,
			Comment:   "First datapoint",
		},
		{
			ID:        "2",
			Timestamp: 1609545600, // 2021-01-02
			Daystamp:  "20210102",
			Value:     7.5,
			Comment:   "Second datapoint",
		},
	}

	goal := &Goal{
		Slug:       "testgoal",
		Title:      "Test Goal",
		Limsum:     "+2 in 3 days",
		Losedate:   1234567890,
		Deadline:   0,
		Pledge:     5.0,
		Autodata:   "none",
		Datapoints: datapoints,
	}

	config := &Config{
		Username: "testuser",
	}

	result := formatGoalDetails(goal, config)

	// Check that the result contains datapoint information
	if !strings.Contains(result, "Recent datapoints:") {
		t.Error("Expected result to contain 'Recent datapoints:' header")
	}

	if !strings.Contains(result, "2021-01-01") {
		t.Error("Expected result to contain first datapoint date '2021-01-01'")
	}

	if !strings.Contains(result, "5") {
		t.Error("Expected result to contain first datapoint value '5'")
	}

	if !strings.Contains(result, "First datapoint") {
		t.Error("Expected result to contain first datapoint comment 'First datapoint'")
	}

	if !strings.Contains(result, "2021-01-02") {
		t.Error("Expected result to contain second datapoint date '2021-01-02'")
	}

	if !strings.Contains(result, "7.5") {
		t.Error("Expected result to contain second datapoint value '7.5'")
	}

	if !strings.Contains(result, "Second datapoint") {
		t.Error("Expected result to contain second datapoint comment 'Second datapoint'")
	}
}

func TestFormatGoalDetailsWithoutDatapoints(t *testing.T) {
	// Test that formatGoalDetails works correctly when no datapoints are present
	goal := &Goal{
		Slug:       "testgoal",
		Title:      "Test Goal",
		Limsum:     "+2 in 3 days",
		Losedate:   1234567890,
		Deadline:   0,
		Pledge:     5.0,
		Autodata:   "none",
		Datapoints: []Datapoint{}, // Empty datapoints
	}

	config := &Config{
		Username: "testuser",
	}

	result := formatGoalDetails(goal, config)

	// Check that the result does NOT contain datapoint information
	if strings.Contains(result, "Recent datapoints:") {
		t.Error("Expected result to NOT contain 'Recent datapoints:' header when no datapoints present")
	}

	// Check that it still contains basic goal information
	if !strings.Contains(result, "Test Goal") {
		t.Error("Expected result to contain goal title 'Test Goal'")
	}

	if !strings.Contains(result, "+2 in 3 days") {
		t.Error("Expected result to contain limsum '+2 in 3 days'")
	}
}

func TestFormatRecentDatapoints(t *testing.T) {
	tests := []struct {
		name       string
		datapoints []Datapoint
		wantEmpty  bool
		wantCount  int
		checkFor   []string
	}{
		{
			name:       "empty datapoints",
			datapoints: []Datapoint{},
			wantEmpty:  true,
		},
		{
			name: "single datapoint with comment",
			datapoints: []Datapoint{
				{
					ID:        "1",
					Timestamp: 1609459200, // 2021-01-01
					Daystamp:  "20210101",
					Value:     5.0,
					Comment:   "Test comment",
				},
			},
			wantEmpty: false,
			wantCount: 1,
			checkFor:  []string{"Recent datapoints:", "2021-01-01", "5", "Test comment"},
		},
		{
			name: "single datapoint without comment",
			datapoints: []Datapoint{
				{
					ID:        "1",
					Timestamp: 1609459200, // 2021-01-01
					Daystamp:  "20210101",
					Value:     10.5,
					Comment:   "",
				},
			},
			wantEmpty: false,
			wantCount: 1,
			checkFor:  []string{"Recent datapoints:", "2021-01-01", "10.5"},
		},
		{
			name: "multiple datapoints",
			datapoints: []Datapoint{
				{
					ID:        "1",
					Timestamp: 1609459200, // 2021-01-01
					Daystamp:  "20210101",
					Value:     5.0,
					Comment:   "First",
				},
				{
					ID:        "2",
					Timestamp: 1609545600, // 2021-01-02
					Daystamp:  "20210102",
					Value:     7.5,
					Comment:   "Second",
				},
				{
					ID:        "3",
					Timestamp: 1609632000, // 2021-01-03
					Daystamp:  "20210103",
					Value:     3.0,
					Comment:   "Third",
				},
			},
			wantEmpty: false,
			wantCount: 3,
			checkFor:  []string{"Recent datapoints:", "2021-01-01", "2021-01-02", "2021-01-03", "5", "7.5", "3", "First", "Second", "Third"},
		},
		{
			name: "more than 5 datapoints shows only 5",
			datapoints: []Datapoint{
				{ID: "1", Timestamp: 1609459200, Daystamp: "20210101", Value: 1.0, Comment: "One"},
				{ID: "2", Timestamp: 1609545600, Daystamp: "20210102", Value: 2.0, Comment: "Two"},
				{ID: "3", Timestamp: 1609632000, Daystamp: "20210103", Value: 3.0, Comment: "Three"},
				{ID: "4", Timestamp: 1609718400, Daystamp: "20210104", Value: 4.0, Comment: "Four"},
				{ID: "5", Timestamp: 1609804800, Daystamp: "20210105", Value: 5.0, Comment: "Five"},
				{ID: "6", Timestamp: 1609891200, Daystamp: "20210106", Value: 6.0, Comment: "Six"},
				{ID: "7", Timestamp: 1609977600, Daystamp: "20210107", Value: 7.0, Comment: "Seven"},
			},
			wantEmpty: false,
			wantCount: 5,
			checkFor:  []string{"Recent datapoints:", "2021-01-01", "2021-01-05", "One", "Five"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRecentDatapoints(tt.datapoints)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("Expected empty string for empty datapoints, got: %s", result)
				}
				return
			}

			// Check all required strings are present
			for _, check := range tt.checkFor {
				if !strings.Contains(result, check) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", check, result)
				}
			}

			// Count lines (excluding header and empty lines)
			lines := strings.Split(result, "\n")
			datapointLines := 0
			for _, line := range lines {
				// Count lines that start with "  " (datapoint lines)
				if strings.HasPrefix(line, "  ") && strings.TrimSpace(line) != "" {
					datapointLines++
				}
			}
			if datapointLines != tt.wantCount {
				t.Errorf("Expected %d datapoint lines, got %d.\nOutput:\n%s", tt.wantCount, datapointLines, result)
			}
		})
	}
}
