package main

import (
	"fmt"
	"io"
)

// loadClient runs the shared credential preamble for the authenticated CLI
// commands: it confirms a config exists, loads it, and builds the API client.
// On failure it writes the standard message to stderr and returns ok=false (the
// caller should exit non-zero). It returns the loaded Config too, since a few
// commands (view, review) need it downstream; client-only callers discard it
// with _. Extracting it means the credentialed commands stop repeating the
// ConfigExists → LoadConfig → NewHTTPClient dance — and, by all going through
// this one stderr path, can't drift into printing the error to stdout.
//
// Distinct from loadConfigAndGoals (main.go), which *returns* wrapped errors
// (for next/schedule, which format their own output) rather than printing here.
func loadClient(stderr io.Writer) (*Config, Client, bool) {
	if !ConfigExists() {
		fmt.Fprintln(stderr, "Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		return nil, nil, false
	}
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to load config: %s\n", redactError(err))
		return nil, nil, false
	}
	return config, NewHTTPClient(config), true
}
