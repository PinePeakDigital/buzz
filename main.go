package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	// Embed the IANA timezone database so time.LoadLocation works on systems
	// without system tzdata (e.g. Windows, minimal containers). The schedule
	// command relies on this to render deadlines in the account timezone.
	_ "time/tzdata"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// version is set via ldflags during build
var version = "dev"

// outputFormat holds the global --format value ("table", "json", or "csv"),
// set once in main from the CLI. The list-style read commands, `data`, and
// `next` honor it; other commands ignore it (like --no-color).
var outputFormat = "table"

// validFormats are the accepted --format values.
var validFormats = map[string]bool{"table": true, "json": true, "csv": true}

func printHelp() {
	fmt.Println("buzz - A terminal user interface for Beeminder")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  buzz                              Launch the interactive TUI")
	fmt.Println("  buzz next                         Output a terse summary of the next due goal")
	fmt.Println("  buzz next --watch                 Watch mode - continuously refresh every 5 minutes")
	fmt.Println("  buzz next -w                      Watch mode (shorthand)")
	fmt.Println("  buzz list                         List all goals with slug, title, units, rate, and stakes")
	fmt.Println("  buzz list --archived              List archived goals instead of active ones")
	fmt.Println("  buzz all                          Output all goals")
	fmt.Println("  buzz today                        Output all goals due today")
	fmt.Println("  buzz tomorrow                     Output all goals due tomorrow")
	fmt.Println("  buzz due <duration>               Output all goals due within duration (e.g., 10m, 1h, 5d, 1w)")
	fmt.Println("  buzz less                         Output all do-less type goals")
	fmt.Println("  buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> <value> [comment]")
	fmt.Println("                                    Add a datapoint to a goal")
	fmt.Println("                                    --daystamp: Date in YYYYMMDD format (default: current time)")
	fmt.Println("                                    Note: Flags must come BEFORE positional args")
	fmt.Println("  echo \"<value>\" | buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> [comment]")
	fmt.Println("                                    Add a datapoint with value from stdin")
	fmt.Println("  buzz refresh <goalslug>           Refresh autodata for a goal")
	fmt.Println("  buzz view <goalslug>              View detailed information about a specific goal")
	fmt.Println("  buzz view <goalslug> --web        Open the goal in the browser")
	fmt.Println("  buzz view <goalslug> --json       Output goal data as JSON")
	fmt.Println("  buzz view <goalslug> --json --datapoints  Include datapoints in JSON output")
	fmt.Println("  buzz data [--asc|--desc] <goalslug>")
	fmt.Println("                                    List a goal's datapoints (date, value, comment)")
	fmt.Println("                                    --asc: oldest-first (default)  --desc: newest-first")
	fmt.Println("  buzz review                       Interactive review of all goals")
	fmt.Println("  buzz charge <amount> <note> [--dryrun]")
	fmt.Println("                                    Create a charge for the authenticated user")
	fmt.Println("  buzz create                       Interactively create a new Beeminder goal")
	fmt.Println("  buzz create --slug=<s> --units=<u> [--title --type --goaldate --goalval --rate --deadline]")
	fmt.Println("                                    Non-interactively create a goal (see --help)")
	fmt.Println("  buzz deadline [--yes] <goalslug> <time>")
	fmt.Println("                                    Change a goal's deadline (e.g., \"3:00 PM\" or \"15:00\")")
	fmt.Println("  buzz schedule                     Display goal deadline distribution throughout a 24-hour day")
	fmt.Println("  buzz uncle [-y|--yes] <goalslug>  Instantly derail a goal that is in the red, paying the pledge")
	fmt.Println("                                    -y, --yes: Skip the confirmation prompt")
	fmt.Println("  buzz ratchet [-y|--yes] <goalslug> <days>")
	fmt.Println("                                    Remove safety buffer, leaving <days> of buffer on the goal")
	fmt.Println("                                    -y, --yes: Skip the confirmation prompt")
	fmt.Println("  buzz api [-X <method>] [-d <key=value>]... <path>")
	fmt.Println("                                    Make a raw authenticated Beeminder API request")
	fmt.Println("                                    e.g. buzz api users/me.json")
	fmt.Println("  buzz auth login                   Authenticate by pasting your Beeminder API credentials")
	fmt.Println("  buzz help                         Show this help message")
	fmt.Println("")
	fmt.Println("GLOBAL OPTIONS:")
	fmt.Println("  --format <table|json|csv>         Output format for the list commands, data, and next (default: table)")
	fmt.Println("  --no-color                        Disable colored output")
	fmt.Println("  -h, --help                        Show this help message")
	fmt.Println("  -v, --version                     Show version information")
	fmt.Println("")
	fmt.Println("For more information, visit: https://buzz.nathanarthur.com")
}

