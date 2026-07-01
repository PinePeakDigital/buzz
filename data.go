package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

// handleDataCommand lists a goal's datapoints.
func handleDataCommand() {
	client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}
	code := runDataCommand(os.Args[2:], client, os.Stdout, os.Stderr)
	if code == 0 {
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// runDataCommand is the testable core of `buzz data`. It expects a single
// <goalslug> argument and prints that goal's datapoints in chronological order,
// one per line as "date  value  comment".
func runDataCommand(args []string, client Client, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		if len(args) < 1 {
			fmt.Fprintln(stderr, "Error: Missing required argument")
		} else {
			fmt.Fprintf(stderr, "Error: Too many arguments: %v\n", args[1:])
		}
		fmt.Fprintln(stderr, "Usage: buzz data <goalslug>")
		return 1
	}
	goalSlug := args[0]

	goal, err := client.FetchGoalWithDatapoints(context.Background(), goalSlug)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", redactError(err))
		return 1
	}

	if len(goal.Datapoints) == 0 {
		fmt.Fprintf(stdout, "No datapoints found for goal: %s\n", goalSlug)
		return 0
	}

	// The API's datapoint order isn't guaranteed; sort ascending by timestamp
	// so the output is deterministic (oldest first, newest last).
	dps := append([]Datapoint(nil), goal.Datapoints...)
	sort.SliceStable(dps, func(i, j int) bool { return dps[i].Timestamp < dps[j].Timestamp })

	maxValueLen := 0
	values := make([]string, len(dps))
	for i, dp := range dps {
		values[i] = fmt.Sprintf("%.6g", dp.Value)
		if len(values[i]) > maxValueLen {
			maxValueLen = len(values[i])
		}
	}

	for i, dp := range dps {
		if dp.Comment != "" {
			fmt.Fprintf(stdout, "%s   %-*s   %s\n", datapointDate(dp), maxValueLen, values[i], dp.Comment)
		} else {
			fmt.Fprintf(stdout, "%s   %s\n", datapointDate(dp), values[i])
		}
	}
	return 0
}

// datapointDate renders a datapoint's day as YYYY-MM-DD, preferring the
// Beeminder daystamp (which avoids timezone drift) and falling back to the
// UTC timestamp when the daystamp is absent or malformed.
func datapointDate(dp Datapoint) string {
	if len(dp.Daystamp) == 8 {
		return dp.Daystamp[:4] + "-" + dp.Daystamp[4:6] + "-" + dp.Daystamp[6:8]
	}
	return time.Unix(dp.Timestamp, 0).UTC().Format("2006-01-02")
}
