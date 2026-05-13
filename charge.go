package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// handleChargeCommand creates a charge for the authenticated user
func handleChargeCommand() {
	// Usage: buzz charge <amount> <note> [--dryrun]
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Error: Missing required arguments")
		fmt.Fprintln(os.Stderr, "Usage: buzz charge <amount> <note> [--dryrun]")
		os.Exit(1)
	}

	args := os.Args[2:]
	amountStr := args[0]
	// Collect note parts and allow --dryrun anywhere after amount
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
		fmt.Fprintln(os.Stderr, "Error: Note is required")
		fmt.Fprintln(os.Stderr, "Usage: buzz charge <amount> <note> [--dryrun]")
		os.Exit(1)
	}

	// Validate amount is a number
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Amount must be a valid number, got: %s\n", amountStr)
		os.Exit(1)
	}
	// ParseFloat accepts "NaN"/"+Inf"/"-Inf"; reject those explicitly before
	// the lower-bound check (NaN comparisons are always false, so NaN would
	// otherwise sneak past `amount < 1.00` and reach the API).
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		fmt.Fprintf(os.Stderr, "Error: Amount must be a finite number, got: %s\n", amountStr)
		os.Exit(1)
	}

	// Validate amount is >= 1.00
	if amount < 1.00 {
		fmt.Fprintf(os.Stderr, "Error: Amount must be at least 1.00, got: %.2f\n", amount)
		os.Exit(1)
	}

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

	// Create the charge (API returns the created/dry-run charge)
	ch, err := client.CreateCharge(amount, note, dryrun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create charge: %s\n", redactError(err))
		os.Exit(1)
	}

	if dryrun {
		fmt.Printf("Dry run: Would charge $%.2f with note: %q for %s\n", ch.Amount, ch.Note, ch.Username)
	} else {
		fmt.Printf("Successfully created charge %s: $%.2f with note: %q for %s\n", ch.ID, ch.Amount, ch.Note, ch.Username)
	}

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}
