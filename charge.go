package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// handleChargeCommand creates a charge for the authenticated user.
func handleChargeCommand() {
	client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}
	code := runChargeCommand(os.Args[2:], client, os.Stdout, os.Stderr)
	if code == 0 {
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// runChargeCommand is the testable core of `buzz charge <amount> <note>
// [--dryrun]`. It validates the amount and note, creates the charge, and
// returns the process exit code.
func runChargeCommand(args []string, client Client, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "Error: Missing required arguments")
		fmt.Fprintln(stderr, "Usage: buzz charge <amount> <note> [--dryrun]")
		return 1
	}

	amountStr := args[0]
	// Collect note parts and allow --dryrun anywhere after amount.
	dryrun := false
	var noteParts []string
	for _, a := range args[1:] {
		if a == "--dryrun" {
			dryrun = true
			continue
		}
		noteParts = append(noteParts, a)
	}
	note := strings.Join(noteParts, " ")
	if strings.TrimSpace(note) == "" {
		fmt.Fprintln(stderr, "Error: Note is required")
		fmt.Fprintln(stderr, "Usage: buzz charge <amount> <note> [--dryrun]")
		return 1
	}

	// Validate amount is a number.
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Amount must be a valid number, got: %s\n", amountStr)
		return 1
	}
	// ParseFloat accepts "NaN"/"+Inf"/"-Inf"; reject those explicitly before
	// the lower-bound check (NaN comparisons are always false, so NaN would
	// otherwise sneak past `amount < 1.00` and reach the API).
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		fmt.Fprintf(stderr, "Error: Amount must be a finite number, got: %s\n", amountStr)
		return 1
	}
	// Validate amount is >= 1.00.
	if amount < 1.00 {
		fmt.Fprintf(stderr, "Error: Amount must be at least 1.00, got: %.2f\n", amount)
		return 1
	}

	// Create the charge (API returns the created/dry-run charge).
	ch, err := client.CreateCharge(context.Background(), amount, note, dryrun)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to create charge: %s\n", redactError(err))
		return 1
	}

	if dryrun {
		fmt.Fprintf(stdout, "Dry run: Would charge $%.2f with note: %q for %s\n", ch.Amount, ch.Note, ch.Username)
	} else {
		fmt.Fprintf(stdout, "Successfully created charge %s: $%.2f with note: %q for %s\n", ch.ID, ch.Amount, ch.Note, ch.Username)
	}
	return 0
}
