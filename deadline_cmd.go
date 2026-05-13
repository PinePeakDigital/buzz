package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// handleDeadlineCommand changes the daily deadline for a goal. The wall-clock
// time parsing and seconds-from-midnight conversion live in timestr.go.
func handleDeadlineCommand() {
	// Usage: buzz deadline [--yes] <goalslug> <time>
	deadlineFlags := flag.NewFlagSet("deadline", flag.ContinueOnError)
	yes := deadlineFlags.Bool("yes", false, "Skip confirmation prompt")
	yesShort := deadlineFlags.Bool("y", false, "Skip confirmation prompt (shorthand)")
	if err := deadlineFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "Usage: buzz deadline [--yes|-y] <goalslug> <time>")
			fmt.Fprintln(os.Stderr, "  <time> can be:")
			fmt.Fprintln(os.Stderr, "    - 12-hour format: \"3:00 PM\", \"11:30 AM\"")
			fmt.Fprintln(os.Stderr, "    - 24-hour format: \"15:00\", \"23:30\"")
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(os.Stderr, "Usage: buzz deadline [--yes|-y] <goalslug> <time>")
		os.Exit(2)
	}

	args := deadlineFlags.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: Missing required arguments")
		fmt.Fprintln(os.Stderr, "Usage: buzz deadline [--yes|-y] <goalslug> <time>")
		fmt.Fprintln(os.Stderr, "  <time> can be:")
		fmt.Fprintln(os.Stderr, "    - 12-hour format: \"3:00 PM\", \"11:30 AM\"")
		fmt.Fprintln(os.Stderr, "    - 24-hour format: \"15:00\", \"23:30\"")
		os.Exit(1)
	}

	skipConfirm := *yes || *yesShort
	goalSlug := args[0]
	timeStr := strings.Join(args[1:], " ")

	offset, err := parseTimeToDeadlineOffset(timeStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)

	if !skipConfirm {
		// Fetch the current goal only when we actually need to render the
		// confirmation prompt — with --yes set, the pre-fetch is just an
		// extra API call that can fail before UpdateGoalDeadline gets a
		// chance to run.
		currentGoal, err := client.FetchGoal(goalSlug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to fetch goal: %s\n", redactError(err))
			os.Exit(1)
		}
		newTime := formatDueTime(offset)
		currentTime := formatDueTime(currentGoal.Deadline)
		fmt.Printf("Change deadline for %s from %s to %s? [y/N] ", goalSlug, currentTime, newTime)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// EOF or read error in non-interactive contexts — treat as
			// "no" so we never change a deadline without explicit consent.
			fmt.Println("Cancelled.")
			return
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return
		}
	}

	goal, err := client.UpdateGoalDeadline(goalSlug, offset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to update deadline: %s\n", redactError(err))
		os.Exit(1)
	}

	fmt.Printf("Updated deadline for %s to %s\n", goal.Slug, formatDueTime(goal.Deadline))

	fmt.Print(getUpdateMessage())
}
