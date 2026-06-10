package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

// handleListCommand outputs a summary list of goals with slug, title, units,
// rate, and stakes. With --archived it lists archived goals instead of active
// ones.
func handleListCommand() {
	listFlags := flag.NewFlagSet("list", flag.ContinueOnError)
	archived := listFlags.Bool("archived", false, "List archived goals instead of active ones")
	if err := listFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println("Usage: buzz list [--archived]")
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(os.Stderr, "Usage: buzz list [--archived]")
		os.Exit(2)
	}
	if args := listFlags.Args(); len(args) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown arguments: %v\n", args)
		fmt.Fprintln(os.Stderr, "Usage: buzz list [--archived]")
		os.Exit(2)
	}

	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)
	code := runListCommand(context.Background(), client, *archived, os.Stdout)
	if code == 0 {
		// Check for updates and display message if available
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// runListCommand is the testable core of `buzz list`. It fetches the requested
// set of goals (active, or archived when archived is true), renders them as a
// table to out, and returns the process exit code.
func runListCommand(ctx context.Context, client Client, archived bool, out io.Writer) int {
	noun := "goals"
	fetch := client.FetchGoals
	if archived {
		noun = "archived goals"
		fetch = client.FetchArchivedGoals
	}

	goals, err := fetch(ctx)
	if err != nil {
		fmt.Fprintf(out, "Error: Failed to fetch %s: %s\n", noun, redactError(err))
		return 1
	}

	// Sort goals alphabetically by slug for easy scanning
	SortGoalsBySlug(goals)

	if len(goals) == 0 {
		fmt.Fprintf(out, "No %s found.\n", noun)
		return 0
	}

	// Print summary header
	fmt.Fprintf(out, "Total %s: %d\n\n", noun, len(goals))

	table := Table{
		ShowHeader: true,
		Columns: []Column{
			{Header: "Slug", Cell: func(g Goal) string { return g.Slug }},
			{Header: "Title", Cell: func(g Goal) string {
				if g.Title == "" {
					return "-"
				}
				return g.Title
			}},
			{Header: "Units", Cell: func(g Goal) string { return getDisplayUnits(g.Gunits) }},
			{Header: "Rate", Cell: func(g Goal) string { return formatListRate(g.Rate, g.Runits) }},
			{Header: "Stakes", Cell: func(g Goal) string { return fmt.Sprintf("$%.2f", g.Pledge) }},
		},
	}
	fmt.Fprint(out, table.Render(goals))

	return 0
}

// getDisplayUnits returns the display value for goal units, using "-" if empty
func getDisplayUnits(gunits string) string {
	if gunits == "" {
		return "-"
	}
	return gunits
}

// formatListRate formats the rate value with its units for the list command
func formatListRate(rate *float64, runits string) string {
	if rate == nil {
		return "-"
	}
	// Format rate to remove unnecessary decimal places
	rateVal := *rate
	if rateVal == float64(int(rateVal)) {
		// Integer value - no decimal places
		return fmt.Sprintf("%d/%s", int(rateVal), runits)
	}
	// Has decimal - use %.6g to show up to 6 significant digits, trimming trailing zeros
	return fmt.Sprintf("%.6g/%s", rateVal, runits)
}
