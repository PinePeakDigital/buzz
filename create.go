package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

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
	{"drinker", "Do Less", "stay under a ceiling that ratchets down; the line is a cap you stay below (e.g. junk food, spending)"},
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
	code := runCreateCommand(os.Stdin, client, os.Stdout, os.Stderr)
	if code == 0 {
		// Check for updates and display message if available
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
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

	slug := promptField(r, stdout, "Goal slug: ")
	title := promptField(r, stdout, "Goal title: ")
	goalType := promptGoalType(r, stdout)
	gunits := promptField(r, stdout, "Goal units: ")

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Provide exactly 2 of the next 3 (leave one blank):")
	goaldate := promptField(r, stdout, "Goal date (epoch timestamp): ")
	goalval := promptField(r, stdout, "Goal value: ")
	rate := promptField(r, stdout, "Rate: ")

	if errMsg := validateCreateGoalInput(slug, title, goalType, gunits, goaldate, goalval, rate); errMsg != "" {
		fmt.Fprintf(stderr, "Error: %s\n", errMsg)
		return 1
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Creating goal...")

	goal, err := client.CreateGoal(context.Background(), slug, title, goalType, gunits, goaldate, goalval, rate)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to create goal: %s\n", redactError(err))
		return 1
	}

	fmt.Fprintf(stdout, "Successfully created goal: %s\n", goal.Slug)
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
