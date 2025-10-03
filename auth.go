package main

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type authModel struct {
	textInput textinput.Model
	err       error
	success   bool
}

// authSuccessMsg is sent when authentication succeeds
type authSuccessMsg struct {
	config *Config
}

func initialAuthModel() authModel {
	ti := textinput.New()
	ti.Placeholder = `{"username":"your_username","auth_token":"your_token"}`
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80

	return authModel{
		textInput: ti,
		err:       nil,
		success:   false,
	}
}

func (m authModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m authModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			// Try to parse and save the credentials
			input := m.textInput.Value()
			if input == "" {
				m.err = fmt.Errorf("please enter your credentials")
				m.textInput.SetValue("")
				return m, nil
			}

			var config Config
			if err := json.Unmarshal([]byte(input), &config); err != nil {
				m.err = fmt.Errorf("invalid JSON format: %v", err)
				m.textInput.SetValue("")
				return m, nil
			}

			// Validate that required fields are present
			if config.Username == "" || config.AuthToken == "" {
				m.err = fmt.Errorf("username and auth_token are required")
				m.textInput.SetValue("")
				return m, nil
			}

			// Save the config
			if err := SaveConfig(&config); err != nil {
				m.err = fmt.Errorf("failed to save config: %v", err)
				m.textInput.SetValue("")
				return m, nil
			}

			// Signal success
			m.success = true
			return m, func() tea.Msg {
				return authSuccessMsg{config: &config}
			}
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m authModel) View() string {
	s := "Beeminder Authentication\n\n"
	s += "Please paste your Beeminder API credentials in JSON format.\n"
	s += "Get them from: https://www.beeminder.com/api/v1/auth_token.json\n\n"
	s += "Format: {\"username\":\"your_username\",\"auth_token\":\"your_token\"}\n\n"
	s += m.textInput.View() + "\n\n"

	if m.err != nil {
		s += fmt.Sprintf("Error: %v\n\n", m.err)
	}

	if m.success {
		s += "✓ Authentication successful! Starting application...\n\n"
	}

	s += "Press Enter to submit • Esc or Ctrl+C to quit\n"

	return s
}
