package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const addUsage = `Usage: buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> <value> [comment]
       echo "<value>" | buzz add [--requestid=<id>] [--daystamp=<date>] <goalslug> [comment]

Note: Flags must come BEFORE positional arguments.
      Example: buzz add --daystamp=20240115 goalslug value comment
      The --daystamp flag accepts dates in YYYYMMDD format.`

// addRequest is a fully-parsed, validated `buzz add` invocation, ready to send.
type addRequest struct {
	goalSlug  string
	value     string // already converted to a decimal-hours string when a time
	comment   string
	daystamp  string // YYYYMMDD, or "" to use the current timestamp
	requestid string
}

// handleAddCommand adds a datapoint to a goal without opening the TUI.
func handleAddCommand() {
	req, code, done := parseAddArgs(os.Args[2:], readValueFromStdin, os.Stdout, os.Stderr)
	if done {
		os.Exit(code)
	}

	client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}

	code = runAddCommand(req, client, os.Stdout, os.Stderr)
	if code == 0 {
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// parseAddArgs parses and validates `buzz add` arguments, returning the resolved
// request, a process exit code, and done=true when the caller should stop (help
// shown, or a parse/validation error). readStdin is called lazily to read a
// piped value only once the positional args warrant it, so `--help` and bad
// input are reported without consuming stdin or requiring authentication.
func parseAddArgs(args []string, readStdin func() (string, error), stdout, stderr io.Writer) (addRequest, int, bool) {
	addFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	// Silence the flag package's own output; we print our own richer usage on
	// both --help and parse errors.
	addFlags.SetOutput(io.Discard)
	requestid := addFlags.String("requestid", "", "Request ID for idempotency")
	daystamp := addFlags.String("daystamp", "", "Date for the datapoint in YYYYMMDD format")
	if err := addFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, addUsage)
			return addRequest{}, 0, true
		}
		fmt.Fprintf(stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(stderr, addUsage)
		return addRequest{}, 1, true
	}

	positional := addFlags.Args()

	// Detect if known flags appear after positional arguments and warn the user.
	if misplacedFlag := detectMisplacedFlag(positional); misplacedFlag != "" {
		fmt.Fprintf(stderr, "Warning: Flag '%s' appears after positional arguments and will be treated as part of the comment.\n", misplacedFlag)
		fmt.Fprintf(stderr, "Flags must come BEFORE positional arguments to be recognized.\n")
		fmt.Fprintf(stderr, "Correct usage: buzz add [--requestid=ID] [--daystamp=DATE] goalslug value comment\n")
		fmt.Fprintln(stderr, "")
	}

	if len(positional) < 1 {
		fmt.Fprintln(stderr, "Error: Missing required arguments")
		fmt.Fprintln(stderr, addUsage)
		return addRequest{}, 1, true
	}

	goalSlug := positional[0]
	var value string
	var commentStartIndex int // index where the optional comment starts

	// Try to read a value from stdin first (for piped input).
	stdinValue, err := readStdin()
	if err == nil && stdinValue != "" {
		// Reject the ambiguous case where a value is piped AND a positional
		// value is supplied — silently taking stdin could submit a different
		// datapoint than the user intended for a write operation.
		if len(positional) >= 2 {
			fmt.Fprintln(stderr, "Error: Provide value either via stdin or as a positional argument, not both")
			fmt.Fprintln(stderr, addUsage)
			return addRequest{}, 1, true
		}
		value = stdinValue
		commentStartIndex = 1
	} else if len(positional) >= 2 {
		value = positional[1]
		commentStartIndex = 2
	} else {
		fmt.Fprintln(stderr, "Error: Missing required value argument")
		fmt.Fprintln(stderr, addUsage)
		return addRequest{}, 1, true
	}

	// Optional comment — default when not provided.
	comment := "Added via buzz"
	if len(positional) >= commentStartIndex+1 {
		comment = strings.Join(positional[commentStartIndex:], " ")
	}

	// Validate the daystamp format (YYYYMMDD) if provided.
	var daystampForAPI string
	if *daystamp != "" {
		if _, err := time.Parse("20060102", *daystamp); err != nil {
			fmt.Fprintf(stderr, "Error: Invalid date format for --daystamp: %s (expected YYYYMMDD)\n", *daystamp)
			return addRequest{}, 1, true
		}
		daystampForAPI = *daystamp
	}

	// Convert a time-format value (e.g. "1:30:00") to decimal hours.
	if isTimeFormat(value) {
		decimalValue, ok := timeToDecimalHours(value)
		if !ok {
			fmt.Fprintf(stderr, "Error: Invalid time format: %s\n", value)
			return addRequest{}, 1, true
		}
		value = fmt.Sprintf("%.6g", decimalValue)
	}

	// Validate the value is a number.
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		fmt.Fprintf(stderr, "Error: Value must be a valid number, got: %s\n", value)
		return addRequest{}, 1, true
	}

	return addRequest{
		goalSlug:  goalSlug,
		value:     value,
		comment:   comment,
		daystamp:  daystampForAPI,
		requestid: *requestid,
	}, 0, false
}

// runAddCommand submits the datapoint for an already-validated request and
// returns the process exit code.
func runAddCommand(req addRequest, client Client, stdout, stderr io.Writer) int {
	// Use the current time as timestamp (only used when daystamp is empty).
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	if _, err := client.CreateDatapointWithDaystamp(context.Background(), req.goalSlug, timestamp, req.daystamp, req.value, req.comment, req.requestid); err != nil {
		fmt.Fprintf(stderr, "Error: Failed to add datapoint: %s\n", redactError(err))
		return 1
	}

	successMsg := fmt.Sprintf("Successfully added datapoint to %s: value=%s, comment=\"%s\"", req.goalSlug, req.value, req.comment)
	if req.daystamp != "" {
		successMsg += fmt.Sprintf(", daystamp=%s", req.daystamp)
	}
	if req.requestid != "" {
		successMsg += fmt.Sprintf(", requestid=\"%s\"", req.requestid)
	}
	fmt.Fprintln(stdout, successMsg)

	// Signal any running TUI instances to refresh so they pick up the new
	// datapoint. Don't fail the command if flag creation fails.
	if err := createRefreshFlag(); err != nil {
		fmt.Fprintf(stderr, "Warning: Could not create refresh flag: %s\n", redactError(err))
	}
	return 0
}
