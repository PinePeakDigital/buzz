package main

import (
	"context"
	"time"
)

// mode is the single foreground screen the app is showing. Exactly one mode is
// active at a time; it is mutated only through the transition methods below so
// that each mode's companion state (e.g. modalGoal) stays consistent by
// construction. See docs/adr/0002-mode-enum-with-guard-railed-transitions.md.
type mode uint8

const (
	modeBrowse         mode = iota // the scrollable grid of goals (default)
	modeGoalDetail                 // a single goal's detail popup, over the grid
	modeDatapointInput             // datapoint entry form, reachable only from modeGoalDetail
	modeCreateGoal                 // new-goal form, reachable only from modeBrowse
)

// appModel is the main application model (previously just "model")
type appModel struct {
	goals              []Goal          // Beeminder goals
	cursor             int             // which goal our cursor is pointing at
	config             *Config         // Beeminder credentials (kept for openBrowser URL building)
	client             Client          // Beeminder API client
	ctx                context.Context // long-lived context derived from main()'s cancellable parent; cancelled when p.Run() returns so in-flight Client calls abort on quit
	loading            bool            // whether we're loading goals
	err                error           // error from loading goals
	width              int             // terminal width
	height             int             // terminal height
	scrollRow          int             // current scroll position (in rows)
	refreshActive      bool            // whether auto-refresh is active
	mode               mode            // current foreground screen (see transition methods)
	modalGoal          *Goal           // the goal shown in the detail modal; non-nil iff mode is modeGoalDetail/modeDatapointInput
	hasNavigated       bool            // whether user has used arrow keys
	lastNavigationTime time.Time       // last time user navigated with arrow keys

	// Datapoint entry form (shown inside the goal detail modal)
	datapoint datapointForm // date/value/comment fields + submitting flag

	// Search is a filter layer orthogonal to mode: it filters the Browse grid
	// and persists underneath whatever mode is foreground.
	searchActive bool   // whether the search/filter layer is active
	searchQuery  string // current search query

	// Goal creation form
	createGoal createGoalForm // slug/title/type/... fields + creating flag
}

// inGoalModal reports whether a goal-detail modal is on screen (whether or not
// the nested datapoint-input form is focused).
func (m *appModel) inGoalModal() bool {
	return m.mode == modeGoalDetail || m.mode == modeDatapointInput
}

// --- Mode transitions ---------------------------------------------------------
// These are the only places mode and modalGoal are mutated, so invalid
// combinations (e.g. a detail modal with no goal attached) are unrepresentable.

// openGoalDetail opens (or re-targets) the goal-detail modal for goal g. Calling
// it while already in modeGoalDetail just switches which goal is shown. A nil
// goal is ignored so the invariant "modalGoal is non-nil whenever a goal modal
// is open" holds by construction.
func (m *appModel) openGoalDetail(g *Goal) {
	if g == nil {
		return
	}
	m.mode = modeGoalDetail
	m.modalGoal = g
}

// startDatapointInput focuses the datapoint-entry form nested in the goal-detail
// modal. It is a no-op unless a goal detail with an attached goal is open (the
// submit path dereferences modalGoal.Slug).
func (m *appModel) startDatapointInput(form datapointForm) {
	if m.mode != modeGoalDetail || m.modalGoal == nil {
		return
	}
	m.mode = modeDatapointInput
	m.datapoint = form
}

// exitDatapointInput cancels datapoint entry and returns to the goal detail.
func (m *appModel) exitDatapointInput() {
	if m.mode != modeDatapointInput {
		return
	}
	m.mode = modeGoalDetail
	m.datapoint.focus = 0
	m.datapoint.err = ""
}

// closeModal closes the goal-detail modal and returns to Browse, leaving any
// active search in place.
func (m *appModel) closeModal() {
	m.mode = modeBrowse
	m.modalGoal = nil
}

// openCreateGoal opens the new-goal form with fresh fields.
func (m *appModel) openCreateGoal() {
	m.mode = modeCreateGoal
	m.createGoal = newCreateGoalForm()
}

// closeCreateGoal closes the new-goal form and returns to Browse.
func (m *appModel) closeCreateGoal() {
	m.mode = modeBrowse
	m.createGoal.err = ""
}

// enterSearch activates the search filter layer with an empty query.
func (m *appModel) enterSearch() {
	m.searchActive = true
	m.searchQuery = ""
}

// exitSearch clears the search filter layer and resets grid navigation.
func (m *appModel) exitSearch() {
	m.searchActive = false
	m.searchQuery = ""
	m.cursor = 0
	m.scrollRow = 0
	m.hasNavigated = false
}

// model is the top-level model that switches between auth and app. It holds
// the cancellable parent context so the appModel reconstructed on
// authSuccessMsg can inherit the same cancellation source as one created
// directly via initialModel.
type model struct {
	state                string // "auth" or "app"
	authModel            authModel
	appModel             appModel
	ctx                  context.Context // cancellable parent threaded into appModel(s); cancelled when main()'s p.Run() returns
	width                int             // terminal width
	height               int             // terminal height
	lastRefreshTimestamp int64           // last processed refresh flag timestamp
}

func initialAppModel(config *Config, ctx context.Context) appModel {
	return appModel{
		goals:         []Goal{},
		config:        config,
		client:        NewHTTPClient(config),
		ctx:           ctx,
		loading:       true,
		refreshActive: true,
		// mode defaults to modeBrowse and searchActive to false (zero values).
	}
}

// filterGoals returns the goals to display based on search query. The query is
// only non-empty while the search layer is active (kept in sync by enterSearch/
// exitSearch), so an empty query is the single "show everything" condition.
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

func initialModel(ctx context.Context) model {
	// Check if config exists
	if ConfigExists() {
		config, err := LoadConfig()
		if err == nil {
			// Config exists and is valid, go straight to app
			return model{
				state:                "app",
				appModel:             initialAppModel(config, ctx),
				ctx:                  ctx,
				lastRefreshTimestamp: time.Now().Unix(), // Initialize to current timestamp
			}
		}
	}

	// No config, start with auth
	return model{
		state:                "auth",
		authModel:            initialAuthModel(),
		ctx:                  ctx,
		lastRefreshTimestamp: time.Now().Unix(), // Initialize to current timestamp
	}
}
