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
	fmt.Println("  buzz auth list                    List authenticated accounts")
	fmt.Println("  buzz auth logout <username>       Remove an account's credentials")
	fmt.Println("  buzz auth help                    Show this help message")
	fmt.Println("")
	fmt.Println("Logging in as a new username adds it alongside your existing accounts;")
	fmt.Println("goals from every account are shown together. Where a goal slug exists on")
	fmt.Println("more than one account, name the account explicitly: buzz add alice/read 1")
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
	case "list":
		os.Exit(runAuthListCommand(os.Stdout, os.Stderr))
	case "logout":
		os.Exit(runAuthLogoutCommand(os.Args[3:], os.Stdout, os.Stderr))
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

// runAuthListCommand prints the configured accounts, primary first. It is the
// only way to see what `buzz auth login` has accumulated without opening
// ~/.buzzrc, and deliberately prints no tokens.
func runAuthListCommand(stdout, stderr io.Writer) int {
	if !ConfigExists() {
		fmt.Fprintln(stderr, "Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		return 1
	}
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to load config: %s\n", redactError(err))
		return 1
	}
	for i, c := range config.accountConfigs() {
		if i == 0 {
			fmt.Fprintf(stdout, "%s (primary)\n", c.Username)
			continue
		}
		fmt.Fprintln(stdout, c.Username)
	}
	return 0
}

// runAuthLogoutCommand removes one account's credentials by username. Removing
// the primary promotes the next account in its place; removing the last one
// leaves buzz unauthenticated.
func runAuthLogoutCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "Usage: buzz auth logout <username>")
		return 1
	}
	if !ConfigExists() {
		fmt.Fprintln(stderr, "Error: No configuration found.")
		return 1
	}
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to load config: %s\n", redactError(err))
		return 1
	}
	if !config.removeAccount(args[0]) {
		fmt.Fprintf(stderr, "Error: No such account: %s\n", args[0])
		return 1
	}
	if err := SaveConfig(config); err != nil {
		fmt.Fprintf(stderr, "Error: Failed to save config: %s\n", redactError(err))
		return 1
	}
	fmt.Fprintf(stdout, "Removed account: %s\n", args[0])
	return 0
}
