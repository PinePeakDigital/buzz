package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// handleUncleCommand instantly derails a goal that is in the red.
func handleUncleCommand() {
	uncleFlags := flag.NewFlagSet("uncle", flag.ContinueOnError)
	uncleFlags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: buzz uncle [-y|--yes] <goalslug>")
	}
	yes := uncleFlags.Bool("yes", false, "Skip the confirmation prompt")
	yesShort := uncleFlags.Bool("y", false, "Skip the confirmation prompt (shorthand)")
	if err := uncleFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			uncleFlags.Usage()
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", err)
		fmt.Fprintln(os.Stderr, "Usage: buzz uncle [-y|--yes] <goalslug>")
		os.Exit(2)
	}

	args := uncleFlags.Args()
	if len(args) != 1 {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: Missing required argument")
		} else {
			fmt.Fprintf(os.Stderr, "Error: Too many arguments: %v\n", args[1:])
		}
		fmt.Fprintln(os.Stderr, "Usage: buzz uncle [-y|--yes] <goalslug>")
		os.Exit(1)
	}

	goalSlug := args[0]
	skipConfirm := *yes || *yesShort

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
		fmt.Printf("Call uncle on %s? This will instantly derail the goal and charge the pledge. [y/N] ", goalSlug)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return
		}
	}

	goal, err := client.CallUncle(goalSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to call uncle: %s\n", redactError(err))
		os.Exit(1)
	}

	fmt.Printf("Called uncle on %s. The goal has been derailed.\n", goal.Slug)

	fmt.Print(getUpdateMessage())
}
