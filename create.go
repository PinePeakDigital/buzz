package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// createUsage documents the non-interactive flag form of `buzz create`.
const createUsage = `Usage: buzz create                 (interactive; prompts for each field)
       buzz create [flags]         (non-interactive; scriptable)

Flags:
  --slug       Goal slug (required)
  --units      Goal units (required)
  --title      Goal title (defaults to the slug if omitted)
  --type       Goal type name/label/number (default: hustler)
  --goaldate   Goal date as an epoch timestamp
  --goalval    Goal value
  --rate       Rate
  --deadline   Deadline in seconds from midnight (may be negative)

Provide exactly 2 of --goaldate, --goalval, --rate.`

// createRequest is a fully-gathered `buzz create` invocation, from either the
// interactive prompts or CLI flags, ready to validate and send.
type createRequest struct {
	slug, title, goalType, gunits string
	goaldate, goalval, rate       string
	deadline                      int
	setDeadline                   bool // whether --deadline was explicitly passed
}

// defaultGoalType is used when the user leaves the goal type prompt blank.
const defaultGoalType = "hustler"

// goalTypeOption describes a Beeminder goal type: its canonical goal_type value
// (name, what the API expects), a human-friendly label, and a one-line
// explanation of how the goal behaves.
type goalTypeOption struct {
	name  string
	label string
	desc  string
}

// goalTypeOptions lists Beeminder's goal types in the order shown in the create
// menu. The labels and descriptions exist so users don't have to memorize the
// jargon names (e.g. "hustler" really means "Do More").
var goalTypeOptions = []goalTypeOption{
	{"hustler", "Do More", "accumulate at least a set amount; the line is a floor you stay above (e.g. exercise, writing)"},
	{"drinker", "Do Less", "keep a running total under a limit; the line is a ceiling you stay below (e.g. junk food, spending)"},
	{"biker", "Odometer", "track a running total that only goes up and occasionally resets, like a car odometer"},
	{"fatloser", "Weight loss", "drive a value down toward a target; the line slopes downward (e.g. lose weight)"},
	{"gainer", "Gain Weight", "drive a value up at a steady rate (e.g. gain weight or muscle)"},
	{"inboxer", "Inbox Fewer", "whittle a count down toward zero and keep it there (e.g. email inbox, open bugs)"},
	{"custom", "Custom", "full manual control over the underlying goal parameters"},
}

// handleCreateCommand creates a new Beeminder goal by prompting interactively
// for its fields. It mirrors the TUI's create-goal modal (the `n` key) but as a
// plain CLI command, reusing the same validateCreateGoalInput validation and the
// client's CreateGoal method.
func handleCreateCommand() {
	args := os.Args[2:]
	interactive := len(args) == 0

	// Parse flags first (when present) so `--help` and bad input are reported
	// without requiring authentication, matching `buzz add`.
	var req createRequest
	if !interactive {
		var code int
		var done bool
		req, code, done = parseCreateArgs(args, os.Stdout, os.Stderr)
		if done {
			os.Exit(code)
		}
	}

	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)
	var code int
	if interactive {
		code = runCreateCommand(os.Stdin, client, os.Stdout, os.Stderr)
	} else {
		code = doCreate(req, client, os.Stdout, os.Stderr)
	}
	if code == 0 {
		// Check for updates and display message if available
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// parseCreateArgs parses non-interactive `buzz create` flags into a request. It
// returns a process exit code and done=true when the caller should stop (help
// shown or a parse error).
func parseCreateArgs(args []string, stdout, stderr io.Writer) (createRequest, int, bool) {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // we print our own richer usage
	slug := fs.String("slug", "", "Goal slug")
	title := fs.String("title", "", "Goal title (defaults to slug)")
	goalType := fs.String("type", defaultGoalType, "Goal type")
	gunits := fs.String("units", "", "Goal units")
	goaldate := fs.String("goaldate", "", "Goal date (epoch timestamp)")
	goalval := fs.String("goalval", "", "Goal value")
	rate := fs.String("rate", "", "Rate")
	deadline := fs.Int("deadline", 0, "Deadline in seconds from midnight")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, createUsage)
			return createRequest{}, 0, true
		}
		fmt.Fprintf(stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(stderr, createUsage)
		return createRequest{}, 1, true
	}

	// Detect whether --deadline was explicitly set: 0 (midnight) is a valid
	// deadline, so we can't infer intent from the value alone.
	setDeadline := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "deadline" {
			setDeadline = true
		}
	})

	return createRequest{
		slug: *slug, title: *title, goalType: *goalType, gunits: *gunits,
		goaldate: *goaldate, goalval: *goalval, rate: *rate,
		deadline: *deadline, setDeadline: setDeadline,
	}, 0, false
}

