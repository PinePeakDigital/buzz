package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
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

	// Skip goals that have already reached their end value — they have no
	// remaining work, so surfacing them as "next" would mislead the user into
	// acting on a completed goal.
	goals = filterOutEndValueReached(goals)

	// Snapshot the time once so the overdue filter and the rendered countdown
	// share a single reference instant. Otherwise a goal could pass the filter
	// here and then render as OVERDUE moments later when formatted.
	now := time.Now()

	// Skip overdue goals: "next" should point at the soonest goal that still
	// has time left, not one that's already past its deadline (which would
	// render as OVERDUE rather than a countdown).
	goals = filterOutOverdue(goals, now)

	// If no goals, return error
	if len(goals) == 0 {
		return fmt.Errorf("no goals found")
	}

	// Get the first goal (most urgent)
	nextGoal := goals[0]

	// Format the output: "goalslug baremin timeframe"
	timeframe := FormatGoalDueDateAt(nextGoal, now)

	// Machine-readable formats emit just the goal (json = the raw object, csv =
	// one row), skipping the update banner so the output stays parseable.
	switch outputFormat {
	case "json":
		b, err := json.MarshalIndent(nextGoal, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	case "csv":
		var buf strings.Builder
		w := csv.NewWriter(&buf)
		w.Write([]string{"slug", "baremin", "due"})
		w.Write([]string{nextGoal.Slug, nextGoal.Baremin, timeframe})
		w.Flush()
		if err := w.Error(); err != nil {
			return err
		}
		fmt.Print(buf.String())
		return nil
	}

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
