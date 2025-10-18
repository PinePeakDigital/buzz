package main

import "time"

// appModel is the main application model (previously just "model")
type appModel struct {
	goals              []Goal    // Beeminder goals
	cursor             int       // which goal our cursor is pointing at
	config             *Config   // Beeminder credentials
	loading            bool      // whether we're loading goals
	err                error     // error from loading goals
	width              int       // terminal width
	height             int       // terminal height
	scrollRow          int       // current scroll position (in rows)
	refreshActive      bool      // whether auto-refresh is active
	showModal          bool      // whether to show goal details modal
	modalGoal          *Goal     // the goal to show in modal
	hasNavigated       bool      // whether user has used arrow keys
	lastNavigationTime time.Time // last time user navigated with arrow keys

	// Modal input fields
	inputDate    string // date input (YYYY-MM-DD format)
	inputValue   string // value input
	inputComment string // comment input
	inputFocus   int    // which input field is focused (0=date, 1=value, 2=comment)
	inputMode    bool   // whether we're in input mode vs viewing mode
	inputError   string // error message for input validation
	submitting   bool   // whether we're currently submitting a datapoint

	// Filter/search fields
	searchMode  bool   // whether we're in search/filter mode
	searchQuery string // current search query

	// Goal creation fields
	showCreateModal bool   // whether to show goal creation modal
	createSlug      string // goal slug
	createTitle     string // goal title
	createGoalType  string // goal type (hustler, biker, etc.)
	createGunits    string // goal units
	createGoaldate  string // goal date (unix timestamp or "null")
	createGoalval   string // goal value (number or "null")
	createRate      string // rate (number or "null")
	createFocus     int    // which input field is focused
	createError     string // error message for creation validation
	creatingGoal    bool   // whether we're currently creating a goal
}

// model is the top-level model that switches between auth and app
type model struct {
	state                string // "auth" or "app"
	authModel            authModel
	appModel             appModel
	width                int   // terminal width
	height               int   // terminal height
	lastRefreshTimestamp int64 // last processed refresh flag timestamp
}

func initialAppModel(config *Config) appModel {
	return appModel{
		goals:         []Goal{},
		config:        config,
		loading:       true,
		refreshActive: true,
		searchMode:    false,
		searchQuery:   "",
	}
}

// filterGoals returns the goals to display based on search query
func (m *appModel) filterGoals() []Goal {
	if m.searchQuery == "" {
		return m.goals
	}

	var filtered []Goal
	for _, goal := range m.goals {
		// Match against slug or title
		if fuzzyMatch(m.searchQuery, goal.Slug) || fuzzyMatch(m.searchQuery, goal.Title) {
			filtered = append(filtered, goal)
		}
	}
	return filtered
}

// getDisplayGoals returns the goals to display (either filtered or all)
func (m *appModel) getDisplayGoals() []Goal {
	return m.filterGoals()
}

func initialModel() model {
	// Check if config exists
	if ConfigExists() {
		config, err := LoadConfig()
		if err == nil {
			// Config exists and is valid, go straight to app
			return model{
				state:                "app",
				appModel:             initialAppModel(config),
				lastRefreshTimestamp: time.Now().Unix(), // Initialize to current timestamp
			}
		}
	}

	// No config, start with auth
	return model{
		state:                "auth",
		authModel:            initialAuthModel(),
		lastRefreshTimestamp: time.Now().Unix(), // Initialize to current timestamp
	}
}
