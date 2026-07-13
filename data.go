package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// handleDataCommand lists a goal's datapoints.
func handleDataCommand() {
	client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}
	code := runDataCommand(os.Args[2:], client, outputFormat, os.Stdout, os.Stderr)
	if code == 0 {
		fmt.Print(getUpdateMessage())
	}
	os.Exit(code)
}

// runDataCommand is the testable core of `buzz data`. It expects a single
// <goalslug> argument and prints that goal's datapoints one per line as
// "date  value  comment", oldest-first by default or newest-first with --desc.
func runDataCommand(args []string, client Client, format string, stdout, stderr io.Writer) int {
	const usage = "Usage: buzz data [--asc|--desc] <goalslug>"

	// Parse flags on either side of the positional slug (as `view` does), so
	// `buzz data --desc g` and `buzz data g --desc` both work.
	dataFlags := flag.NewFlagSet("data", flag.ContinueOnError)
	dataFlags.SetOutput(stderr)
	dataFlags.Usage = func() {} // we print our own usage on error
	asc := dataFlags.Bool("asc", false, "Sort datapoints oldest-first (default)")
	desc := dataFlags.Bool("desc", false, "Sort datapoints newest-first")

	var positional []string
	remaining := args
	for len(remaining) > 0 {
		if err := dataFlags.Parse(remaining); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				fmt.Fprintln(stdout, usage)
				return 0
			}
			fmt.Fprintf(stderr, "Error parsing flags: %s\n", redactError(err))
			fmt.Fprintln(stderr, usage)
			return 2
		}
		rest := dataFlags.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		remaining = rest[1:]
	}

	if *asc && *desc {
		fmt.Fprintln(stderr, "Error: --asc and --desc are mutually exclusive")
		fmt.Fprintln(stderr, usage)
		return 1
	}

	if len(positional) != 1 {
		if len(positional) == 0 {
			fmt.Fprintln(stderr, "Error: Missing required argument")
		} else {
			fmt.Fprintf(stderr, "Error: Too many arguments: %v\n", positional[1:])
		}
		fmt.Fprintln(stderr, usage)
		return 1
	}
	goalSlug := positional[0]

	goal, err := client.FetchGoalWithDatapoints(context.Background(), goalSlug)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", redactError(err))
		return 1
	}

	// The API's datapoint order isn't guaranteed; sort by timestamp so the
	// output is deterministic. Default oldest-first; --desc flips to newest-first.
	dps := append([]Datapoint(nil), goal.Datapoints...)
	sort.SliceStable(dps, func(i, j int) bool {
		if *desc {
			return dps[i].Timestamp > dps[j].Timestamp
		}
		return dps[i].Timestamp < dps[j].Timestamp
	})

	// Machine-readable formats emit valid output even when empty ([] / header
	// row), so they run before the human "No datapoints" short-circuit.
	if format != "table" {
		rendered, err := renderDatapointsAs(format, dps)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %s\n", redactError(err))
			return 1
		}
		fmt.Fprint(stdout, rendered)
		return 0
	}

	if len(dps) == 0 {
		fmt.Fprintf(stdout, "No datapoints found for goal: %s\n", goalSlug)
		return 0
	}

	dates, values, maxValueLen := formatDatapointRows(dps)
	for i, dp := range dps {
		if dp.Comment != "" {
			fmt.Fprintf(stdout, "%s   %-*s   %s\n", dates[i], maxValueLen, values[i], dp.Comment)
		} else {
			fmt.Fprintf(stdout, "%s   %s\n", dates[i], values[i])
		}
	}
	return 0
}

// renderDatapointsAs renders datapoints as json (the raw datapoint objects) or
// csv (date, value, comment — matching the human table's columns). The date and
// value formatting mirror the text output so all three formats agree.
func renderDatapointsAs(format string, dps []Datapoint) (string, error) {
	switch format {
	case "json":
		if dps == nil {
			dps = []Datapoint{} // marshal an empty list as [] rather than null
		}
		b, err := json.MarshalIndent(dps, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b) + "\n", nil
	case "csv":
		var buf strings.Builder
		w := csv.NewWriter(&buf)
		if err := w.Write([]string{"date", "value", "comment"}); err != nil {
			return "", err
		}
		for _, dp := range dps {
			row := []string{datapointDate(dp), fmt.Sprintf("%.6g", dp.Value), dp.Comment}
			if err := w.Write(row); err != nil {
				return "", err
			}
		}
		w.Flush()
		return buf.String(), w.Error()
	default:
		return "", fmt.Errorf("unknown format %q (want table, json, or csv)", format)
	}
}

// formatDatapointRows renders per-datapoint date and value strings (parallel to
// dps) and the width of the widest value, so callers can print aligned
// "date   value   comment" rows. Shared by `buzz data` and the review view's
// recent-datapoints block, keeping the date and value formatting identical.
func formatDatapointRows(dps []Datapoint) (dates, values []string, maxValueLen int) {
	dates = make([]string, len(dps))
	values = make([]string, len(dps))
	for i, dp := range dps {
		dates[i] = datapointDate(dp)
		values[i] = fmt.Sprintf("%.6g", dp.Value)
		if len(values[i]) > maxValueLen {
			maxValueLen = len(values[i])
		}
	}
	return dates, values, maxValueLen
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
