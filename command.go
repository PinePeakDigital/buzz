package main

import (
	"fmt"
	"io"
)

// loadClient runs the shared credential preamble for the authenticated CLI
// commands: it confirms a config exists, loads it, and builds the API client.
// On failure it writes the standard message to stderr and returns ok=false (the
// caller should exit non-zero). Extracting it means the credentialed commands
// stop repeating the ConfigExists → LoadConfig → NewHTTPClient dance, and their
// real logic moves into testable run* cores that take a Client.
func loadClient(stderr io.Writer) (Client, bool) {
	if !ConfigExists() {
		fmt.Fprintln(stderr, "Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		return nil, false
	}
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to load config: %s\n", redactError(err))
		return nil, false
	}
	return NewHTTPClient(config), true
}
