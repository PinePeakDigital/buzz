package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// defaultGoalType is used when the user leaves the goal type prompt blank.
const defaultGoalType = "hustler"

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
	goalType := promptField(r, stdout, fmt.Sprintf("Goal type (default: %s): ", defaultGoalType))
	if goalType == "" {
		goalType = defaultGoalType
	}
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
