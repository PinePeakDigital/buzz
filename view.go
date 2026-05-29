package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

// handleViewCommand displays detailed information about a specific goal
func handleViewCommand() {
	// Parse flags for the view command. We support flags on either side of
	// the positional goal slug, so we loop over `viewFlags.Parse` until only
	// non-flag args remain — that way every valid Go-flag spelling
	// (`--web`, `-web`, `--json=true`, etc.) works whether it appears before
	// or after the slug, and unknown tokens still surface as errors instead
	// of being silently dropped.
	viewFlags := flag.NewFlagSet("view", flag.ContinueOnError)
	web := viewFlags.Bool("web", false, "Open the goal in the browser")
	jsonOutput := viewFlags.Bool("json", false, "Output goal data as JSON")
	datapoints := viewFlags.Bool("datapoints", false, "Include datapoints in output (use with --json)")

	const usage = "Usage: buzz view <goalslug> [--web] [--json] [--datapoints]"
	var positional []string
	remaining := os.Args[2:]
	for len(remaining) > 0 {
		if err := viewFlags.Parse(remaining); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				fmt.Println(usage)
				return
			}
			fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
			fmt.Fprintln(os.Stderr, usage)
			os.Exit(2)
		}
		rest := viewFlags.Args()
		if len(rest) == 0 {
			break
		}
		// Pull off the first non-flag token as a positional, then continue
		// re-parsing from the remainder so trailing flags get consumed.
		positional = append(positional, rest[0])
		remaining = rest[1:]
	}

	webFlag := *web
	jsonFlag := *jsonOutput
	datapointsFlag := *datapoints

	if len(positional) != 1 {
		if len(positional) == 0 {
			fmt.Fprintln(os.Stderr, "Error: Missing required argument")
		} else {
			fmt.Fprintf(os.Stderr, "Error: Too many arguments: %v\n", positional[1:])
		}
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	goalSlug := positional[0]

	// Load config
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

	// If --web flag is present, open in browser and exit
	if webFlag {
		if err := openBrowser(config, goalSlug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to open browser: %s\n", redactError(err))
			os.Exit(1)
		}
		return
	}

	// If --json flag is present, fetch and output raw JSON
	if jsonFlag {
		rawJSON, err := client.FetchGoalRawJSON(context.Background(), goalSlug, datapointsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
			os.Exit(1)
		}

		// Pretty print the raw JSON
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, rawJSON, "", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to format JSON: %s\n", redactError(err))
			os.Exit(1)
		}
		fmt.Println(prettyJSON.String())
		return
	}

	// Warn if --datapoints is used without --json
	if datapointsFlag {
		fmt.Fprintln(os.Stderr, "Warning: --datapoints flag has no effect without --json")
	}

	// Fetch the goal for human-readable output
	goal, err := client.FetchGoal(context.Background(), goalSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
		os.Exit(1)
	}

	// Display goal information (human-readable format)
	fmt.Printf("Goal: %s\n", goal.Slug)
	fmt.Print(formatGoalDetails(goal, config))

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}
