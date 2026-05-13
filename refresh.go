package main

import (
	"fmt"
	"os"
)

// handleRefreshCommand refreshes autodata for a goal
func handleRefreshCommand() {
	// Check arguments: buzz refresh <goalslug>
	if len(os.Args) != 3 {
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing required argument")
		} else {
			fmt.Printf("Error: Too many arguments: %v\n", os.Args[3:])
		}
		fmt.Println("Usage: buzz refresh <goalslug>")
		os.Exit(1)
	}

	goalSlug := os.Args[2]

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

	// Refresh the goal
	queued, err := client.RefreshGoal(goalSlug)
	if err != nil {
		fmt.Printf("Error: Failed to refresh goal: %s\n", redactError(err))
		os.Exit(1)
	}

	if queued {
		fmt.Printf("Successfully queued refresh for goal: %s\n", goalSlug)
	} else {
		fmt.Printf("Goal %s was not queued for refresh\n", goalSlug)
	}

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}
