package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

// limsumFetchDelay is the duration to wait after adding a datapoint before fetching the updated limsum
// This gives the Beeminder server time to update the goal's limsum with the new datapoint
// This is a variable (not const) to allow tests to override it
var limsumFetchDelay = 2 * time.Second

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
	fmt.Println("  buzz tomorrow                     Output all goals due tomorrow")
	fmt.Println("  buzz less                         Output all do-less type goals")
	fmt.Println("  buzz add [--requestid=<id>] <goalslug> <value> [comment]")
	fmt.Println("                                    Add a datapoint to a goal")
	fmt.Println("                                    Note: Flags must come BEFORE positional args")
	fmt.Println("  echo \"<value>\" | buzz add [--requestid=<id>] <goalslug> [comment]")
	fmt.Println("                                    Add a datapoint with value from stdin")
	fmt.Println("  buzz refresh <goalslug>           Refresh autodata for a goal")
	fmt.Println("  buzz view <goalslug>              View detailed information about a specific goal")
	fmt.Println("  buzz view <goalslug> --web        Open the goal in the browser")
	fmt.Println("  buzz view <goalslug> --json       Output goal data as JSON")
	fmt.Println("  buzz view <goalslug> --json --datapoints  Include datapoints in JSON output")
	fmt.Println("  buzz review                       Interactive review of all goals")
	fmt.Println("  buzz charge <amount> <note> [--dryrun]")
	fmt.Println("                                    Create a charge for the authenticated user")
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

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
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
		case "tomorrow":
			handleTomorrowCommand()
			return
		case "less":
			handleLessCommand()
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
		case "charge":
			handleChargeCommand()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		case "-v", "--version", "version":
			printVersion()
			return
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			fmt.Println("Available commands: next, today, tomorrow, less, add, refresh, view, review, charge, help, version")
			fmt.Println("Run 'buzz --help' for more information.")
			os.Exit(1)
		}
	}

	// No arguments, run the interactive TUI
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %s", redactError(err))
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
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
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
			fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
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

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())

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
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
	}
	fmt.Printf("\nRefreshing every %dm... (Press Ctrl+C to exit)\n", int(RefreshInterval.Minutes()))
}

// isDueTodayFilter returns true if the goal is due today
func isDueTodayFilter(g Goal) bool {
	return IsDueToday(g.Losedate)
}

// isDueTomorrowFilter returns true if the goal is due tomorrow
func isDueTomorrowFilter(g Goal) bool {
	return IsDueTomorrow(g.Losedate)
}

// isDoLessFilter returns true if the goal is a do-less type goal
func isDoLessFilter(g Goal) bool {
	return IsDoLessGoal(g)
}

// handleTodayCommand outputs all goals that are due today
func handleTodayCommand() {
	handleFilteredCommand("today", isDueTodayFilter)
}

// handleTomorrowCommand outputs all goals that are due tomorrow
func handleTomorrowCommand() {
	handleFilteredCommand("tomorrow", isDueTomorrowFilter)
}

// handleLessCommand outputs all do-less type goals
func handleLessCommand() {
	handleFilteredCommand("do-less", isDoLessFilter)
}

// handleFilteredCommand is a shared helper that outputs all goals matching the given filter
// filterName is used in messages (e.g., "today", "tomorrow", or "do-less")
// filter is a function that takes a Goal and returns true if the goal matches
func handleFilteredCommand(filterName string, filter func(Goal) bool) {
	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	// Fetch goals
	goals, err := FetchGoals(config)
	if err != nil {
		fmt.Printf("Error: Failed to fetch goals: %s\n", redactError(err))
		os.Exit(1)
	}

	// Sort goals (by due date ascending, then by stakes descending, then by name)
	SortGoals(goals)

	// Filter goals that match the criteria
	var filteredGoals []Goal
	for _, goal := range goals {
		if filter(goal) {
			filteredGoals = append(filteredGoals, goal)
		}
	}

	// If no matching goals, exit
	if len(filteredGoals) == 0 {
		fmt.Printf("No %s goals found.\n", filterName)
		return
	}

	// Pre-calculate formatted values to avoid redundant formatting calls
	type goalDisplay struct {
		goal             Goal
		timeframe        string
		absoluteDeadline string
	}

	displays := make([]goalDisplay, len(filteredGoals))
	maxSlugWidth := 0
	maxBareminWidth := 0
	maxRelativeWidth := 0

	for i, goal := range filteredGoals {
		timeframe := FormatDueDate(goal.Losedate)
		absoluteDeadline := FormatAbsoluteDeadline(goal.Losedate)
		displays[i] = goalDisplay{
			goal:             goal,
			timeframe:        timeframe,
			absoluteDeadline: absoluteDeadline,
		}

		if len(goal.Slug) > maxSlugWidth {
			maxSlugWidth = len(goal.Slug)
		}
		if len(goal.Baremin) > maxBareminWidth {
			maxBareminWidth = len(goal.Baremin)
		}
		if len(timeframe) > maxRelativeWidth {
			maxRelativeWidth = len(timeframe)
		}
	}

	// Output each goal on a separate line with aligned columns
	for _, display := range displays {
		fmt.Printf("%-*s  %-*s  %-*s  %s\n",
			maxSlugWidth, display.goal.Slug,
			maxBareminWidth, display.goal.Baremin,
			maxRelativeWidth, display.timeframe,
			display.absoluteDeadline)
	}

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}

