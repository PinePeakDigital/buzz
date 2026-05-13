package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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

// displayNextGoal fetches and displays the next due goal
// Returns error instead of calling os.Exit() for reusability in watch mode
func displayNextGoal() error {
	_, _, goals, err := loadConfigAndGoals()
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
