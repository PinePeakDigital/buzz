package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

	// With goals present, Init dispatches the lazy fetch of the first goal's
	// details, so it returns a command (not nil).
	m := initialReviewModel(goals, config)
	if cmd := m.Init(); cmd == nil {
		t.Error("Expected Init() to return a details-fetch command when goals exist")
	}
	if !m.loading {
		t.Error("Expected loading=true on init when goals exist")
	}

	// With no goals, there's nothing to fetch.
	empty := initialReviewModel(nil, config)
	if cmd := empty.Init(); cmd != nil {
		t.Error("Expected Init() to return nil when there are no goals")
	}
	if empty.loading {
		t.Error("Expected loading=false on init when there are no goals")
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
		// Full-precision API rates are rounded to a readable number of decimals
		// rather than dumping the raw float (issue #260).
		{0.21317778888888886, "d", "hours", "0.2132 hours / day"},
		{0.1900775022222092, "d", "", "0.1901/day"},
		{0.0, "d", "", "0/day"},
		// A small negative rate that rounds to zero must render as "0", not
		// "-0", for do-less / downward-sloping goals (issue #260).
		{-0.00001, "d", "", "0/day"},
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

func TestReviewModelViewWithCurrentAndEndRate(t *testing.T) {
	// When the current rate differs from the end rate, both are shown so the
	// user sees today's rate and where the goal is heading (issue #259).
	endRate := 0.21317778888888886
	curRate := 0.0
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "Test Goal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
			Rate:     &endRate,
			Currate:  &curRate,
			Runits:   "d",
			Gunits:   "hours",
		},
	}

	config := &Config{Username: "testuser", AuthToken: "testtoken"}
	view := initialReviewModel(goals, config).View()

	expectedRate := "Rate:        0 hours / day (current), 0.2132 (end)"
	if !strings.Contains(view, expectedRate) {
		t.Errorf("Expected view to contain '%s', but got:\n%s", expectedRate, view)
	}
}

func TestReviewModelViewWithEqualCurrentAndEndRate(t *testing.T) {
	// On a flat road the current and end rates match, so only a single rate is
	// shown rather than redundantly repeating it (issue #259).
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
			Currate:  &rate,
			Runits:   "d",
		},
	}

	config := &Config{Username: "testuser", AuthToken: "testtoken"}
	view := initialReviewModel(goals, config).View()

	if !strings.Contains(view, "Rate:        2/day") {
		t.Errorf("Expected single rate 'Rate:        2/day', but got:\n%s", view)
	}
	if strings.Contains(view, "(current)") {
		t.Errorf("Expected no current/end split when rates are equal, but got:\n%s", view)
	}
}

