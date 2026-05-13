package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

// handleViewCommand displays detailed information about a specific goal
func handleViewCommand() {
	// Parse flags for the view command
	viewFlags := flag.NewFlagSet("view", flag.ContinueOnError)
	web := viewFlags.Bool("web", false, "Open the goal in the browser")
	jsonOutput := viewFlags.Bool("json", false, "Output goal data as JSON")
	datapoints := viewFlags.Bool("datapoints", false, "Include datapoints in output (use with --json)")
	if err := viewFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Help was requested; print usage and exit 0
			fmt.Println("Usage: buzz view <goalslug> [--web] [--json] [--datapoints]")
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(os.Stderr, "Usage: buzz view <goalslug> [--web] [--json] [--datapoints]")
		os.Exit(2)
	}

	// Get goal slug from remaining arguments
	args := viewFlags.Args()

	// Check if flags appear after the goal slug (handle both positions)
	webFlag := *web
	jsonFlag := *jsonOutput
	datapointsFlag := *datapoints
	var goalSlug string
	var filteredArgs []string

	for _, arg := range args {
		switch arg {
		case "--web":
			webFlag = true
		case "--json":
			jsonFlag = true
		case "--datapoints":
			datapointsFlag = true
		default:
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) != 1 {
		if len(filteredArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Error: Missing required argument")
		} else {
			fmt.Fprintf(os.Stderr, "Error: Too many arguments: %v\n", filteredArgs[1:])
		}
		fmt.Fprintln(os.Stderr, "Usage: buzz view <goalslug> [--web] [--json] [--datapoints]")
		os.Exit(1)
	}

	goalSlug = filteredArgs[0]

	// Load config
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
		rawJSON, err := client.FetchGoalRawJSON(goalSlug, datapointsFlag)
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
	goal, err := client.FetchGoal(goalSlug)
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
