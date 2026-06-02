package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// handleRatchetCommand removes safety buffer from a goal, leaving it with a
// specified number of days of buffer. The Beeminder ratchet endpoint only ever
// tightens a goal: requests that would add buffer are ignored by the server.
func handleRatchetCommand() {
	ratchetFlags := flag.NewFlagSet("ratchet", flag.ContinueOnError)
	ratchetFlags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: buzz ratchet [-y|--yes] <goalslug> <days>")
		fmt.Fprintln(os.Stderr, "  <days> is the number of days of safety buffer to leave on the goal")
	}
	yes := ratchetFlags.Bool("yes", false, "Skip the confirmation prompt")
	yesShort := ratchetFlags.Bool("y", false, "Skip the confirmation prompt (shorthand)")
	if err := ratchetFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			ratchetFlags.Usage()
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", err)
		ratchetFlags.Usage()
		os.Exit(2)
	}

	args := ratchetFlags.Args()
	if len(args) != 2 {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: Missing required arguments")
		} else {
			fmt.Fprintf(os.Stderr, "Error: Too many arguments: %v\n", args[2:])
		}
		ratchetFlags.Usage()
		os.Exit(1)
	}

	goalSlug := args[0]
	days, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid number of days %q: must be a whole number\n", args[1])
		os.Exit(1)
	}
	if days < 0 {
		fmt.Fprintln(os.Stderr, "Error: Number of days must not be negative")
		os.Exit(1)
	}

	skipConfirm := *yes || *yesShort

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

	if !skipConfirm {
		// Fetch the current goal only when we need to show the confirmation
		// prompt, so the --yes path doesn't pay for an extra API call that
		// can fail before the ratchet itself runs.
		currentGoal, err := client.FetchGoal(context.Background(), goalSlug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to fetch goal: %s\n", redactError(err))
			os.Exit(1)
		}
		fmt.Printf("Ratchet %s from %d to %d days of safety buffer? This removes buffer and cannot add it back. [y/N] ", goalSlug, currentGoal.Safebuf, days)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// EOF or read error in non-interactive contexts — treat as "no"
			// so we never remove buffer without an explicit affirmative.
			fmt.Println("Cancelled.")
			return
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return
		}
	}

	goal, err := client.RatchetGoal(context.Background(), goalSlug, days)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to ratchet goal: %s\n", redactError(err))
		os.Exit(1)
	}

	fmt.Printf("Ratcheted %s to %d days of safety buffer.\n", goal.Slug, goal.Safebuf)

	fmt.Print(getUpdateMessage())
}