func TestReviewModelViewCurrentRateFromLegacyRcur(t *testing.T) {
	// Some API payloads carry the current rate as `rcur` rather than `currate`;
	// CurrentRate() falls back to it so the current/end split still renders.
	endRate := 1.0
	curRate := 0.5
	goals := []Goal{
		{
			Slug:     "testgoal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
			Rate:     &endRate,
			Rcur:     &curRate,
			Runits:   "d",
		},
	}

	config := &Config{Username: "testuser", AuthToken: "testtoken"}
	view := initialReviewModel(goals, config).View()

	if !strings.Contains(view, "Rate:        0.5/day (current), 1 (end)") {
		t.Errorf("Expected current/end split from rcur fallback, but got:\n%s", view)
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
		// Regression: -3599s (59:59 before midnight = 23:00:01) used to drift
		// to 11:01 PM under the hand-rolled negative branch.
		{"before midnight (-3599s)", -3599, "11:00 PM"},
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

// TestReviewModelViewWithTitle verifies that title is shown when present
func TestReviewModelViewWithTitle(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "My Test Goal",
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

	// Check that the view contains the title when it's not empty
	expectedTitle := "Title:       My Test Goal"
	if !strings.Contains(view, expectedTitle) {
		t.Errorf("Expected view to contain '%s' when title is set, but got:\n%s", expectedTitle, view)
	}
}

// TestReviewModelViewWithEmptyTitle verifies that empty title is not shown
func TestReviewModelViewWithEmptyTitle(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "", // Empty title
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

	// Check that the view doesn't contain "Title:" when title is empty
	if strings.Contains(view, "Title:") {
		t.Errorf("Expected view to not contain 'Title:' when title is empty, but got:\n%s", view)
	}
}

// TestReviewModelViewWithEmptyAutodata verifies that empty autodata is not shown
func TestReviewModelViewWithEmptyAutodata(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "Test Goal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
			Autodata: "", // Empty autodata
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view doesn't contain "Autodata:" when autodata is empty
	if strings.Contains(view, "Autodata:") {
		t.Errorf("Expected view to not contain 'Autodata:' when autodata is empty, but got:\n%s", view)
	}
}

// TestReviewModelViewWithAutodata verifies that autodata is shown when present
func TestReviewModelViewWithAutodata(t *testing.T) {
	goals := []Goal{
		{
			Slug:     "testgoal",
			Title:    "Test Goal",
			Safebuf:  5,
			Pledge:   10.0,
			Losedate: 1234567890,
			Limsum:   "+1 in 2 days",
			Baremin:  "+2 in 1 day",
			Autodata: "manual",
		},
	}

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialReviewModel(goals, config)
	view := m.View()

	// Check that the view contains the autodata when it's not empty
	expectedAutodata := "Autodata:    manual"
	if !strings.Contains(view, expectedAutodata) {
		t.Errorf("Expected view to contain '%s' when autodata is set, but got:\n%s", expectedAutodata, view)
	}
}

func TestFormatGoalDetailsWithDatapoints(t *testing.T) {
	// Test that formatGoalDetails includes datapoints when present
	datapoints := []Datapoint{
		{ID: "1", Timestamp: 1609459200, Daystamp: "20210101", Value: 5.0, Comment: "First datapoint"},
		{ID: "2", Timestamp: 1609545600, Daystamp: "20210102", Value: 7.5, Comment: "Second datapoint"},
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

	config := &Config{Username: "testuser"}

	result := formatGoalDetails(goal, config)

	for _, want := range []string{
		"Recent datapoints:",
		"2021-01-01", "5", "First datapoint",
		"2021-01-02", "7.5", "Second datapoint",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("Expected result to contain %q, but got:\n%s", want, result)
		}
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
		Datapoints: []Datapoint{},
	}

	config := &Config{Username: "testuser"}

	result := formatGoalDetails(goal, config)

	if strings.Contains(result, "Recent datapoints:") {
		t.Error("Expected result to NOT contain 'Recent datapoints:' header when no datapoints present")
	}
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
		absent     []string
	}{
		{
			name:       "empty datapoints",
			datapoints: []Datapoint{},
			wantEmpty:  true,
		},
		{
			name: "single datapoint with comment",
			datapoints: []Datapoint{
				{ID: "1", Timestamp: 1609459200, Daystamp: "20210101", Value: 5.0, Comment: "Test comment"},
			},
			wantCount: 1,
			checkFor:  []string{"Recent datapoints:", "2021-01-01", "5", "Test comment"},
		},
		{
			name: "single datapoint without comment",
			datapoints: []Datapoint{
				{ID: "1", Timestamp: 1609459200, Daystamp: "20210101", Value: 10.5, Comment: ""},
			},
			wantCount: 1,
			checkFor:  []string{"Recent datapoints:", "2021-01-01", "10.5"},
		},
		{
			name: "multiple datapoints",
			datapoints: []Datapoint{
				{ID: "1", Timestamp: 1609459200, Daystamp: "20210101", Value: 5.0, Comment: "First"},
				{ID: "2", Timestamp: 1609545600, Daystamp: "20210102", Value: 7.5, Comment: "Second"},
				{ID: "3", Timestamp: 1609632000, Daystamp: "20210103", Value: 3.0, Comment: "Third"},
			},
			wantCount: 3,
			checkFor:  []string{"Recent datapoints:", "2021-01-01", "2021-01-02", "2021-01-03", "5", "7.5", "3", "First", "Second", "Third"},
		},
		{
			// API returns datapoints oldest-first, so the most recent 5 of 7
			// are 2021-01-03..07. The two oldest (01-01/"One", 01-02/"Two")
			// must be dropped.
			name: "more than 5 datapoints shows only the 5 most recent",
			datapoints: []Datapoint{
				{ID: "1", Timestamp: 1609459200, Daystamp: "20210101", Value: 1.0, Comment: "One"},
				{ID: "2", Timestamp: 1609545600, Daystamp: "20210102", Value: 2.0, Comment: "Two"},
				{ID: "3", Timestamp: 1609632000, Daystamp: "20210103", Value: 3.0, Comment: "Three"},
				{ID: "4", Timestamp: 1609718400, Daystamp: "20210104", Value: 4.0, Comment: "Four"},
				{ID: "5", Timestamp: 1609804800, Daystamp: "20210105", Value: 5.0, Comment: "Five"},
				{ID: "6", Timestamp: 1609891200, Daystamp: "20210106", Value: 6.0, Comment: "Six"},
				{ID: "7", Timestamp: 1609977600, Daystamp: "20210107", Value: 7.0, Comment: "Seven"},
			},
			wantCount: 5,
			checkFor:  []string{"Recent datapoints:", "2021-01-03", "2021-01-07", "Three", "Seven"},
			absent:    []string{"2021-01-01", "2021-01-02", "One", "Two"},
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

			for _, check := range tt.checkFor {
				if !strings.Contains(result, check) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", check, result)
				}
			}
			for _, gone := range tt.absent {
				if strings.Contains(result, gone) {
					t.Errorf("Expected output to NOT contain %q, but it did.\nOutput:\n%s", gone, result)
				}
			}

			// Count datapoint lines (indented, non-empty)
			datapointLines := 0
			for _, line := range strings.Split(result, "\n") {
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

func TestFormatRecentDatapointsOrdersMostRecentFirst(t *testing.T) {
	// Datapoints arrive oldest-first; the output must list newest first.
	datapoints := []Datapoint{
		{ID: "1", Daystamp: "20210101", Value: 1.0, Comment: "Oldest"},
		{ID: "2", Daystamp: "20210102", Value: 2.0, Comment: "Middle"},
		{ID: "3", Daystamp: "20210103", Value: 3.0, Comment: "Newest"},
	}

	result := formatRecentDatapoints(datapoints)

	newestIdx := strings.Index(result, "2021-01-03")
	oldestIdx := strings.Index(result, "2021-01-01")
	if newestIdx == -1 || oldestIdx == -1 {
		t.Fatalf("Expected both dates in output, got:\n%s", result)
	}
	if newestIdx > oldestIdx {
		t.Errorf("Expected most recent datapoint (2021-01-03) before oldest (2021-01-01), got:\n%s", result)
	}
}

func TestFormatRecentDatapointsFallsBackToTimestamp(t *testing.T) {
	// When daystamp is missing/malformed, fall back to the UTC timestamp date.
	datapoints := []Datapoint{
		{ID: "1", Timestamp: 1609459200, Value: 5.0, Comment: "No daystamp"}, // 2021-01-01 UTC
	}

	result := formatRecentDatapoints(datapoints)

	if !strings.Contains(result, "2021-01-01") {
		t.Errorf("Expected timestamp fallback date '2021-01-01', got:\n%s", result)
	}
}

// TestGoalDetailsFieldOrder locks in the field ordering from issue #229:
// Rate, Autoratchet, Limsum, Deadline, Due time, Pledge, Title, URL — with the
// fields the issue didn't enumerate following after.
func TestGoalDetailsFieldOrder(t *testing.T) {
	rate := 0.5
	autoratchet := 7.0
	goal := &Goal{
		Slug:        "clean",
		Title:       "#autodialMax=0.5",
		Limsum:      "+0.07 in 6 days",
		Pledge:      5.0,
		Rate:        &rate,
		Runits:      "d",
		Gunits:      "hours",
		Autoratchet: &autoratchet,
		Autodata:    "ifttt",
		Fineprint:   "be honest",
		Losedate:    4102444800, // fixed future timestamp; only the label's position matters here
	}
	config := &Config{Username: "narthur"}

	out := formatGoalDetails(goal, config)

	// Each label must appear, and in this exact relative order.
	want := []string{
		"Rate:", "Autoratchet:", "Limsum:", "Deadline:", "Due time:",
		"Pledge:", "Title:", "URL:", "Autodata:", "Fine print:",
	}
	prev := -1
	for _, label := range want {
		idx := strings.Index(out, label)
		if idx == -1 {
			t.Fatalf("output missing %q\n%s", label, out)
		}
		if idx < prev {
			t.Errorf("%q appears out of order (want sequence %v)\n%s", label, want, out)
		}
		prev = idx
	}
}

// TestGoalDetailsFieldOrderMinimal verifies the #229 order still holds — and
// the conditional fields are omitted — when Rate, Autoratchet, Title, Autodata,
// and Fine print are all unset.
func TestGoalDetailsFieldOrderMinimal(t *testing.T) {
	goal := &Goal{
		Slug:     "spark",
		Limsum:   "+1 in 3 days",
		Pledge:   5.0,
		Losedate: 4102444800, // fixed future timestamp; only label positions matter
		// Rate nil, Autoratchet nil, Title/Autodata/Fineprint empty → all omitted.
	}
	out := formatGoalDetails(goal, &Config{Username: "narthur"})

	for _, absent := range []string{"Rate:", "Autoratchet:", "Title:", "Autodata:", "Fine print:"} {
		if strings.Contains(out, absent) {
			t.Errorf("expected %q to be omitted when unset\n%s", absent, out)
		}
	}
	// The always-present fields keep their #229 relative order.
	prev := -1
	for _, label := range []string{"Limsum:", "Deadline:", "Due time:", "Pledge:", "URL:"} {
		idx := strings.Index(out, label)
		if idx == -1 {
			t.Fatalf("output missing %q\n%s", label, out)
		}
		if idx < prev {
			t.Errorf("%q appears out of order\n%s", label, out)
		}
		prev = idx
	}
}

func TestReviewGoalDetailsMsgCachesAndClearsLoading(t *testing.T) {
	m := initialReviewModel([]Goal{{Slug: "g1"}, {Slug: "g2"}}, &Config{Username: "u"})
	m.err = "stale error"

	fetched := &Goal{Slug: "g1", Title: "Hydrated"}
	updated, _ := m.Update(goalDetailsMsg{slug: "g1", goal: fetched})
	m = updated.(reviewModel)

	if m.loading {
		t.Error("expected loading=false after the current goal's details arrive")
	}
	if got, ok := m.details["g1"]; !ok || got.Title != "Hydrated" {
		t.Errorf("expected details[g1] cached as the fetched goal, got %+v (present=%v)", got, ok)
	}
	if m.err != "" {
		t.Errorf("expected err cleared on success, got %q", m.err)
	}
}

func TestReviewGoalDetailsMsgError(t *testing.T) {
	m := initialReviewModel([]Goal{{Slug: "g1"}}, &Config{Username: "u"})

	updated, _ := m.Update(goalDetailsMsg{slug: "g1", err: fmt.Errorf("boom")})
	m = updated.(reviewModel)

	if m.loading {
		t.Error("expected loading=false even on fetch error")
	}
	if !strings.Contains(m.err, "Failed to load goal details") {
		t.Errorf("expected err to mention the failure, got %q", m.err)
	}
}

func TestReviewGoalDetailsMsgForNonCurrentGoalDoesNotClearLoading(t *testing.T) {
	// A late result for a goal the user already navigated away from should be
	// cached but must not clear the loading state of the current goal.
	m := initialReviewModel([]Goal{{Slug: "g1"}, {Slug: "g2"}}, &Config{Username: "u"})
	m.current = 1 // viewing g2, still loading it
	m.loading = true

	updated, _ := m.Update(goalDetailsMsg{slug: "g1", goal: &Goal{Slug: "g1"}})
	m = updated.(reviewModel)

	if _, ok := m.details["g1"]; !ok {
		t.Error("expected g1 details to be cached even though it's not current")
	}
	if !m.loading {
		t.Error("expected loading to stay true: g2 (current) is still loading")
	}
}

func TestReviewNavigationTriggersFetchOnlyWhenUncached(t *testing.T) {
	m := initialReviewModel([]Goal{{Slug: "g1"}, {Slug: "g2"}}, &Config{Username: "u"})
	m.details["g1"] = &Goal{Slug: "g1"} // g1 cached, g2 not

	// Navigate to g2 (uncached) → loading + a fetch command.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(reviewModel)
	if m.current != 1 {
		t.Fatalf("expected to move to g2 (index 1), got %d", m.current)
	}
	if !m.loading || cmd == nil {
		t.Errorf("expected loading=true and a fetch command for uncached g2 (loading=%v, cmd=%v)", m.loading, cmd != nil)
	}

	// Navigate back to g1 (cached) → no loading, no command.
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(reviewModel)
	if m.current != 0 {
		t.Fatalf("expected to move back to g1 (index 0), got %d", m.current)
	}
	if m.loading || cmd != nil {
		t.Errorf("expected no loading/command for cached g1 (loading=%v, cmd=%v)", m.loading, cmd != nil)
	}
}

func TestReviewNavigateAwayAndBackDoesNotRefetch(t *testing.T) {
	// goals[0] (g1) is marked in-flight by the constructor (Init dispatches it).
	m := initialReviewModel([]Goal{{Slug: "g1"}, {Slug: "g2"}}, &Config{Username: "u"})
	if _, ok := m.inFlight["g1"]; !ok {
		t.Fatal("expected g1 marked in-flight after construction")
	}

	// Navigate to g2 (uncached, not in flight) → dispatch a fetch for it.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(reviewModel)
	if cmd == nil {
		t.Fatal("expected a fetch command for uncached g2")
	}
	if _, ok := m.inFlight["g2"]; !ok {
		t.Error("expected g2 marked in-flight after navigating to it")
	}

	// Navigate back to g1 before its first fetch resolves. It's still in flight,
	// so no second request should be dispatched — just keep the spinner.
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(reviewModel)
	if cmd != nil {
		t.Error("expected NO second fetch for g1 while its first fetch is in flight")
	}
	if !m.loading {
		t.Error("expected loading=true while g1 is still being fetched")
	}

	// When g1's single fetch finally resolves, it clears in-flight and loading.
	updated, _ = m.Update(goalDetailsMsg{slug: "g1", goal: &Goal{Slug: "g1"}})
	m = updated.(reviewModel)
	if _, ok := m.inFlight["g1"]; ok {
		t.Error("expected g1 cleared from in-flight after its fetch resolved")
	}
	if m.loading {
		t.Error("expected loading=false after g1 (current) resolved")
	}
}

func TestReviewViewMergesDetailFieldsOntoSummaryGoal(t *testing.T) {
	// Bulk summary goal: has summary fields but no datapoints/road. The cached
	// detail goal deliberately has an EMPTY Title to prove View keeps the
	// summary's Title (merge, not replace) while pulling in datapoints + road.
	rate := 0.5
	goals := []Goal{{
		Slug:     "g",
		Title:    "Bulk Title",
		Limsum:   "+1 in 2 days",
		Pledge:   5.0,
		Rate:     &rate,
		Runits:   "d",
		Losedate: 4102444800,
	}}

	now := time.Now()
	detail := &Goal{
		Slug:  "g",
		Yaw:   1,
		Title: "", // empty on purpose: must NOT overwrite the summary title
		Datapoints: []Datapoint{
			{Timestamp: now.AddDate(0, 0, -2).Unix(), Value: 1},
			{Timestamp: now.AddDate(0, 0, -1).Unix(), Value: 2},
		},
		Roadall: [][]*float64{
			roadallRow(float64(now.AddDate(0, 0, -30).Unix()), fptr(0.0), nil),
			roadallRow(float64(now.Unix()), fptr(5.0), nil),
		},
	}

	m := initialReviewModel(goals, &Config{Username: "u"})
	m.details["g"] = detail
	m.loading = false
	m.width = 100

	out := m.View()

	// Summary field survives the merge (came from the bulk goal, not the detail).
	if !strings.Contains(out, "Bulk Title") {
		t.Errorf("expected summary Title preserved after merge\n%s", out)
	}
	// Detail-only data is merged in and rendered.
	if !strings.Contains(out, "Recent datapoints") {
		t.Errorf("expected merged datapoints to render\n%s", out)
	}
	if !strings.Contains(out, "Goal Progress Chart") {
		t.Errorf("expected chart (from merged road + datapoints) to render\n%s", out)
	}
}