// runCreateCommand is the testable core of `buzz create`. It prompts for goal
// fields on stdin, validates them, creates the goal, and returns the process
// exit code. Reading line-by-line means it also works with piped input for
// scripting (e.g. `printf '...' | buzz create`).
func runCreateCommand(stdin io.Reader, client Client, stdout, stderr io.Writer) int {
	r := bufio.NewReader(stdin)

	fmt.Fprintln(stdout, "Create a new Beeminder goal")
	fmt.Fprintln(stdout, "===========================")
	fmt.Fprintln(stdout, "")

	req := createRequest{
		slug:     promptField(r, stdout, "Goal slug: "),
		title:    promptField(r, stdout, "Goal title (defaults to slug): "),
		goalType: promptGoalType(r, stdout),
		gunits:   promptField(r, stdout, "Goal units: "),
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Provide exactly 2 of the next 3 (leave one blank):")
	req.goaldate = promptField(r, stdout, "Goal date (epoch timestamp): ")
	req.goalval = promptField(r, stdout, "Goal value: ")
	req.rate = promptField(r, stdout, "Rate: ")

	return doCreate(req, client, stdout, stderr)
}

// doCreate validates a gathered request, creates the goal, and (if requested)
// sets its deadline. Shared by the interactive and non-interactive paths. Title
// defaults to the slug when omitted, so callers needn't supply one.
func doCreate(req createRequest, client Client, stdout, stderr io.Writer) int {
	if req.title == "" {
		req.title = req.slug
	}

	if errMsg := validateCreateGoalInput(req.slug, req.title, req.goalType, req.gunits, req.goaldate, req.goalval, req.rate); errMsg != "" {
		fmt.Fprintf(stderr, "Error: %s\n", errMsg)
		return 1
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Creating goal...")

	goal, err := client.CreateGoal(context.Background(), req.slug, req.title, req.goalType, req.gunits, req.goaldate, req.goalval, req.rate)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to create goal: %s\n", redactError(err))
		return 1
	}

	fmt.Fprintf(stdout, "Successfully created goal: %s\n", goal.Slug)

	if req.setDeadline {
		if _, err := client.UpdateGoalDeadline(context.Background(), goal.Slug, req.deadline); err != nil {
			fmt.Fprintf(stderr, "Error: Goal created but failed to set deadline: %s\n", redactError(err))
			return 1
		}
		fmt.Fprintf(stdout, "Set deadline: %d seconds from midnight\n", req.deadline)
	}

	return 0
}

// promptField writes a prompt and reads a single trimmed line of input. A read
// error (including EOF before a newline) still returns whatever was read so far;
// missing required fields are caught by validation rather than here.
func promptField(r *bufio.Reader, w io.Writer, label string) string {
	fmt.Fprint(w, label)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

// promptGoalType prints the explained goal-type menu and resolves the user's
// choice to a canonical goal_type value. Accepted input: a number from the list,
// a type's name or label (e.g. "hustler" or "Do More", case-insensitive — keeps
// piped/scripted input working), or blank to take the default. An unrecognized
// non-empty entry is passed through unchanged so newly-added Beeminder types
// still work; the API validates the final value.
func promptGoalType(r *bufio.Reader, w io.Writer) string {
	fmt.Fprintln(w, "Goal type:")
	for i, gt := range goalTypeOptions {
		marker := ""
		if gt.name == defaultGoalType {
			marker = " [default]"
		}
		fmt.Fprintf(w, "  %d. %s (%s)%s — %s\n", i+1, gt.label, gt.name, marker, gt.desc)
	}

	choice := promptField(r, w, fmt.Sprintf("Choose a number or name (default: %s): ", defaultGoalType))
	if choice == "" {
		return defaultGoalType
	}

	// Numeric selection from the menu.
	if n, err := strconv.Atoi(choice); err == nil && n >= 1 && n <= len(goalTypeOptions) {
		return goalTypeOptions[n-1].name
	}

	// Match a canonical name or human label, case-insensitively.
	for _, gt := range goalTypeOptions {
		if strings.EqualFold(choice, gt.name) || strings.EqualFold(choice, gt.label) {
			return gt.name
		}
	}

	// Pass through anything else; validation / the API decides if it's valid.
	return choice
}
