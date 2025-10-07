package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// appModel is the main application model (previously just "model")
type appModel struct {
	goals         []Goal  // Beeminder goals
	cursor        int     // which goal our cursor is pointing at
	config        *Config // Beeminder credentials
	loading       bool    // whether we're loading goals
	err           error   // error from loading goals
	width         int     // terminal width
	height        int     // terminal height
	scrollRow     int     // current scroll position (in rows)
	refreshActive bool    // whether auto-refresh is active
	showModal     bool    // whether to show goal details modal
	modalGoal     *Goal   // the goal to show in modal
	hasNavigated  bool    // whether user has used arrow keys

	// Modal input fields
	inputDate    string // date input (YYYY-MM-DD format)
	inputValue   string // value input
	inputComment string // comment input
	inputFocus   int    // which input field is focused (0=date, 1=value, 2=comment)
	inputMode    bool   // whether we're in input mode vs viewing mode
	inputError   string // error message for input validation
	submitting   bool   // whether we're currently submitting a datapoint

	// Filter/search fields
	searchMode    bool   // whether we're in search/filter mode
	searchQuery   string // current search query
	filteredGoals []Goal // goals filtered by search query

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
	state     string // "auth" or "app"
	authModel authModel
	appModel  appModel
	width     int // terminal width
	height    int // terminal height
}

