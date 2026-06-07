package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Bubble Tea lifecycle for the top-level `model` defined in model.go.
// Init / Update / View dispatch between the auth flow and the main app view;
// updateApp handles all the in-app messages (goals loaded, datapoint submitted,
// goal details fetched, etc.) plus key presses (delegated to handlers.go).

// navigationTimeout is the duration of inactivity before the cell highlight is auto-disabled
const navigationTimeout = 3 * time.Second

func (m model) Init() tea.Cmd {
	if m.state == "auth" {
		return m.authModel.Init()
	}
	// In app state, load goals and start refresh timer
	return tea.Batch(
		loadGoalsCmd(m.appModel.ctx, m.appModel.client),
		refreshTickCmd(),
		checkRefreshFlagCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size messages for both states
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.appModel.width = msg.Width
		m.appModel.height = msg.Height
		// Re-clamp scroll position to keep cursor visible after resize
		if m.state == "app" {
			displayGoals := m.appModel.getDisplayGoals()
			updateScrollForCursor(&m, len(displayGoals))
		}
	}

	if m.state == "auth" {
		// Handle auth state
		switch msg := msg.(type) {
		case authSuccessMsg:
			// Authentication succeeded, switch to app. Start the refresh
			// ticker and the refresh-flag poller in the same Batch as the
			// initial goal load — without these, auto-refresh wouldn't kick
			// in until the user quit and relaunched.
			m.state = "app"
			m.appModel = initialAppModel(msg.config, m.ctx)
			m.appModel.width = m.width
			m.appModel.height = m.height
			return m, tea.Batch(
				loadGoalsCmd(m.appModel.ctx, m.appModel.client),
				refreshTickCmd(),
				checkRefreshFlagCmd(),
			)
		default:
			var cmd tea.Cmd
			updatedModel, cmd := m.authModel.Update(msg)
			if authModel, ok := updatedModel.(authModel); ok {
				m.authModel = authModel
			} else {
				// Type assertion failed - log error and keep current authModel unchanged
				fmt.Fprintf(os.Stderr, "Warning: authModel.Update returned unexpected type %T, keeping current authModel\n", updatedModel)
				cmd = nil // Return safe command
			}
			return m, cmd
		}
	}

	// Handle app state
	return m.updateApp(msg)
}

func (m model) updateApp(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case goalsLoadedMsg:
		// Goals have been loaded from the API
		m.appModel.loading = false
		if msg.err != nil {
			m.appModel.err = msg.err
		} else {
			m.appModel.goals = msg.goals
			m.appModel.err = nil
		}
		return m, nil

	case refreshTickMsg:
		// Time to refresh data
		if m.appModel.refreshActive {
			return m, tea.Batch(
				loadGoalsCmd(m.appModel.ctx, m.appModel.client),
				refreshTickCmd(), // Schedule the next refresh
			)
		}
		return m, nil

	case datapointSubmittedMsg:
		// Datapoint submission completed
		m.appModel.datapoint.submitting = false
		if msg.err != nil {
			m.appModel.datapoint.err = fmt.Sprintf("Failed to submit: %v", msg.err)
		} else {
			// Success - exit input mode and refresh goals (without showing loading state)
			m.appModel.inputMode = false
			m.appModel.datapoint.focus = 0
			m.appModel.datapoint.err = ""
			// Don't set loading = true here to avoid the full-app loading state
			return m, loadGoalsCmd(m.appModel.ctx, m.appModel.client)
		}
		return m, nil

	case goalDetailsLoadedMsg:
		// Goal details with datapoints have been loaded
		if msg.err != nil {
			// Error loading goal details - continue with basic goal info
			return m, nil
		}
		if m.appModel.showModal && m.appModel.modalGoal != nil && msg.goal != nil {
			// Update the modal goal with the detailed information
			if m.appModel.modalGoal.Slug == msg.goal.Slug {
				m.appModel.modalGoal = msg.goal
			}
		}
		return m, nil

	case goalCreatedMsg:
		// Goal creation completed
		m.appModel.createGoal.creating = false
		if msg.err != nil {
			m.appModel.createGoal.err = fmt.Sprintf("Failed to create goal: %v", msg.err)
		} else {
			// Success - close modal and refresh goals
			m.appModel.showCreateModal = false
			m.appModel.createGoal.err = ""
			return m, loadGoalsCmd(m.appModel.ctx, m.appModel.client)
		}
		return m, nil

	case checkRefreshFlagMsg:
		// Check if another process requested a refresh
		flagTimestamp := getRefreshFlagTimestamp()
		if flagTimestamp > m.lastRefreshTimestamp {
			// New refresh event detected - update our last processed timestamp
			m.lastRefreshTimestamp = flagTimestamp
			return m, tea.Batch(
				loadGoalsCmd(m.appModel.ctx, m.appModel.client),
				checkRefreshFlagCmd(), // Schedule next check
			)
		}
		// No new refresh event, but continue checking
		return m, checkRefreshFlagCmd()

	case navigationTimeoutMsg:
		// Auto-disable highlight after inactivity
		// Only disable if not in modal or search mode
		if !m.appModel.showModal && !m.appModel.searchMode {
			// Check if enough time has elapsed since last navigation
			elapsed := time.Since(m.appModel.lastNavigationTime)
			if elapsed >= navigationTimeout {
				m.appModel.hasNavigated = false
			}
		}
		return m, nil

	// Is it a key press?
	case tea.KeyMsg:
		return handleKeyPress(m, msg)

	// Is it a mouse click?
	case tea.MouseMsg:
		// Only handle clicks when not in a modal
		if !m.appModel.showModal && !m.appModel.showCreateModal {
			if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
				return handleMouseClick(m, msg)
			}
		}
		return m, nil
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	if m.state == "auth" {
		return m.authModel.View()
	}
	return m.viewApp()
}

func (m model) viewApp() string {
	if m.appModel.loading {
		return "Loading goals...\n\nPress q to quit.\n"
	}

	if m.appModel.err != nil {
		return fmt.Sprintf("Error loading goals: %v\n\nPress q to quit.\n", m.appModel.err)
	}

	// Get the goals to display (filtered or all)
	displayGoals := m.appModel.getDisplayGoals()

	// Render the grid and footer
	grid := RenderGrid(displayGoals, m.appModel.width, m.appModel.height, m.appModel.scrollRow, m.appModel.cursor, m.appModel.hasNavigated, m.appModel.config.Username, m.appModel.searchMode, m.appModel.searchQuery)
	footer := RenderFooter(displayGoals, m.appModel.width, m.appModel.height, m.appModel.scrollRow, m.appModel.refreshActive)

	baseView := grid + footer

	// Show create goal modal if active
	if m.appModel.showCreateModal {
		cg := &m.appModel.createGoal
		modal := RenderCreateGoalModal(m.appModel.width, m.appModel.height, cg.slug(), cg.title(),
			cg.goalType(), cg.gunits(), cg.goaldate(), cg.goalval(),
			cg.rate(), cg.focus, cg.err, cg.creating)
		return modal
	}

	// Show modal overlay if modal is active
	if m.appModel.showModal && m.appModel.modalGoal != nil {
		dp := &m.appModel.datapoint
		modal := RenderModal(m.appModel.modalGoal, m.appModel.width, m.appModel.height, dp.date(), dp.value(), dp.comment(), dp.focus, m.appModel.inputMode, dp.err, dp.submitting)
		return modal
	}

	return baseView
}
