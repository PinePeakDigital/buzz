package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// printAddUsageAndExit prints the usage for buzz add command and exits with code 1
func printAddUsageAndExit(errorMsg string) {
	fmt.Println("Error: " + errorMsg)
	fmt.Println("Usage: buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> <value> [comment]")
	fmt.Println("       echo \"<value>\" | buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> [comment]")
	fmt.Println("")
	fmt.Println("Note: Flags must come BEFORE positional arguments.")
	fmt.Println("      Example: buzz add --daystamp=20240115 goalslug value comment")
	fmt.Println("      The --daystamp flag accepts dates in YYYYMMDD format.")
	os.Exit(1)
}

// handleAddCommand adds a datapoint to a goal without opening the TUI
func handleAddCommand() {
	// Parse flags for the add command
	addFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	requestid := addFlags.String("requestid", "", "Request ID for idempotency")
	daystamp := addFlags.String("daystamp", "", "Date for the datapoint in YYYYMMDD format")
	if err := addFlags.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println("Usage: buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> <value> [comment]")
			fmt.Println("       echo \"<value>\" | buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> [comment]")
			fmt.Println("")
			fmt.Println("Note: Flags must come BEFORE positional arguments.")
			fmt.Println("      The --daystamp flag accepts dates in YYYYMMDD format.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", redactError(err))
		printAddUsageAndExit("Invalid flags")
	}

	// Get remaining positional arguments after flag parsing
	args := addFlags.Args()

	// Detect if known flags appear after positional arguments and warn the user
	if misplacedFlag := detectMisplacedFlag(args); misplacedFlag != "" {
		fmt.Fprintf(os.Stderr, "Warning: Flag '%s' appears after positional arguments and will be treated as part of the comment.\n", misplacedFlag)
		fmt.Fprintf(os.Stderr, "Flags must come BEFORE positional arguments to be recognized.\n")
		fmt.Fprintf(os.Stderr, "Correct usage: buzz add [--requestid=ID] [--daystamp=DATE] goalslug value comment\n")
		fmt.Fprintln(os.Stderr, "")
	}

	// Check arguments: buzz add <goalslug> <value> [comment]
	// Value can also be piped via stdin: echo "123" | buzz add mygoal [comment]
	if len(args) < 1 {
		printAddUsageAndExit("Missing required arguments")
	}

	goalSlug := args[0]
	var value string
	var commentStartIndex int // Index where optional comment starts in args

	// Try to read value from stdin first (for piped input)
	stdinValue, err := readValueFromStdin()
	if err == nil && stdinValue != "" {
		// Reject the ambiguous case where a value is piped AND a positional
		// value is supplied — the previous behaviour silently took stdin and
		// reinterpreted the positional as part of the comment, which could
		// submit a different datapoint than the user intended for a write
		// operation like `buzz add`.
		if len(args) >= 2 {
			printAddUsageAndExit("Provide value either via stdin or as a positional argument, not both")
		}
		// Value provided via stdin
		value = stdinValue
		commentStartIndex = 1 // Comment starts at index 1 when value is piped
	} else if len(args) >= 2 {
		// Value provided as argument
		value = args[1]
		commentStartIndex = 2 // Comment starts at index 2
	} else {
		printAddUsageAndExit("Missing required value argument")
	}

	// Optional comment - default to "Added via buzz" if not provided
	comment := "Added via buzz"
	if len(args) >= commentStartIndex+1 {
		comment = strings.Join(args[commentStartIndex:], " ")
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

	// Parse and validate daystamp if provided
	var daystampForAPI string
	if *daystamp != "" {
		// Validate date format (YYYYMMDD)
		_, err := time.Parse("20060102", *daystamp)
		if err != nil {
			fmt.Printf("Error: Invalid date format for --daystamp: %s (expected YYYYMMDD)\n", *daystamp)
			os.Exit(1)
		}
		daystampForAPI = *daystamp
	}

	// Use current time as timestamp (only used if daystamp is not provided)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Convert time format to decimal hours if needed
	if isTimeFormat(value) {
		decimalValue, ok := timeToDecimalHours(value)
		if !ok {
			fmt.Printf("Error: Invalid time format: %s\n", value)
			os.Exit(1)
		}
		value = fmt.Sprintf("%.6g", decimalValue)
	}

	// Validate value is a number
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		fmt.Printf("Error: Value must be a valid number, got: %s\n", value)
		os.Exit(1)
	}

	// Create the datapoint
	_, err = client.CreateDatapointWithDaystamp(context.Background(), goalSlug, timestamp, daystampForAPI, value, comment, *requestid)
	if err != nil {
		fmt.Printf("Error: Failed to add datapoint: %s\n", redactError(err))
		os.Exit(1)
	}

	successMsg := fmt.Sprintf("Successfully added datapoint to %s: value=%s, comment=\"%s\"", goalSlug, value, comment)
	if *daystamp != "" {
		successMsg += fmt.Sprintf(", daystamp=%s", *daystamp)
	}
	if *requestid != "" {
		successMsg += fmt.Sprintf(", requestid=\"%s\"", *requestid)
	}
	fmt.Println(successMsg)

	// Signal any running TUI instances to refresh so they pick up the new
	// datapoint.
	if err := createRefreshFlag(); err != nil {
		// Don't fail the command if flag creation fails
		fmt.Fprintf(os.Stderr, "Warning: Could not create refresh flag: %s\n", redactError(err))
	}

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}
