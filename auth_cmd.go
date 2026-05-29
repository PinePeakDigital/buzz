package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// printAuthHelp prints usage for the `buzz auth` command group.
func printAuthHelp() {
	fmt.Println("buzz auth - Manage Beeminder authentication")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  buzz auth login                   Authenticate by pasting your API credentials")
	fmt.Println("  buzz auth help                    Show this help message")
}

// handleAuthCommand dispatches `buzz auth <subcommand>`.
func handleAuthCommand() {
	if len(os.Args) < 3 {
		printAuthHelp()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "login":
		handleAuthLoginCommand()
	case "help", "-h", "--help":
		printAuthHelp()
	default:
		fmt.Printf("Unknown auth subcommand: %s\n", os.Args[2])
		printAuthHelp()
		os.Exit(1)
	}
}

// handleAuthLoginCommand reads Beeminder credentials interactively from stdin
// and saves them. Reading from stdin (rather than command-line arguments) keeps
// the auth token out of shell history. It also works with piped input, so
// `buzz auth login < creds.json` is supported for scripting.
func handleAuthLoginCommand() {
	fmt.Println("Beeminder Authentication")
	fmt.Println("")
	fmt.Println("Paste your Beeminder API credentials in JSON format.")
	fmt.Println("Get them from: https://www.beeminder.com/api/v1/auth_token.json")
	fmt.Println("")
	fmt.Println(`Format: {"username":"your_username","auth_token":"your_token"}`)
	fmt.Print("> ")

	// When stdin is piped (not a terminal), read the whole stream so
	// pretty-printed/multiline JSON survives intact. For an interactive
	// terminal, read a single line so the prompt returns as soon as the user
	// pastes their one-line credentials and presses Enter (reading to EOF
	// would instead block until Ctrl+D).
	var input string
	if fi, statErr := os.Stdin.Stat(); statErr == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read credentials: %s\n", err)
			os.Exit(1)
		}
		input = string(b)
	} else {
		// ReadString returns io.EOF along with any data read before the stream
		// ended (e.g. piped input with no trailing newline). Only treat it as a
		// failure when nothing was read at all.
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			fmt.Fprintf(os.Stderr, "Error: failed to read credentials: %s\n", err)
			os.Exit(1)
		}
		input = line
	}

	if _, err := parseAndSaveCredentials(input); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("✓ Authentication successful! Credentials saved to ~/.buzzrc")
}
