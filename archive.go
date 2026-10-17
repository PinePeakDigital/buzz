package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// archiveRequest is a parsed, validated `buzz archive` invocation.
type archiveRequest struct {
	goalSlug    string
	skipConfirm bool
	// updateNotice, when set, is printed after a successful archive only — a
	// cancelled prompt must not nag about updates. The handler wires it to
	// getUpdateMessage; tests leave it nil to stay off the network.
	updateNotice func() string
}

// handleArchiveCommand schedules a goal for archive.
func handleArchiveCommand() {
	req, code, done := parseArchiveArgs(os.Args[2:], os.Stdout, os.Stderr)
	if done {
		os.Exit(code)
	}

	_, client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}

	req.updateNotice = getUpdateMessage
	os.Exit(runArchiveCommand(req, os.Stdin, client, os.Stdout, os.Stderr))
}

// parseArchiveArgs parses and validates `buzz archive` arguments, returning the
// request, a process exit code, and done=true when the caller should stop (help
// shown, or a parse/validation error). It touches no config or network, so
// --help and bad input are handled without authentication.
func parseArchiveArgs(args []string, stdout, stderr io.Writer) (archiveRequest, int, bool) {
	const usage = "Usage: buzz archive [-y|--yes] <goalslug>"
	archiveFlags := flag.NewFlagSet("archive", flag.ContinueOnError)
	// Silence the flag package's own output; we print our own usage.
	archiveFlags.SetOutput(io.Discard)
	yes := archiveFlags.Bool("yes", false, "Skip the confirmation prompt")
	yesShort := archiveFlags.Bool("y", false, "Skip the confirmation prompt (shorthand)")
	if err := archiveFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, usage)
			return archiveRequest{}, 0, true
		}
		fmt.Fprintf(stderr, "Error parsing flags: %s\n", redactError(err))
		fmt.Fprintln(stderr, usage)
		return archiveRequest{}, 2, true
	}

	rest := archiveFlags.Args()
	if len(rest) != 1 {
		if len(rest) < 1 {
			fmt.Fprintln(stderr, "Error: Missing required argument: goal slug")
		} else {
			fmt.Fprintf(stderr, "Error: Too many arguments: %v\n", rest[1:])
		}
		fmt.Fprintln(stderr, usage)
		return archiveRequest{}, 1, true
	}

	return archiveRequest{goalSlug: rest[0], skipConfirm: *yes || *yesShort}, 0, false
}

// runArchiveCommand schedules the archive, prompting for confirmation on stdin
// unless skipConfirm is set, and returns the process exit code.
func runArchiveCommand(req archiveRequest, stdin io.Reader, client Client, stdout, stderr io.Writer) int {
	if !req.skipConfirm {
		// Fetch the goal only to render the pledge in the prompt, so the --yes
		// path pays for one API call rather than two.
		currentGoal, err := client.FetchGoal(context.Background(), req.goalSlug)
		if err != nil {
			fmt.Fprintf(stderr, "Error: Failed to fetch goal: %s\n", redactError(err))
			return 1
		}
		pledge := strconv.FormatFloat(currentGoal.Pledge, 'f', -1, 64)
		if !confirm(stdin, stdout, fmt.Sprintf("Archive %s? $%s pledged. [y/N] ", req.goalSlug, pledge)) {
			return 0
		}
	}

	goal, err := client.ArchiveGoal(context.Background(), req.goalSlug)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to archive goal: %s\n", redactError(err))
		return 1
	}

	// A goal that has already won or lost comes back with a past archive date;
	// Beeminder archives it on its next callback sweep, so say that instead of
	// printing a date that has already gone by.
	if goal.ScheduledForArchive(time.Now()) {
		fmt.Fprintf(stdout, "Scheduled %s for archive on %s.\n", goal.Slug, time.Unix(goal.Archivedate, 0).Format("Mon Jan 2, 2006"))
	} else {
		fmt.Fprintf(stdout, "Archiving %s now.\n", goal.Slug)
	}
	if req.updateNotice != nil {
		fmt.Fprint(stdout, req.updateNotice())
	}
	return 0
}
