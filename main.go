package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// appModel is the main application model (previously just "model")
type appModel struct {
	goals    []Goal           // Beeminder goals
	cursor   int              // which goal our cursor is pointing at
	selected map[int]struct{} // which goals are selected
	config   *Config          // Beeminder credentials
	loading  bool             // whether we're loading goals
	err      error            // error from loading goals
	width    int              // terminal width
	height   int              // terminal height
}

// model is the top-level model that switches between auth and app
type model struct {
	state     string // "auth" or "app"
	authModel authModel
	appModel  appModel
	width     int    // terminal width
	height    int    // terminal height
}

// goalsLoadedMsg is sent when goals are loaded from the API
type goalsLoadedMsg struct {
	goals []Goal
	err   error
}

// loadGoalsCmd fetches goals from Beeminder API
func loadGoalsCmd(config *Config) tea.Cmd {
	return func() tea.Msg {
		goals, err := FetchGoals(config)
		if err != nil {
			return goalsLoadedMsg{err: err}
		}
		SortGoals(goals)
		return goalsLoadedMsg{goals: goals}
	}
}

func initialAppModel(config *Config) appModel {
	return appModel{
		goals:    []Goal{},
		selected: make(map[int]struct{}),
		config:   config,
		loading:  true,
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
	// In app state, load goals
	return loadGoalsCmd(m.appModel.config)
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

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.appModel.cursor > 0 {
				m.appModel.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.appModel.cursor < len(m.appModel.goals)-1 {
				m.appModel.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := m.appModel.selected[m.appModel.cursor]
			if ok {
				delete(m.appModel.selected, m.appModel.cursor)
			} else {
				m.appModel.selected[m.appModel.cursor] = struct{}{}
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

	if len(m.appModel.goals) == 0 {
		return "No goals found.\n\nPress q to quit.\n"
	}

	// The header
	s := "Beeminder Goals\n\n"

	// Define color styles
	redStyle := lipgloss.NewStyle().Background(lipgloss.Color("1")).Foreground(lipgloss.Color("15")).Padding(0, 1)
	orangeStyle := lipgloss.NewStyle().Background(lipgloss.Color("208")).Foreground(lipgloss.Color("0")).Padding(0, 1)
	blueStyle := lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15")).Padding(0, 1)
	greenStyle := lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0")).Padding(0, 1)
	grayStyle := lipgloss.NewStyle().Background(lipgloss.Color("8")).Foreground(lipgloss.Color("15")).Padding(0, 1)

	// Calculate grid dimensions (4 columns)
	const cols = 4
	rows := (len(m.appModel.goals) + cols - 1) / cols

	// Build grid
	for row := 0; row < rows; row++ {
		var rowCells []string
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(m.appModel.goals) {
				break
			}

			goal := m.appModel.goals[idx]

			// Get color based on buffer
			color := GetBufferColor(goal.Safebuf)
			var style lipgloss.Style
			switch color {
			case "red":
				style = redStyle
			case "orange":
				style = orangeStyle
			case "blue":
				style = blueStyle
			case "green":
				style = greenStyle
			default:
				style = grayStyle
			}

			// Format goal display
			display := fmt.Sprintf("%s\n$%.0f | %s",
				truncateString(goal.Title, 16),
				goal.Pledge,
				FormatDueDate(goal.Losedate))

			cell := style.Render(display)
			rowCells = append(rowCells, cell)
		}
		s += lipgloss.JoinHorizontal(lipgloss.Top, rowCells...) + "\n\n"
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		// Pad with spaces to ensure consistent width
		return s + strings.Repeat(" ", maxLen-len(s))
	}
	return s[:maxLen-3] + "..."
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
