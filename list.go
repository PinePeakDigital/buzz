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
	archived, code, done := parseListArgs(os.Args[2:], os.Stdout, os.Stderr)
	if done {
		if code != 0 {
			os.Exit(code)
		}
		return
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
	code = runListCommand(context.Background(), client, archived, outputFormat, os.Stdout, os.Stderr)
	if code == 0 {
		// Check for updates and display message if available
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// parseListArgs parses the `buzz list` arguments (everything after the
// subcommand). It returns the --archived flag, a process exit code, and done:
// when done is true the caller should stop (help was printed, or a usage error
// occurred) and honor exitCode (0 = help/clean stop, non-zero = error). On the
// normal path done is false and exitCode is 0. Usage/errors are written to
// out/errOut rather than fixed streams so the parsing is unit-testable.
func parseListArgs(args []string, out, errOut io.Writer) (archived bool, exitCode int, done bool) {
	listFlags := flag.NewFlagSet("list", flag.ContinueOnError)
	// Silence flag's built-in error/usage printing so it doesn't duplicate (and
	// cross-stream) the explicit messages below; we own all user-facing output.
	listFlags.SetOutput(io.Discard)
	archivedFlag := listFlags.Bool("archived", false, "List archived goals instead of active ones")
	if err := listFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(out, "Usage: buzz list [--archived]")
			return false, 0, true
		}
		fmt.Fprintf(errOut, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(errOut, "Usage: buzz list [--archived]")
		return false, 2, true
	}
	if extra := listFlags.Args(); len(extra) > 0 {
		fmt.Fprintf(errOut, "Unknown arguments: %v\n", extra)
		fmt.Fprintln(errOut, "Usage: buzz list [--archived]")
		return false, 2, true
	}
	return *archivedFlag, 0, false
}

// runListCommand is the testable core of `buzz list`. It fetches the requested
// set of goals (active, or archived when archived is true), renders the table
// to out, writes any fetch error to errOut, and returns the process exit code.
// Splitting stdout (out) from stderr (errOut) keeps the table pipeable and
// matches the other command cores (e.g. runCreateCommand).
func runListCommand(ctx context.Context, client Client, archived bool, format string, out, errOut io.Writer) int {
	noun := "goals"
	fetch := client.FetchGoals
	if archived {
		noun = "archived goals"
		fetch = client.FetchArchivedGoals
	}

	goals, err := fetch(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "Error: Failed to fetch %s: %s\n", noun, redactError(err))
		return 1
	}

	// Sort goals alphabetically by slug for easy scanning
	SortGoalsBySlug(goals)

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

	// Machine-readable formats: emit just the data (json/csv handle the empty
	// case as [] / a header-only file), skipping the human summary header.
	if format != "table" {
		rendered, err := table.RenderAs(format, goals)
		if err != nil {
			fmt.Fprintf(errOut, "Error: %s\n", redactError(err))
			return 1
		}
		fmt.Fprint(out, rendered)
		return 0
	}

	if len(goals) == 0 {
		fmt.Fprintf(out, "No %s found.\n", noun)
		return 0
	}

	// Print summary header
	fmt.Fprintf(out, "Total %s: %d\n\n", noun, len(goals))
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