// printAddUsageAndExit prints the usage for buzz add command and exits with code 1
func printAddUsageAndExit(errorMsg string) {
	fmt.Println("Error: " + errorMsg)
	fmt.Println("Usage: buzz add [--requestid=<id>] <goalslug> <value> [comment]")
	fmt.Println("       echo \"<value>\" | buzz add [--requestid=<id>] <goalslug> [comment]")
	fmt.Println("")
	fmt.Println("Note: Flags (--requestid) must come BEFORE positional arguments.")
	fmt.Println("      Example: buzz add --requestid=ID goalslug value comment")
	os.Exit(1)
}

// handleAddCommand adds a datapoint to a goal without opening the TUI
func handleAddCommand() {
	// Parse flags for the add command
	addFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	requestid := addFlags.String("requestid", "", "Request ID for idempotency")
	if err := addFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println("Usage: buzz add [--requestid=<id>] <goalslug> <value> [comment]")
			fmt.Println("       echo \"<value>\" | buzz add [--requestid=<id>] <goalslug> [comment]")
			fmt.Println("")
			fmt.Println("Note: Flags must come BEFORE positional arguments.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
		printAddUsageAndExit("Invalid flags")
	}

	// Get remaining positional arguments after flag parsing
	args := addFlags.Args()

	// Detect if known flags appear after positional arguments and warn the user
	if misplacedFlag := detectMisplacedFlag(args); misplacedFlag != "" {
		fmt.Fprintf(os.Stderr, "Warning: Flag '%s' appears after positional arguments and will be treated as part of the comment.\n", misplacedFlag)
		fmt.Fprintf(os.Stderr, "Flags must come BEFORE positional arguments to be recognized.\n")
		fmt.Fprintf(os.Stderr, "Correct usage: buzz add --requestid=ID goalslug value comment\n")
		fmt.Fprintln(os.Stderr, "")
	}

	// Check arguments: buzz add <goalslug> <value> [comment]
	// Value can also be piped via stdin: echo "123" | buzz add mygoal [comment]
	if len(args) < 1 {
		printAddUsageAndExit("Missing required arguments")
	}

	goalSlug := args[0]
	var value string
	var commentStartIndex int // Index where optional comment starts in args

	// Try to read value from stdin first (for piped input)
	stdinValue, err := readValueFromStdin()
	if err == nil && stdinValue != "" {
		// Value provided via stdin
		value = stdinValue
		commentStartIndex = 1 // Comment starts at index 1 when value is piped
	} else if len(args) >= 2 {
		// Value provided as argument
		value = args[1]
		commentStartIndex = 2 // Comment starts at index 2
	} else {
		printAddUsageAndExit("Missing required value argument")
	}

	// Optional comment - default to "Added via buzz" if not provided
	comment := "Added via buzz"
	if len(args) >= commentStartIndex+1 {
		comment = strings.Join(args[commentStartIndex:], " ")
	}

	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %s\n", redactError(err))
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
	err = CreateDatapoint(config, goalSlug, timestamp, value, comment, *requestid)
	if err != nil {
		fmt.Printf("Error: Failed to add datapoint: %s\n", redactError(err))
		os.Exit(1)
	}

	// Signal any running TUI instances to refresh
	if err := createRefreshFlag(); err != nil {
		// Don't fail the command if flag creation fails
		fmt.Fprintf(os.Stderr, "Warning: Could not create refresh flag: %s\n", redactError(err))
	}

	successMsg := fmt.Sprintf("Successfully added datapoint to %s: value=%s, comment=\"%s\"", goalSlug, value, comment)
	if *requestid != "" {
		successMsg += fmt.Sprintf(", requestid=\"%s\"", *requestid)
	}
	fmt.Println(successMsg)

	// Wait briefly before fetching limsum to allow the server to update
	time.Sleep(limsumFetchDelay)

	// Fetch the goal to display the updated limsum
	goal, err := FetchGoal(config, goalSlug)
	if err != nil {
		// Don't fail the command if fetching limsum fails, just skip displaying it
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch goal status: %s\n", redactError(err))
	} else {
		fmt.Printf("Limsum: %s\n", goal.Limsum)
	}

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
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
		fmt.Printf("Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	// Refresh the goal
	queued, err := RefreshGoal(config, goalSlug)
	if err != nil {
		fmt.Printf("Error: Failed to refresh goal: %s\n", redactError(err))
		os.Exit(1)
	}

	if queued {
		fmt.Printf("Successfully queued refresh for goal: %s\n", goalSlug)
	} else {
		fmt.Printf("Goal %s was not queued for refresh\n", goalSlug)
	}

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}

// handleViewCommand displays detailed information about a specific goal
func handleViewCommand() {
	// Parse flags for the view command
	viewFlags := flag.NewFlagSet("view", flag.ContinueOnError)
	web := viewFlags.Bool("web", false, "Open the goal in the browser")
	jsonOutput := viewFlags.Bool("json", false, "Output goal data as JSON")
	datapoints := viewFlags.Bool("datapoints", false, "Include datapoints in output (use with --json)")
	if err := viewFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Help was requested; print usage and exit 0
			fmt.Println("Usage: buzz view <goalslug> [--web] [--json] [--datapoints]")
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(os.Stderr, "Usage: buzz view <goalslug> [--web] [--json] [--datapoints]")
		os.Exit(2)
	}

	// Get goal slug from remaining arguments
	args := viewFlags.Args()

	// Check if flags appear after the goal slug (handle both positions)
	webFlag := *web
	jsonFlag := *jsonOutput
	datapointsFlag := *datapoints
	var goalSlug string
	var filteredArgs []string

	for _, arg := range args {
		switch arg {
		case "--web":
			webFlag = true
		case "--json":
			jsonFlag = true
		case "--datapoints":
			datapointsFlag = true
		default:
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Missing required argument")
		fmt.Fprintln(os.Stderr, "Usage: buzz view <goalslug> [--web] [--json] [--datapoints]")
		os.Exit(1)
	}

	goalSlug = filteredArgs[0]

	// Load config
	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	// If --web flag is present, open in browser and exit
	if webFlag {
		if err := openBrowser(config, goalSlug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to open browser: %s\n", redactError(err))
			os.Exit(1)
		}
		return
	}

	// If --json flag is present, fetch and output raw JSON
	if jsonFlag {
		rawJSON, err := FetchGoalRawJSON(config, goalSlug, datapointsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
			os.Exit(1)
		}

		// Pretty print the raw JSON
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, rawJSON, "", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to format JSON: %s\n", redactError(err))
			os.Exit(1)
		}
		fmt.Println(prettyJSON.String())
		return
	}

	// Warn if --datapoints is used without --json
	if datapointsFlag {
		fmt.Fprintln(os.Stderr, "Warning: --datapoints flag has no effect without --json")
	}

	// Fetch the goal for human-readable output
	goal, err := FetchGoal(config, goalSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
		os.Exit(1)
	}

	// Display goal information (human-readable format)
	fmt.Printf("Goal: %s\n", goal.Slug)
	fmt.Print(formatGoalDetails(goal, config))

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
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
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	// Fetch goals
	goals, err := FetchGoals(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch goals: %s\n", redactError(err))
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
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
		os.Exit(1)
	}
}

