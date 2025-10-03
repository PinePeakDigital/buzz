package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)


// appModel is the main application model (previously just "model")
type appModel struct {
	goals         []Goal           // Beeminder goals
	cursor        int              // which goal our cursor is pointing at
	selected      map[int]struct{} // which goals are selected
	config        *Config          // Beeminder credentials
	loading       bool             // whether we're loading goals
	err           error            // error from loading goals
	width         int              // terminal width
	height        int              // terminal height
	scrollRow     int              // current scroll position (in rows)
	refreshActive bool             // whether auto-refresh is active
	showModal     bool             // whether to show goal details modal
	modalGoal     *Goal            // the goal to show in modal
}

// model is the top-level model that switches between auth and app
type model struct {
	state     string // "auth" or "app"
	authModel authModel
	appModel  appModel
	width     int // terminal width
	height    int // terminal height
}

func initialAppModel(config *Config) appModel {
	return appModel{
		goals:         []Goal{},
		selected:      make(map[int]struct{}),
		config:        config,
		loading:       true,
		refreshActive: true,
	}
}

func initialModel() model {
	// Check if config exists
	if ConfigExists() {
		config, err := LoadConfig()
		if err == nil {
			// Config exists and is valid, go straight to app
			return model{
				state:    "app",
				appModel: initialAppModel(config),
			}
		}
	}

	// No config, start with auth
	return model{
		state:     "auth",
		authModel: initialAuthModel(),
	}
}

func (m model) Init() tea.Cmd {
	if m.state == "auth" {
		return m.authModel.Init()
	}
	// In app state, load goals and start refresh timer
	return tea.Batch(
		loadGoalsCmd(m.appModel.config),
		refreshTickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size messages for both states
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.appModel.width = msg.Width
		m.appModel.height = msg.Height
	}

	if m.state == "auth" {
		// Handle auth state
		switch msg := msg.(type) {
		case authSuccessMsg:
			// Authentication succeeded, switch to app
			m.state = "app"
			m.appModel = initialAppModel(msg.config)
			m.appModel.width = m.width
			m.appModel.height = m.height
			return m, loadGoalsCmd(msg.config)
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
				loadGoalsCmd(m.appModel.config),
				refreshTickCmd(), // Schedule the next refresh
			)
		}
		return m, nil

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// Escape key closes the modal or quits if no modal
		case "esc":
			if m.appModel.showModal {
				m.appModel.showModal = false
				m.appModel.modalGoal = nil
			} else {
				return m, tea.Quit
			}

		// Navigation keys - spatial movement through grid (only when modal is closed)
		case "up", "k":
			if !m.appModel.showModal && len(m.appModel.goals) > 0 {
				cols := calculateColumns(m.appModel.width)
				newCursor := m.appModel.cursor - cols
				if newCursor >= 0 {
					m.appModel.cursor = newCursor
				}
			}

		case "down", "j":
			if !m.appModel.showModal && len(m.appModel.goals) > 0 {
				cols := calculateColumns(m.appModel.width)
				newCursor := m.appModel.cursor + cols
				if newCursor < len(m.appModel.goals) {
					m.appModel.cursor = newCursor
				}
			}

		case "left", "h":
			if !m.appModel.showModal && len(m.appModel.goals) > 0 {
				cols := calculateColumns(m.appModel.width)
				currentCol := m.appModel.cursor % cols
				if currentCol > 0 {
					m.appModel.cursor--
				}
			}

		case "right", "l":
			if !m.appModel.showModal && len(m.appModel.goals) > 0 {
				cols := calculateColumns(m.appModel.width)
				currentCol := m.appModel.cursor % cols
				if currentCol < cols-1 && m.appModel.cursor+1 < len(m.appModel.goals) {
					m.appModel.cursor++
				}
			}

		// The "enter" key shows goal details modal (only when modal is closed)
		case "enter":
			if !m.appModel.showModal && len(m.appModel.goals) > 0 && m.appModel.cursor < len(m.appModel.goals) {
				m.appModel.showModal = true
				m.appModel.modalGoal = &m.appModel.goals[m.appModel.cursor]
			}

		// The spacebar toggles the selected state (only when modal is closed)
		case " ":
			if !m.appModel.showModal {
				_, ok := m.appModel.selected[m.appModel.cursor]
				if ok {
					delete(m.appModel.selected, m.appModel.cursor)
				} else {
					m.appModel.selected[m.appModel.cursor] = struct{}{}
				}
			}

		// Scroll up with Page Up or 'u' (only when modal is closed)
		case "pgup", "u":
			if !m.appModel.showModal && m.appModel.scrollRow > 0 {
				m.appModel.scrollRow--
			}

		// Scroll down with Page Down or 'd' (only when modal is closed)
		case "pgdown", "d":
			if !m.appModel.showModal {
				cols := calculateColumns(m.appModel.width)
				totalRows := (len(m.appModel.goals) + cols - 1) / cols
				maxVisibleRows := max(1, (m.appModel.height-4)/4) // Rough estimate of rows that fit
				if m.appModel.scrollRow < totalRows-maxVisibleRows {
					m.appModel.scrollRow++
				}
			}

		// Manual refresh with 'r' (only when modal is closed)
		case "r":
			if !m.appModel.showModal {
				m.appModel.loading = true
				return m, loadGoalsCmd(m.appModel.config)
			}

		// Toggle auto-refresh with 't' (only when modal is closed)
		case "t":
			if !m.appModel.showModal {
				m.appModel.refreshActive = !m.appModel.refreshActive
				if m.appModel.refreshActive {
					// If we just enabled auto-refresh, start the timer
					return m, refreshTickCmd()
				}
			}
		}
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

	// Render the grid and footer
	grid := RenderGrid(m.appModel.goals, m.appModel.width, m.appModel.height, m.appModel.scrollRow, m.appModel.cursor)
	footer := RenderFooter(m.appModel.goals, m.appModel.width, m.appModel.height, m.appModel.scrollRow, m.appModel.refreshActive)
	
	baseView := grid + footer
	
	// Show modal overlay if modal is active
	if m.appModel.showModal && m.appModel.modalGoal != nil {
		modal := RenderModal(m.appModel.modalGoal, m.appModel.width, m.appModel.height)
		return modal
	}
	
	return baseView
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
