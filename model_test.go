package main

import (
	"context"
	"testing"
)

// TestFilterGoals tests the filterGoals method
func TestFilterGoals(t *testing.T) {
	goals := []Goal{
		{Slug: "exercise", Title: "Daily Exercise"},
		{Slug: "reading", Title: "Read Books"},
		{Slug: "meditation", Title: "Daily Meditation"},
		{Slug: "writing", Title: "Write Blog Posts"},
	}

	tests := []struct {
		name     string
		query    string
		expected []string // slugs of expected goals
	}{
		{
			name:     "empty query returns all",
			query:    "",
			expected: []string{"exercise", "reading", "meditation", "writing"},
		},
		{
			name:     "exact slug match",
			query:    "exercise",
			expected: []string{"exercise"},
		},
		{
			name:     "partial slug match",
			query:    "read",
			expected: []string{"reading"},
		},
		{
			name:     "title match",
			query:    "daily",
			expected: []string{"exercise", "meditation"},
		},
		{
			name:     "fuzzy match",
			query:    "ex",
			expected: []string{"exercise"},
		},
		{
			name:     "no match",
			query:    "xyz",
			expected: []string{},
		},
		{
			name:     "case insensitive",
			query:    "EXERCISE",
			expected: []string{"exercise"},
		},
		{
			name:     "fuzzy match across multiple goals",
			query:    "d",
			expected: []string{"exercise", "reading", "meditation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &appModel{
				goals:       goals,
				searchQuery: tt.query,
			}

			result := m.filterGoals()

			if len(result) != len(tt.expected) {
				t.Errorf("filterGoals() returned %d goals, want %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", getSlugs(result))
				t.Errorf("Want: %v", tt.expected)
				return
			}

			resultSlugs := getSlugs(result)
			for i, expectedSlug := range tt.expected {
				if resultSlugs[i] != expectedSlug {
					t.Errorf("Goal %d: got slug %q, want %q", i, resultSlugs[i], expectedSlug)
				}
			}
		})
	}
}

// TestGetDisplayGoals tests the getDisplayGoals method
func TestGetDisplayGoals(t *testing.T) {
	allGoals := []Goal{
		{Slug: "goal1", Title: "Goal 1"},
		{Slug: "goal2", Title: "Goal 2"},
		{Slug: "goal3", Title: "Goal 3"},
	}

	tests := []struct {
		name        string
		searchQuery string
		expected    int // expected number of goals
	}{
		{
			name:        "no search query returns all goals",
			searchQuery: "",
			expected:    3,
		},
		{
			name:        "with search query returns filtered goals",
			searchQuery: "goal1",
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &appModel{
				goals:       allGoals,
				searchQuery: tt.searchQuery,
			}

			result := m.getDisplayGoals()

			if len(result) != tt.expected {
				t.Errorf("getDisplayGoals() returned %d goals, want %d", len(result), tt.expected)
			}
		})
	}
}

// TestInitialModel tests the initialModel function
func TestInitialModel(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		// Since we can't easily mock ConfigExists in the test,
		// we just verify the function creates a valid model structure
		m := initialModel(context.Background())

		if m.state != "auth" && m.state != "app" {
			t.Errorf("initialModel() state = %q, want 'auth' or 'app'", m.state)
		}
	})
}

// TestInitialAppModel tests the initialAppModel function
func TestInitialAppModel(t *testing.T) {
	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	m := initialAppModel(config, context.Background())

	// Verify initial state
	if m.config != config {
		t.Error("initialAppModel() did not set config correctly")
	}

	if !m.loading {
		t.Error("initialAppModel() should start in loading state")
	}

	if !m.refreshActive {
		t.Error("initialAppModel() should start with refreshActive = true")
	}

	if m.searchActive {
		t.Error("initialAppModel() should start with searchActive = false")
	}

	if m.mode != modeBrowse {
		t.Errorf("initialAppModel() mode = %d, want modeBrowse", m.mode)
	}

	if m.searchQuery != "" {
		t.Errorf("initialAppModel() searchQuery = %q, want empty", m.searchQuery)
	}

	if len(m.goals) != 0 {
		t.Errorf("initialAppModel() should start with empty goals slice, got %d goals", len(m.goals))
	}
}

// TestModelContextPropagation verifies that the cancellable parent context
// passed to initialModel reaches the appModel (both directly and through the
// auth → app transition via authSuccessMsg). When the parent cancels,
// m.appModel.ctx.Done() must fire — that's what makes quit-cancellation work
// for in-flight Client calls.
func TestModelContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := initialModel(ctx)
	if m.ctx != ctx {
		t.Error("initialModel did not store the passed ctx on the model")
	}
	// When ConfigExists() returns true the appModel is built immediately;
	// otherwise it's built later from authSuccessMsg. Either way, the
	// appModel ctx should match the model ctx — exercise the direct path
	// here, and TestAuthSuccessPropagatesCtx covers the auth flow.
	if m.state == "app" && m.appModel.ctx != ctx {
		t.Error("appModel.ctx should equal the parent ctx when built directly")
	}
}

func TestInitialAppModelStoresCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := initialAppModel(&Config{Username: "u", AuthToken: "t"}, ctx)
	if app.ctx != ctx {
		t.Error("initialAppModel did not store the passed ctx")
	}
	// Cancellation on the parent must be observable through the app's ctx.
	cancel()
	select {
	case <-app.ctx.Done():
		// expected
	default:
		t.Error("cancelling parent ctx did not cancel appModel.ctx")
	}
}

// TestModeTransitions exercises the guard-railed mode transitions directly,
// asserting that each one keeps mode and its companion state consistent so that
// invalid combinations (e.g. a goal-detail modal with no goal attached) cannot
// arise. See docs/adr/0002-mode-enum-with-guard-railed-transitions.md.
func TestModeTransitions(t *testing.T) {
	t.Run("openGoalDetail attaches the goal", func(t *testing.T) {
		m := appModel{}
		g := &Goal{Slug: "exercise"}
		m.openGoalDetail(g)
		if m.mode != modeGoalDetail {
			t.Errorf("mode = %d, want modeGoalDetail", m.mode)
		}
		if m.modalGoal != g {
			t.Error("openGoalDetail should attach the goal")
		}
		if !m.inGoalModal() {
			t.Error("inGoalModal() should be true in modeGoalDetail")
		}
	})

	t.Run("openGoalDetail re-targets while already open", func(t *testing.T) {
		m := appModel{}
		m.openGoalDetail(&Goal{Slug: "first"})
		m.openGoalDetail(&Goal{Slug: "second"})
		if m.modalGoal == nil || m.modalGoal.Slug != "second" {
			t.Error("openGoalDetail should switch to the new goal")
		}
	})

	t.Run("startDatapointInput only works from goal detail", func(t *testing.T) {
		// From Browse it is a no-op.
		m := appModel{}
		m.startDatapointInput(newDatapointForm("1"))
		if m.mode != modeBrowse {
			t.Errorf("startDatapointInput from Browse should be a no-op, mode = %d", m.mode)
		}

		// From goal detail it enters input mode.
		m.openGoalDetail(&Goal{Slug: "exercise"})
		m.startDatapointInput(newDatapointForm("2.5"))
		if m.mode != modeDatapointInput {
			t.Errorf("mode = %d, want modeDatapointInput", m.mode)
		}
		if m.datapoint.value() != "2.5" {
			t.Errorf("datapoint value = %q, want %q", m.datapoint.value(), "2.5")
		}
		if !m.inGoalModal() {
			t.Error("inGoalModal() should be true in modeDatapointInput")
		}
	})

	t.Run("exitDatapointInput returns to goal detail", func(t *testing.T) {
		m := appModel{}
		m.openGoalDetail(&Goal{Slug: "exercise"})
		m.startDatapointInput(newDatapointForm("1"))
		m.exitDatapointInput()
		if m.mode != modeGoalDetail {
			t.Errorf("mode = %d, want modeGoalDetail after exitDatapointInput", m.mode)
		}
		if m.modalGoal == nil {
			t.Error("goal should remain attached after exiting datapoint input")
		}
	})

	t.Run("closeModal returns to Browse and clears the goal but keeps search", func(t *testing.T) {
		m := appModel{}
		m.enterSearch()
		m.searchQuery = "weight"
		m.openGoalDetail(&Goal{Slug: "weight"})
		m.closeModal()
		if m.mode != modeBrowse {
			t.Errorf("mode = %d, want modeBrowse", m.mode)
		}
		if m.modalGoal != nil {
			t.Error("closeModal should clear the attached goal")
		}
		if !m.searchActive || m.searchQuery != "weight" {
			t.Error("closeModal should leave the search layer intact")
		}
	})

	t.Run("openCreateGoal and closeCreateGoal", func(t *testing.T) {
		m := appModel{}
		m.openCreateGoal()
		if m.mode != modeCreateGoal {
			t.Errorf("mode = %d, want modeCreateGoal", m.mode)
		}
		m.createGoal.err = "boom"
		m.closeCreateGoal()
		if m.mode != modeBrowse {
			t.Errorf("mode = %d, want modeBrowse after closeCreateGoal", m.mode)
		}
		if m.createGoal.err != "" {
			t.Error("closeCreateGoal should clear the form error")
		}
	})

	t.Run("enterSearch and exitSearch", func(t *testing.T) {
		m := appModel{cursor: 5, scrollRow: 3, hasNavigated: true}
		m.enterSearch()
		if !m.searchActive || m.searchQuery != "" {
			t.Error("enterSearch should activate search with an empty query")
		}
		m.searchQuery = "abc"
		m.exitSearch()
		if m.searchActive || m.searchQuery != "" {
			t.Error("exitSearch should clear the search layer")
		}
		if m.cursor != 0 || m.scrollRow != 0 || m.hasNavigated {
			t.Error("exitSearch should reset grid navigation state")
		}
	})
}

// Helper function to extract slugs from goals
func getSlugs(goals []Goal) []string {
	slugs := make([]string, len(goals))
	for i, goal := range goals {
		slugs[i] = goal.Slug
	}
	return slugs
}
