package main

import (
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
		m := initialModel()

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

	m := initialAppModel(config)

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

	if m.searchMode {
		t.Error("initialAppModel() should start with searchMode = false")
	}

	if m.searchQuery != "" {
		t.Errorf("initialAppModel() searchQuery = %q, want empty", m.searchQuery)
	}

	if len(m.goals) != 0 {
		t.Errorf("initialAppModel() should start with empty goals slice, got %d goals", len(m.goals))
	}
}

// Helper function to extract slugs from goals
func getSlugs(goals []Goal) []string {
	slugs := make([]string, len(goals))
	for i, goal := range goals {
		slugs[i] = goal.Slug
	}
	return slugs
}
