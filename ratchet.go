package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

// ratchetRequest is a parsed, validated `buzz ratchet` invocation.
type ratchetRequest struct {
	goalSlug    string
	days        int
	skipConfirm bool
}

// handleRatchetCommand removes safety buffer from a goal, leaving it with at
// most the specified number of days of buffer. The Beeminder ratchet endpoint
// only ever tightens a goal: requests that would add buffer are ignored by the
// server, so a goal already at or below the target is left unchanged.
func handleRatchetCommand() {
	req, code, done := parseRatchetArgs(os.Args[2:], os.Stderr)
	if done {
		os.Exit(code)
	}

	_, client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}

	code = runRatchetCommand(req, os.Stdin, client, os.Stdout, os.Stderr)
	if code == 0 {
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// parseRatchetArgs parses and validates `buzz ratchet` arguments, returning the
// request, a process exit code, and done=true when the caller should stop (help
// shown, or a parse/validation error). It touches no config or network, so
// --help and bad input are handled without authentication.
func parseRatchetArgs(args []string, stderr io.Writer) (ratchetRequest, int, bool) {
	ratchetFlags := flag.NewFlagSet("ratchet", flag.ContinueOnError)
	ratchetFlags.SetOutput(stderr)
	ratchetFlags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: buzz ratchet [-y|--yes] <goalslug> <days>")
		fmt.Fprintln(stderr, "  <days> is the number of days of safety buffer to leave on the goal")
	}
	yes := ratchetFlags.Bool("yes", false, "Skip the confirmation prompt")
	yesShort := ratchetFlags.Bool("y", false, "Skip the confirmation prompt (shorthand)")
	if err := ratchetFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			ratchetFlags.Usage()
			return ratchetRequest{}, 0, true
		}
		fmt.Fprintf(stderr, "Error parsing flags: %s\n", err)
		ratchetFlags.Usage()
		return ratchetRequest{}, 2, true
	}

	rest := ratchetFlags.Args()
	if len(rest) != 2 {
		if len(rest) < 2 {
			fmt.Fprintln(stderr, "Error: Missing required arguments")
		} else {
			fmt.Fprintf(stderr, "Error: Too many arguments: %v\n", rest[2:])
		}
		ratchetFlags.Usage()
		return ratchetRequest{}, 1, true
	}

	days, err := strconv.Atoi(rest[1])
	if err != nil {
		fmt.Fprintf(stderr, "Error: Invalid number of days %q: must be a whole number\n", rest[1])
		return ratchetRequest{}, 1, true
	}
	if days < 0 {
		fmt.Fprintln(stderr, "Error: Number of days must not be negative")
		return ratchetRequest{}, 1, true
	}

	return ratchetRequest{
		goalSlug:    rest[0],
		days:        days,
		skipConfirm: *yes || *yesShort,
	}, 0, false
}

// runRatchetCommand applies the ratchet, prompting for confirmation on stdin
// unless skipConfirm is set, and returns the process exit code.
func runRatchetCommand(req ratchetRequest, stdin io.Reader, client Client, stdout, stderr io.Writer) int {
	if !req.skipConfirm {
		// Fetch the current goal only when we need to show the confirmation
		// prompt, so the --yes path doesn't pay for an extra API call that
		// can fail before the ratchet itself runs.
		currentGoal, err := client.FetchGoal(context.Background(), req.goalSlug)
		if err != nil {
			fmt.Fprintf(stderr, "Error: Failed to fetch goal: %s\n", redactError(err))
			return 1
		}
		var prompt string
		if currentGoal.Safebuf <= req.days {
			prompt = fmt.Sprintf("%s already has %d days of safety buffer, which is at or below %d days. No buffer will be removed. Continue anyway? [y/N] ", req.goalSlug, currentGoal.Safebuf, req.days)
		} else {
			prompt = fmt.Sprintf("Ratchet %s from %d to at most %d days of safety buffer? This removes buffer and cannot add it back. [y/N] ", req.goalSlug, currentGoal.Safebuf, req.days)
		}
		if !confirm(stdin, stdout, prompt) {
			return 0
		}
	}

	goal, err := client.RatchetGoal(context.Background(), req.goalSlug, req.days)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to ratchet goal: %s\n", redactError(err))
		return 1
	}

	fmt.Fprintf(stdout, "Ratcheted %s to %d days of safety buffer.\n", goal.Slug, goal.Safebuf)
	return 0
}
