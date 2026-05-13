package main

import (
	"context"
	"fmt"
	"os"
)

// handleListCommand outputs a summary list of all goals with slug, title, rate, and stakes
func handleListCommand() {
	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz' first to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)

	// Fetch goals
	goals, err := client.FetchGoals(context.Background())
	if err != nil {
		fmt.Printf("Error: Failed to fetch goals: %s\n", redactError(err))
		os.Exit(1)
	}

	// Sort goals alphabetically by slug for easy scanning
	SortGoalsBySlug(goals)

	// If no goals, exit
	if len(goals) == 0 {
		fmt.Println("No goals found.")
		return
	}

	// Print summary header
	fmt.Printf("Total goals: %d\n\n", len(goals))

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
	fmt.Print(table.Render(goals))

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
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
