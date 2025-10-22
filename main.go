package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set via ldflags during build
var version = "dev"

// navigationTimeout is the duration of inactivity before the cell highlight is auto-disabled
const navigationTimeout = 3 * time.Second

func (m model) Init() tea.Cmd {
	if m.state == "auth" {
		return m.authModel.Init()
	}
	// In app state, load goals and start refresh timer
	return tea.Batch(
		loadGoalsCmd(m.appModel.config),
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

	case checkRefreshFlagMsg:
		// Check if another process requested a refresh
		flagTimestamp := getRefreshFlagTimestamp()
		if flagTimestamp > m.lastRefreshTimestamp {
			// New refresh event detected - update our last processed timestamp
			m.lastRefreshTimestamp = flagTimestamp
			return m, tea.Batch(
				loadGoalsCmd(m.appModel.config),
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

func printHelp() {
	fmt.Println("buzz - A terminal user interface for Beeminder")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  buzz                              Launch the interactive TUI")
	fmt.Println("  buzz next                         Output a terse summary of the next due goal")
	fmt.Println("  buzz next --watch                 Watch mode - continuously refresh every 5 minutes")
	fmt.Println("  buzz next -w                      Watch mode (shorthand)")
	fmt.Println("  buzz today                        Output all goals due today")
	fmt.Println("  buzz add <goalslug> <value> [comment]")
	fmt.Println("                                    Add a datapoint to a goal")
	fmt.Println("  buzz refresh <goalslug>           Refresh autodata for a goal")
	fmt.Println("  buzz view <goalslug>              View detailed information about a specific goal")
	fmt.Println("  buzz review                       Interactive review of all goals")
	fmt.Println("  buzz help                         Show this help message")
	fmt.Println("")
	fmt.Println("OPTIONS:")
	fmt.Println("  -h, --help                        Show this help message")
	fmt.Println("  -v, --version                     Show version information")
	fmt.Println("")
	fmt.Println("For more information, visit: https://github.com/pinepeakdigital/buzz")
}

func printVersion() {
	fmt.Printf("buzz version %s\n", version)
}

func main() {
	// Check for CLI arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "next":
			handleNextCommand()
			return
		case "today":
			handleTodayCommand()
			return
		case "add":
			handleAddCommand()
			return
		case "refresh":
			handleRefreshCommand()
			return
		case "view":
			handleViewCommand()
			return
		case "review":
			handleReviewCommand()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		case "-v", "--version", "version":
			printVersion()
			return
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			fmt.Println("Available commands: next, today, add, refresh, view, review, help, version")
			fmt.Println("Run 'buzz --help' for more information.")
			os.Exit(1)
		}
	}

	// No arguments, run the interactive TUI
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

// handleNextCommand outputs a terse summary of the next due goal
func handleNextCommand() {
	// Parse flags for the next command
	nextFlags := flag.NewFlagSet("next", flag.ContinueOnError)
	watch := nextFlags.Bool("watch", false, "Watch mode - continuously refresh every 5 minutes")
	watchShort := nextFlags.Bool("w", false, "Watch mode - continuously refresh every 5 minutes (shorthand)")
	if err := nextFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Help was requested; print usage and exit 0
			fmt.Println("Usage: buzz next [-w|--watch]")
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		fmt.Fprintln(os.Stderr, "Usage: buzz next [-w|--watch]")
		os.Exit(2)
	}
	if args := nextFlags.Args(); len(args) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown arguments: %v\n", args)
		fmt.Fprintln(os.Stderr, "Usage: buzz next [-w|--watch]")
		os.Exit(2)
	}

	// If either watch flag is set, enable watch mode
	watchMode := *watch || *watchShort

	if watchMode {
		runWatchMode()
	} else {
		// One-shot mode - display and exit
		if err := displayNextGoal(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// loadConfigAndGoals loads configuration and fetches sorted goals from Beeminder
func loadConfigAndGoals() (*Config, []Goal, error) {
	if !ConfigExists() {
		return nil, nil, fmt.Errorf("no configuration found. Please run 'buzz' first to authenticate")
	}

	config, err := LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	goals, err := FetchGoals(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch goals: %w", err)
	}

	SortGoals(goals)
	return config, goals, nil
}

// displayNextGoal fetches and displays the next due goal
// Returns error instead of calling os.Exit() for reusability in watch mode
func displayNextGoal() error {
	_, goals, err := loadConfigAndGoals()
	if err != nil {
		return err
	}

	// If no goals, return error
	if len(goals) == 0 {
		return fmt.Errorf("no goals found")
	}

	// Get the first goal (most urgent)
	nextGoal := goals[0]

	// Format the output: "goalslug baremin timeframe"
	timeframe := FormatDueDate(nextGoal.Losedate)

	// Output the terse summary
	fmt.Printf("%s %s %s\n", nextGoal.Slug, nextGoal.Baremin, timeframe)
	return nil
}

// runWatchMode runs the next command in watch mode with periodic refresh
func runWatchMode() {
	ticker := time.NewTicker(RefreshInterval)
	defer ticker.Stop()

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Initial display
	clearScreen()
	displayNextGoalWithTimestamp()

	for {
		select {
		case <-ticker.C:
			clearScreen()
			displayNextGoalWithTimestamp()
		case <-sigChan:
			fmt.Println("\nExiting...")
			return
		}
	}
}

// clearScreen clears the terminal screen
func clearScreen() {
	if fi, err := os.Stdout.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		return // not a terminal; skip clearing
	}
	fmt.Print("\033[2J\033[H")
}

// displayNextGoalWithTimestamp displays the next goal with a timestamp and refresh info
func displayNextGoalWithTimestamp() {
	fmt.Printf("[%s]\n", time.Now().Format("2006-01-02 15:04:05"))
	if err := displayNextGoal(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	fmt.Printf("\nRefreshing every %dm... (Press Ctrl+C to exit)\n", int(RefreshInterval.Minutes()))
}

// handleTodayCommand outputs all goals that are due today
func handleTodayCommand() {
	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Fetch goals
	goals, err := FetchGoals(config)
	if err != nil {
		fmt.Printf("Error: Failed to fetch goals: %v\n", err)
		os.Exit(1)
	}

	// Sort goals (by due date ascending, then by stakes descending, then by name)
	SortGoals(goals)

	// Filter goals that are due today
	var todayGoals []Goal
	for _, goal := range goals {
		if IsDueToday(goal.Losedate) {
			todayGoals = append(todayGoals, goal)
		}
	}

	// If no goals due today, exit
	if len(todayGoals) == 0 {
		fmt.Println("No goals due today.")
		return
	}

	// Calculate column widths for alignment
	maxSlugWidth := 0
	maxBareminWidth := 0
	for _, goal := range todayGoals {
		if len(goal.Slug) > maxSlugWidth {
			maxSlugWidth = len(goal.Slug)
		}
		if len(goal.Baremin) > maxBareminWidth {
			maxBareminWidth = len(goal.Baremin)
		}
	}

	// Output each goal on a separate line with aligned columns
	for _, goal := range todayGoals {
		timeframe := FormatDueDate(goal.Losedate)
		fmt.Printf("%-*s  %-*s  %s\n", maxSlugWidth, goal.Slug, maxBareminWidth, goal.Baremin, timeframe)
	}
}

// handleAddCommand adds a datapoint to a goal without opening the TUI
func handleAddCommand() {
	// Check arguments: buzz add <goalslug> <value> [comment]
	if len(os.Args) < 4 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: buzz add <goalslug> <value> [comment]")
		os.Exit(1)
	}

	goalSlug := os.Args[2]
	value := os.Args[3]

	// Optional comment - default to "Added via buzz" if not provided
	comment := "Added via buzz"
	if len(os.Args) >= 5 {
		comment = strings.Join(os.Args[4:], " ")
	}

	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Use current time as timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Convert time format to decimal hours if needed
	if isTimeFormat(value) {
		decimalValue, ok := timeToDecimalHours(value)
		if !ok {
			fmt.Printf("Error: Invalid time format: %s\n", value)
			os.Exit(1)
		}
		value = fmt.Sprintf("%.6g", decimalValue)
	}

	// Validate value is a number
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		fmt.Printf("Error: Value must be a valid number, got: %s\n", value)
		os.Exit(1)
	}

	// Create the datapoint
	err = CreateDatapoint(config, goalSlug, timestamp, value, comment)
	if err != nil {
		fmt.Printf("Error: Failed to add datapoint: %v\n", err)
		os.Exit(1)
	}

	// Signal any running TUI instances to refresh
	if err := createRefreshFlag(); err != nil {
		// Don't fail the command if flag creation fails
		fmt.Fprintf(os.Stderr, "Warning: Could not create refresh flag: %v\n", err)
	}

	fmt.Printf("Successfully added datapoint to %s: value=%s, comment=\"%s\"\n", goalSlug, value, comment)
}

// handleRefreshCommand refreshes autodata for a goal
func handleRefreshCommand() {
	// Check arguments: buzz refresh <goalslug>
	if len(os.Args) < 3 {
		fmt.Println("Error: Missing required argument")
		fmt.Println("Usage: buzz refresh <goalslug>")
		os.Exit(1)
	}

	goalSlug := os.Args[2]

	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Refresh the goal
	queued, err := RefreshGoal(config, goalSlug)
	if err != nil {
		fmt.Printf("Error: Failed to refresh goal: %v\n", err)
		os.Exit(1)
	}

	if queued {
		fmt.Printf("Successfully queued refresh for goal: %s\n", goalSlug)
	} else {
		fmt.Printf("Goal %s was not queued for refresh\n", goalSlug)
	}
}

// handleViewCommand displays detailed information about a specific goal
func handleViewCommand() {
	// Check arguments: buzz view <goalslug>
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: Missing required argument")
		fmt.Fprintln(os.Stderr, "Usage: buzz view <goalslug>")
		os.Exit(1)
	}

	goalSlug := os.Args[2]

	// Load config
	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Fetch the goal
	goal, err := FetchGoal(config, goalSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Display goal information
	fmt.Printf("Goal: %s\n", goal.Slug)
	fmt.Printf("Title:       %s\n", goal.Title)
	fmt.Printf("Limsum:      %s\n", goal.Limsum)
	fmt.Printf("Pledge:      $%.2f\n", goal.Pledge)
	fmt.Printf("Autodata:    %s\n", goal.Autodata)
	fmt.Printf("Autoratchet: %.0f\n", goal.Autoratchet)

	// Generate and display goal URL
	baseURL := getBaseURL(config)
	goalURL := fmt.Sprintf("%s/%s/%s", baseURL, url.PathEscape(config.Username), url.PathEscape(goal.Slug))
	fmt.Printf("URL:         %s\n", goalURL)
}

// handleReviewCommand launches an interactive review of all goals
func handleReviewCommand() {
	// Load config
	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Fetch goals
	goals, err := FetchGoals(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch goals: %v\n", err)
		os.Exit(1)
	}

	if len(goals) == 0 {
		fmt.Println("No goals found.")
		return
	}

	// Sort goals alphabetically by slug as specified
	SortGoalsBySlug(goals)

	// Launch the interactive review TUI
	p := tea.NewProgram(initialReviewModel(goals, config), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
