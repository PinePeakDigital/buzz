package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const deadlineUsage = `Usage: buzz deadline [--yes|-y] <goalslug> <time>
  <time> can be:
    - 12-hour format: "3:00 PM", "11:30 AM"
    - 24-hour format: "15:00", "23:30"`

// deadlineRequest is a parsed, validated `buzz deadline` invocation.
type deadlineRequest struct {
	goalSlug    string
	offset      int // seconds from midnight
	skipConfirm bool
}

// handleDeadlineCommand changes the daily deadline for a goal. The wall-clock
// time parsing and seconds-from-midnight conversion live in timestr.go.
func handleDeadlineCommand() {
	req, code, done := parseDeadlineArgs(os.Args[2:], os.Stdout, os.Stderr)
	if done {
		os.Exit(code)
	}

	client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}

	code = runDeadlineCommand(req, os.Stdin, client, os.Stdout, os.Stderr)
	if code == 0 {
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// parseDeadlineArgs parses and validates `buzz deadline` arguments, returning
// the request, a process exit code, and done=true when the caller should stop
// (help shown, or a parse/validation error). It touches no config or network,
// so --help and bad input are handled without authentication.
func parseDeadlineArgs(args []string, stdout, stderr io.Writer) (deadlineRequest, int, bool) {
	deadlineFlags := flag.NewFlagSet("deadline", flag.ContinueOnError)
	// Silence the flag package's own output; we print our own usage.
	deadlineFlags.SetOutput(io.Discard)
	yes := deadlineFlags.Bool("yes", false, "Skip confirmation prompt")
	yesShort := deadlineFlags.Bool("y", false, "Skip confirmation prompt (shorthand)")
	if err := deadlineFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, deadlineUsage)
			return deadlineRequest{}, 0, true
		}
		fmt.Fprintf(stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(stderr, deadlineUsage)
		return deadlineRequest{}, 2, true
	}

	rest := deadlineFlags.Args()
	if len(rest) < 2 {
		fmt.Fprintln(stderr, "Error: Missing required arguments")
		fmt.Fprintln(stderr, deadlineUsage)
		return deadlineRequest{}, 1, true
	}

	offset, err := parseTimeToDeadlineOffset(strings.Join(rest[1:], " "))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return deadlineRequest{}, 1, true
	}

	return deadlineRequest{
		goalSlug:    rest[0],
		offset:      offset,
		skipConfirm: *yes || *yesShort,
	}, 0, false
}

// runDeadlineCommand applies the deadline change, prompting for confirmation on
// stdin unless skipConfirm is set, and returns the process exit code.
func runDeadlineCommand(req deadlineRequest, stdin io.Reader, client Client, stdout, stderr io.Writer) int {
	if !req.skipConfirm {
		// Fetch the current goal only when we actually need to render the
		// confirmation prompt — with --yes set, the pre-fetch is just an extra
		// API call that can fail before UpdateGoalDeadline gets a chance to run.
		currentGoal, err := client.FetchGoal(context.Background(), req.goalSlug)
		if err != nil {
			fmt.Fprintf(stderr, "Error: Failed to fetch goal: %s\n", redactError(err))
			return 1
		}
		fmt.Fprintf(stdout, "Change deadline for %s from %s to %s? [y/N] ",
			req.goalSlug, formatDueTime(currentGoal.Deadline), formatDueTime(req.offset))

		// EOF or empty input is treated as "no" so we never change a deadline
		// without explicit consent.
		line, _ := bufio.NewReader(stdin).ReadString('\n')
		response := strings.TrimSpace(strings.ToLower(line))
		if response != "y" && response != "yes" {
			fmt.Fprintln(stdout, "Cancelled.")
			return 0
		}
	}

	goal, err := client.UpdateGoalDeadline(context.Background(), req.goalSlug, req.offset)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to update deadline: %s\n", redactError(err))
		return 1
	}

	fmt.Fprintf(stdout, "Updated deadline for %s to %s\n", goal.Slug, formatDueTime(goal.Deadline))
	return 0
}