func printVersion() {
	fmt.Printf("buzz version %s\n", version)

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}

// parseNoColorFlag extracts the --no-color flag from the provided arguments
// and returns whether the flag was found and the filtered arguments without the flag
func parseNoColorFlag(args []string) (noColor bool, filteredArgs []string) {
	filteredArgs = []string{args[0]} // Keep program name
	for i := 1; i < len(args); i++ {
		if args[i] == "--no-color" {
			noColor = true
		} else {
			filteredArgs = append(filteredArgs, args[i])
		}
	}
	return noColor, filteredArgs
}

// parseFormatFlag extracts a global --format <value> (or --format=<value>) flag
// from args, returning the chosen format ("table" when absent) and args with
// the flag removed. A missing or unknown value is an error.
func parseFormatFlag(args []string) (format string, filteredArgs []string, err error) {
	format = "table"
	filteredArgs = []string{args[0]} // Keep program name
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--format":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--format requires a value (table, json, or csv)")
			}
			format = args[i+1]
			i++
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
		default:
			filteredArgs = append(filteredArgs, arg)
			continue
		}
		if !validFormats[format] {
			return "", nil, fmt.Errorf("invalid --format value %q (want table, json, or csv)", format)
		}
	}
	return format, filteredArgs, nil
}

func main() {
	// Check for global --no-color flag before processing other commands
	noColor, filteredArgs := parseNoColorFlag(os.Args)
	os.Args = filteredArgs

	// Disable colors if --no-color flag is present
	if noColor {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	// Extract the global --format flag before command dispatch, mirroring
	// --no-color. Handlers read outputFormat; unknown values fail fast.
	format, formatFiltered, err := parseFormatFlag(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(2)
	}
	os.Args = formatFiltered
	outputFormat = format

	// Check for CLI arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "next":
			handleNextCommand()
			return
		case "list":
			handleListCommand()
			return
		case "all":
			handleAllCommand()
			return
		case "today":
			handleTodayCommand()
			return
		case "tomorrow":
			handleTomorrowCommand()
			return
		case "due":
			handleDueCommand()
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
		case "data":
			handleDataCommand()
			return
		case "review":
			handleReviewCommand()
			return
		case "charge":
			handleChargeCommand()
			return
		case "create":
			handleCreateCommand()
			return
		case "deadline":
			handleDeadlineCommand()
			return
		case "schedule":
			handleScheduleCommand()
			return
		case "uncle":
			handleUncleCommand()
			return
		case "ratchet":
			handleRatchetCommand()
			return
		case "api":
			handleAPICommand()
			return
		case "auth":
			handleAuthCommand()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		case "-v", "--version", "version":
			printVersion()
			return
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			fmt.Println("Available commands: next, list, all, today, tomorrow, due, less, add, refresh, view, data, review, charge, create, deadline, schedule, uncle, ratchet, api, auth, help, version")
			fmt.Println("Run 'buzz --help' for more information.")
			os.Exit(1)
		}
	}

	// No arguments, run the interactive TUI. The cancellable context is
	// stored on the model and threaded into every Client call; the deferred
	// cancel fires when p.Run() returns (user quit, error, or signal) so
	// any in-flight HTTP request aborts instead of hanging until the 30s
	// http.Client.Timeout fires.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := tea.NewProgram(initialModel(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %s", redactError(err))
		os.Exit(1)
	}
}

// loadConfigAndGoals loads configuration, constructs an HTTP client, and fetches
// sorted goals from Beeminder. Returns the client so callers can make further API
// calls without rebuilding it. Lives here because both next and schedule
// commands use it.
func loadConfigAndGoals() (*Config, Client, []Goal, error) {
	if !ConfigExists() {
		return nil, nil, nil, fmt.Errorf("no configuration found. Please run 'buzz auth login' to authenticate")
	}

	config, err := LoadConfig()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	client := NewHTTPClient(config)
	goals, err := client.FetchGoals(context.Background())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch goals: %w", err)
	}

	SortGoals(goals)
	return config, client, goals, nil
}