// handleChargeCommand creates a charge for the authenticated user
func handleChargeCommand() {
	// Usage: buzz charge <amount> <note> [--dryrun]
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Error: Missing required arguments")
		fmt.Fprintln(os.Stderr, "Usage: buzz charge <amount> <note> [--dryrun]")
		os.Exit(1)
	}

	args := os.Args[2:]
	amountStr := args[0]
	// Collect note parts and allow --dryrun anywhere after amount
	dryrun := false
	var noteParts []string
	for _, a := range args[1:] {
		if a == "--dryrun" {
			dryrun = true
			continue
		}
		noteParts = append(noteParts, a)
	}
	note := strings.Join(noteParts, " ")
	if strings.TrimSpace(note) == "" {
		fmt.Fprintln(os.Stderr, "Error: Note is required")
		fmt.Fprintln(os.Stderr, "Usage: buzz charge <amount> <note> [--dryrun]")
		os.Exit(1)
	}

	// Validate amount is a number
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Amount must be a valid number, got: %s\n", amountStr)
		os.Exit(1)
	}

	// Validate amount is >= 1.00
	if amount < 1.00 {
		fmt.Fprintf(os.Stderr, "Error: Amount must be at least 1.00, got: %.2f\n", amount)
		os.Exit(1)
	}

	// Load config
	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	// Create the charge (API returns the created/dry-run charge)
	ch, err := CreateCharge(config, amount, note, dryrun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create charge: %s\n", redactError(err))
		os.Exit(1)
	}

	if dryrun {
		fmt.Printf("Dry run: Would charge $%.2f with note: %q for %s\n", ch.Amount, ch.Note, ch.Username)
	} else {
		fmt.Printf("Successfully created charge %s: $%.2f with note: %q for %s\n", ch.ID, ch.Amount, ch.Note, ch.Username)
	}

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}
