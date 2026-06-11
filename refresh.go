package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

// handleRefreshCommand refreshes autodata for a goal.
func handleRefreshCommand() {
	client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}
	code := runRefreshCommand(os.Args[2:], client, os.Stdout, os.Stderr)
	if code == 0 {
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// runRefreshCommand is the testable core of `buzz refresh`. It expects a single
// <goalslug> argument, queues a refresh, and returns the process exit code.
func runRefreshCommand(args []string, client Client, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		if len(args) < 1 {
			fmt.Fprintln(stderr, "Error: Missing required argument")
		} else {
			fmt.Fprintf(stderr, "Error: Too many arguments: %v\n", args[1:])
		}
		fmt.Fprintln(stderr, "Usage: buzz refresh <goalslug>")
		return 1
	}
	goalSlug := args[0]

	queued, err := client.RefreshGoal(context.Background(), goalSlug)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to refresh goal: %s\n", redactError(err))
		return 1
	}

	if queued {
		fmt.Fprintf(stdout, "Successfully queued refresh for goal: %s\n", goalSlug)
	} else {
		fmt.Fprintf(stdout, "Goal %s was not queued for refresh\n", goalSlug)
	}
	return 0
}
