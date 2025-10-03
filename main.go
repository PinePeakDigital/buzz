package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// appModel is the main application model (previously just "model")
type appModel struct {
	choices  []string           // items on the to-do list
	cursor   int                // which to-do list item our cursor is pointing at
	selected map[int]struct{}   // which to-do items are selected
	config   *Config            // Beeminder credentials
}

// model is the top-level model that switches between auth and app
type model struct {
	state    string      // "auth" or "app"
	authModel authModel
	appModel  appModel
}

func initialAppModel(config *Config) appModel {
	return appModel{
		// A to-do list can have any number of items
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

		// A map which indicates which choices are selected. We're using
		// the map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
		config:   config,
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
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == "auth" {
		// Handle auth state
		switch msg := msg.(type) {
		case authSuccessMsg:
			// Authentication succeeded, switch to app
			m.state = "app"
			m.appModel = initialAppModel(msg.config)
			return m, nil
		default:
			var cmd tea.Cmd
			updatedModel, cmd := m.authModel.Update(msg)
			m.authModel = updatedModel.(authModel)
			return m, cmd
		}
	}

	// Handle app state
	return m.updateApp(msg)
}

func (m model) updateApp(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

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
			if m.appModel.cursor < len(m.appModel.choices)-1 {
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
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range m.appModel.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.appModel.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.appModel.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}