func initialAppModel(config *Config) appModel {
	return appModel{
		goals:         []Goal{},
		config:        config,
		loading:       true,
		refreshActive: true,
		searchMode:    false,
		searchQuery:   "",
		filteredGoals: []Goal{},
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
	if m.searchQuery != "" {
		return m.filteredGoals
	}
	return m.goals
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

	case datapointSubmittedMsg:
		// Datapoint submission completed
		m.appModel.submitting = false
		if msg.err != nil {
			m.appModel.inputError = fmt.Sprintf("Failed to submit: %v", msg.err)
		} else {
			// Success - exit input mode and refresh goals (without showing loading state)
			m.appModel.inputMode = false
			m.appModel.inputFocus = 0
			m.appModel.inputError = ""
			// Don't set loading = true here to avoid the full-app loading state
			return m, loadGoalsCmd(m.appModel.config)
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
		m.appModel.creatingGoal = false
		if msg.err != nil {
			m.appModel.createError = fmt.Sprintf("Failed to create goal: %v", msg.err)
		} else {
			// Success - close modal and refresh goals
			m.appModel.showCreateModal = false
			m.appModel.createError = ""
			return m, loadGoalsCmd(m.appModel.config)
		}
		return m, nil

	// Is it a key press?
	case tea.KeyMsg:
		// Handle text input in search mode FIRST
		if m.appModel.searchMode && !m.appModel.showModal {
			char := msg.String()
			// Allow printable characters in search
			if len(char) == 1 && char >= " " && char <= "~" {
				m.appModel.searchQuery += char
				m.appModel.filteredGoals = m.appModel.filterGoals()
				// Reset cursor and scroll when search query changes
				m.appModel.cursor = 0
				m.appModel.scrollRow = 0
				m.appModel.hasNavigated = false
				return m, nil
			}
		}

		// Handle text input in create goal modal
		if m.appModel.showCreateModal && !m.appModel.creatingGoal {
			char := msg.String()
			if len(char) == 1 && char >= " " && char <= "~" {
				switch m.appModel.createFocus {
				case 0: // Slug - allow alphanumeric and dashes/underscores
					if (char >= "a" && char <= "z") || (char >= "A" && char <= "Z") ||
						(char >= "0" && char <= "9") || char == "-" || char == "_" {
						m.appModel.createSlug += char
						return m, nil
					}
				case 1: // Title - allow all printable characters
					m.appModel.createTitle += char
					return m, nil
				case 2: // Goal type - allow letters
					if (char >= "a" && char <= "z") || (char >= "A" && char <= "Z") {
						m.appModel.createGoalType += char
						return m, nil
					}
				case 3: // Gunits - allow all printable characters
					m.appModel.createGunits += char
					return m, nil
				case 4: // Goaldate - allow digits
					if (char >= "0" && char <= "9") || char == "n" || char == "u" || char == "l" {
						m.appModel.createGoaldate += char
						return m, nil
					}
				case 5: // Goalval - allow digits, decimal point, and negative sign
					if (char >= "0" && char <= "9") || char == "." || char == "-" || char == "n" || char == "u" || char == "l" {
						m.appModel.createGoalval += char
						return m, nil
					}
				case 6: // Rate - allow digits, decimal point, and negative sign
					if (char >= "0" && char <= "9") || char == "." || char == "-" || char == "n" || char == "u" || char == "l" {
						m.appModel.createRate += char
						return m, nil
					}
				}
			}
		}

		// Handle text input in input mode SECOND, before command keys
		// This ensures that single-character command keys (like 't', 'r', 'd', etc.)
		// can still be typed in comment fields
		if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
			char := msg.String()
			if len(char) == 1 {
				switch m.appModel.inputFocus {
				case 0: // Date field - allow digits and dashes
					if (char >= "0" && char <= "9") || char == "-" {
						m.appModel.inputDate += char
						return m, nil
					}
				case 1: // Value field - allow digits, decimal point, and negative sign
					if (char >= "0" && char <= "9") || char == "." || char == "-" {
						m.appModel.inputValue += char
						return m, nil
					}
				case 2: // Comment field - allow all printable characters
					if char >= " " && char <= "~" {
						m.appModel.inputComment += char
						return m, nil
					}
				}
			}
		}

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// Escape key closes search mode, modal, or quits
		case "esc":
			if m.appModel.searchMode {
				// Exit search mode
				m.appModel.searchMode = false
				m.appModel.searchQuery = ""
				m.appModel.filteredGoals = []Goal{}
				m.appModel.cursor = 0
				m.appModel.scrollRow = 0
				m.appModel.hasNavigated = false
			} else if m.appModel.showCreateModal {
				// Close create goal modal
				m.appModel.showCreateModal = false
				m.appModel.createError = ""
			} else if m.appModel.showModal {
				if m.appModel.inputMode {
					// Exit input mode
					m.appModel.inputMode = false
					m.appModel.inputFocus = 0
					m.appModel.inputError = ""
				} else {
					// Close modal
					m.appModel.showModal = false
					m.appModel.modalGoal = nil
				}
			} else {
				return m, tea.Quit
			}

		// Enter input mode with 'a' (only when modal is open but not in input mode and not submitting)
		case "a":
			if m.appModel.showModal && !m.appModel.inputMode && !m.appModel.submitting {
				m.appModel.inputMode = true
				m.appModel.inputFocus = 0
				m.appModel.inputError = "" // Clear any previous errors
				// Set default values
				m.appModel.inputDate = time.Now().Format("2006-01-02")
				m.appModel.inputComment = "Added via buzz"

				// Try to get the last datapoint value, default to "1" if it fails
				if lastValue, err := GetLastDatapointValue(m.appModel.config, m.appModel.modalGoal.Slug); err == nil && lastValue != 0 {
					m.appModel.inputValue = fmt.Sprintf("%.1f", lastValue)
				} else {
					m.appModel.inputValue = "1"
				}
			}

		// Tab navigation between input fields (only in input mode and not submitting)
		case "tab":
			if m.appModel.showCreateModal && !m.appModel.creatingGoal {
				m.appModel.createFocus = (m.appModel.createFocus + 1) % 7
			} else if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
				m.appModel.inputFocus = (m.appModel.inputFocus + 1) % 3
			} // Shift+Tab navigation in input mode (reverse)
		case "shift+tab":
			if m.appModel.showCreateModal && !m.appModel.creatingGoal {
				m.appModel.createFocus = (m.appModel.createFocus + 6) % 7 // +6 is same as -1 in mod 7
			} else if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
				m.appModel.inputFocus = (m.appModel.inputFocus + 2) % 3 // +2 is same as -1 in mod 3
			}

		// Backspace handling in search mode or input mode
		case "backspace":
			if m.appModel.showCreateModal && !m.appModel.creatingGoal {
				switch m.appModel.createFocus {
				case 0: // Slug
					if len(m.appModel.createSlug) > 0 {
						m.appModel.createSlug = m.appModel.createSlug[:len(m.appModel.createSlug)-1]
					}
				case 1: // Title
					if len(m.appModel.createTitle) > 0 {
						m.appModel.createTitle = m.appModel.createTitle[:len(m.appModel.createTitle)-1]
					}
				case 2: // Goal type
					if len(m.appModel.createGoalType) > 0 {
						m.appModel.createGoalType = m.appModel.createGoalType[:len(m.appModel.createGoalType)-1]
					}
				case 3: // Gunits
					if len(m.appModel.createGunits) > 0 {
						m.appModel.createGunits = m.appModel.createGunits[:len(m.appModel.createGunits)-1]
					}
				case 4: // Goaldate
					if len(m.appModel.createGoaldate) > 0 {
						m.appModel.createGoaldate = m.appModel.createGoaldate[:len(m.appModel.createGoaldate)-1]
					}
				case 5: // Goalval
					if len(m.appModel.createGoalval) > 0 {
						m.appModel.createGoalval = m.appModel.createGoalval[:len(m.appModel.createGoalval)-1]
					}
				case 6: // Rate
					if len(m.appModel.createRate) > 0 {
						m.appModel.createRate = m.appModel.createRate[:len(m.appModel.createRate)-1]
					}
				}
			} else if m.appModel.searchMode && !m.appModel.showModal {
				// Remove last character from search query
				if len(m.appModel.searchQuery) > 0 {
					m.appModel.searchQuery = m.appModel.searchQuery[:len(m.appModel.searchQuery)-1]
					m.appModel.filteredGoals = m.appModel.filterGoals()
					// Reset cursor and scroll when search query changes
					m.appModel.cursor = 0
					m.appModel.scrollRow = 0
					m.appModel.hasNavigated = false
				}
			} else if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
				switch m.appModel.inputFocus {
				case 0: // Date field
					if len(m.appModel.inputDate) > 0 {
						m.appModel.inputDate = m.appModel.inputDate[:len(m.appModel.inputDate)-1]
					}
				case 1: // Value field
					if len(m.appModel.inputValue) > 0 {
						m.appModel.inputValue = m.appModel.inputValue[:len(m.appModel.inputValue)-1]
					}
				case 2: // Comment field
					if len(m.appModel.inputComment) > 0 {
						m.appModel.inputComment = m.appModel.inputComment[:len(m.appModel.inputComment)-1]
					}
				}
			}

		// Submit form with Enter in input mode or create modal
		case "enter":
			if m.appModel.showCreateModal && !m.appModel.creatingGoal {
				// Clear previous error
				m.appModel.createError = ""

				// Validate input fields
				if m.appModel.createSlug == "" {
					m.appModel.createError = "Slug cannot be empty"
					return m, nil
				}

				if m.appModel.createTitle == "" {
					m.appModel.createError = "Title cannot be empty"
					return m, nil
				}

				if m.appModel.createGoalType == "" {
					m.appModel.createError = "Goal type cannot be empty"
					return m, nil
				}

				if m.appModel.createGunits == "" {
					m.appModel.createError = "Goal units cannot be empty"
					return m, nil
				}

				// Validate that exactly 2 out of 3 (goaldate, goalval, rate) are provided
				countProvided := 0
				if m.appModel.createGoaldate != "" && m.appModel.createGoaldate != "null" {
					countProvided++
				}
				if m.appModel.createGoalval != "" && m.appModel.createGoalval != "null" {
					countProvided++
				}
				if m.appModel.createRate != "" && m.appModel.createRate != "null" {
					countProvided++
				}

				if countProvided != 2 {
					m.appModel.createError = "Exactly 2 out of 3 (goaldate, goalval, rate) must be provided"
					return m, nil
				}

				// Set creating state and submit goal creation asynchronously
				m.appModel.creatingGoal = true
				return m, createGoalCmd(m.appModel.config, m.appModel.createSlug, m.appModel.createTitle,
					m.appModel.createGoalType, m.appModel.createGunits, m.appModel.createGoaldate,
					m.appModel.createGoalval, m.appModel.createRate)
			} else if m.appModel.showModal && m.appModel.inputMode && !m.appModel.submitting {
				// Clear previous error
				m.appModel.inputError = ""

				// Validate input fields
				if m.appModel.inputDate == "" {
					m.appModel.inputError = "Date cannot be empty"
					return m, nil
				}

				if m.appModel.inputValue == "" {
					m.appModel.inputError = "Value cannot be empty"
					return m, nil
				}

				// Parse and validate date
				date, err := time.Parse("2006-01-02", m.appModel.inputDate)
				if err != nil {
					m.appModel.inputError = "Invalid date format (use YYYY-MM-DD)"
					return m, nil
				}

				// Validate that date is not in the future beyond today
				if date.After(time.Now().AddDate(0, 0, 1)) {
					m.appModel.inputError = "Date cannot be more than 1 day in the future"
					return m, nil
				}

				// Parse and validate value (must be a valid number)
				if _, err := strconv.ParseFloat(m.appModel.inputValue, 64); err != nil {
					m.appModel.inputError = "Value must be a valid number"
					return m, nil
				}

				timestamp := fmt.Sprintf("%d", date.Unix())

				// Set submitting state and submit datapoint asynchronously
				m.appModel.submitting = true
				return m, submitDatapointCmd(m.appModel.config, m.appModel.modalGoal.Slug,
					timestamp, m.appModel.inputValue, m.appModel.inputComment)
			} else if !m.appModel.showModal {
				// Show goal details modal (existing functionality)
				displayGoals := m.appModel.getDisplayGoals()
				if len(displayGoals) > 0 && m.appModel.cursor < len(displayGoals) {
					m.appModel.showModal = true
					m.appModel.modalGoal = &displayGoals[m.appModel.cursor]

					// Update cursor to point to the goal in the original goals list
					// This is necessary for left/right navigation in modal
					for i, goal := range m.appModel.goals {
						if goal.Slug == displayGoals[m.appModel.cursor].Slug {
							m.appModel.cursor = i
							break
						}
					}

					// Load detailed goal information including datapoints
					return m, loadGoalDetailsCmd(m.appModel.config, m.appModel.modalGoal.Slug)
				}
			}

		// Navigation keys - spatial movement through grid (only when modal is closed)
		case "up", "k":
			if !m.appModel.showModal {
				displayGoals := m.appModel.getDisplayGoals()
				if len(displayGoals) > 0 {
					m.appModel.hasNavigated = true
					cols := calculateColumns(m.appModel.width)
					newCursor := m.appModel.cursor - cols
					if newCursor >= 0 {
						m.appModel.cursor = newCursor
					}
				}
			}

		case "down", "j":
			if !m.appModel.showModal {
				displayGoals := m.appModel.getDisplayGoals()
				if len(displayGoals) > 0 {
					m.appModel.hasNavigated = true
					cols := calculateColumns(m.appModel.width)
					newCursor := m.appModel.cursor + cols
					if newCursor < len(displayGoals) {
						m.appModel.cursor = newCursor
					}
				}
			}

		case "left", "h":
			if m.appModel.showModal && !m.appModel.inputMode && !m.appModel.submitting && len(m.appModel.goals) > 0 {
				// Navigate to previous goal in modal view
				if m.appModel.cursor > 0 {
					m.appModel.cursor--
					m.appModel.modalGoal = &m.appModel.goals[m.appModel.cursor]
					// Load detailed goal information including datapoints
					return m, loadGoalDetailsCmd(m.appModel.config, m.appModel.modalGoal.Slug)
				}
			} else if !m.appModel.showModal {
				displayGoals := m.appModel.getDisplayGoals()
				if len(displayGoals) > 0 {
					m.appModel.hasNavigated = true
					cols := calculateColumns(m.appModel.width)
					currentCol := m.appModel.cursor % cols
					if currentCol > 0 {
						m.appModel.cursor--
					}
				}
			}

		case "right", "l":
			if m.appModel.showModal && !m.appModel.inputMode && !m.appModel.submitting && len(m.appModel.goals) > 0 {
				// Navigate to next goal in modal view
				if m.appModel.cursor < len(m.appModel.goals)-1 {
					m.appModel.cursor++
					m.appModel.modalGoal = &m.appModel.goals[m.appModel.cursor]
					// Load detailed goal information including datapoints
					return m, loadGoalDetailsCmd(m.appModel.config, m.appModel.modalGoal.Slug)
				}
			} else if !m.appModel.showModal {
				displayGoals := m.appModel.getDisplayGoals()
				if len(displayGoals) > 0 {
					m.appModel.hasNavigated = true
					cols := calculateColumns(m.appModel.width)
					currentCol := m.appModel.cursor % cols
					if currentCol < cols-1 && m.appModel.cursor+1 < len(displayGoals) {
						m.appModel.cursor++
					}
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
				displayGoals := m.appModel.getDisplayGoals()
				cols := calculateColumns(m.appModel.width)
				totalRows := (len(displayGoals) + cols - 1) / cols
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

		// Enter search mode with '/' (only when modal is closed and not already in search mode)
		case "/":
			if !m.appModel.showModal && !m.appModel.searchMode {
				m.appModel.searchMode = true
				m.appModel.searchQuery = ""
				m.appModel.filteredGoals = []Goal{}
			}

		// Open create goal modal with 'n' for new (only when no modal is open)
		case "n":
			if !m.appModel.showModal && !m.appModel.showCreateModal && !m.appModel.searchMode {
				m.appModel.showCreateModal = true
				m.appModel.createFocus = 0
				m.appModel.createError = ""
				// Set default values
				m.appModel.createSlug = ""
				m.appModel.createTitle = ""
				m.appModel.createGoalType = "hustler"
				m.appModel.createGunits = "units"
				m.appModel.createGoaldate = ""
				m.appModel.createGoalval = "0"
				m.appModel.createRate = "1"
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

	// Get the goals to display (filtered or all)
	displayGoals := m.appModel.getDisplayGoals()

	// Render the grid and footer
	grid := RenderGrid(displayGoals, m.appModel.width, m.appModel.height, m.appModel.scrollRow, m.appModel.cursor, m.appModel.hasNavigated, m.appModel.config.Username, m.appModel.searchMode, m.appModel.searchQuery)
	footer := RenderFooter(displayGoals, m.appModel.width, m.appModel.height, m.appModel.scrollRow, m.appModel.refreshActive)

	baseView := grid + footer

	// Show create goal modal if active
	if m.appModel.showCreateModal {
		modal := RenderCreateGoalModal(m.appModel.width, m.appModel.height, m.appModel.createSlug, m.appModel.createTitle,
			m.appModel.createGoalType, m.appModel.createGunits, m.appModel.createGoaldate, m.appModel.createGoalval,
			m.appModel.createRate, m.appModel.createFocus, m.appModel.createError, m.appModel.creatingGoal)
		return modal
	}

	// Show modal overlay if modal is active
	if m.appModel.showModal && m.appModel.modalGoal != nil {
		modal := RenderModal(m.appModel.modalGoal, m.appModel.width, m.appModel.height, m.appModel.inputDate, m.appModel.inputValue, m.appModel.inputComment, m.appModel.inputFocus, m.appModel.inputMode, m.appModel.inputError, m.appModel.submitting)
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
