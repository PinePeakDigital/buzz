package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

// goaltable renders a list of goals as a column-aligned text table. The
// pipeline every list command was reimplementing —
//
//	measure max width per column → pad cells → optionally colour by urgency → join with two-space separators
//
// — now lives here. Callers describe their columns declaratively and call
// Render; sorting and filtering stay outside the renderer where they belong.

// Column is one column in a goal table.
//
// Cell is a pure function from a Goal to its rendered string value. The
// renderer calls it once per goal to measure column widths, caches the result,
// and reuses it during print — so it must be deterministic but doesn't need to
// be especially cheap.
type Column struct {
	Header string
	Cell   func(Goal) string
}

// Table is a declarative description of a goal table.
//
//   - Columns gives the header text and per-cell formatter for each column.
//   - ShowHeader adds a header row and a hyphen-rule separator above the data
//     rows (matches `buzz list`); filtered list commands set it false.
//   - Colorize wraps each data row in the goal's urgency TextStyle so the
//     `today` / `tomorrow` / `due` / `less` / `all` views keep their colour
//     coding. The header row is never coloured.
type Table struct {
	Columns    []Column
	ShowHeader bool
	Colorize   bool
}

// Render produces the table as a single string with a trailing newline per
// row. Columns are separated by two spaces; the last column is not padded so
// trailing whitespace doesn't bleed past the value (matches the previous
// fmt.Printf("...%s\n") pattern in every list command).
func (t Table) Render(goals []Goal) string {
	if len(t.Columns) == 0 {
		return ""
	}

	// Precompute every cell so we can measure once and print without
	// re-calling the Cell functions during render.
	cells := make([][]string, len(goals))
	widths := make([]int, len(t.Columns))
	if t.ShowHeader {
		// Seed widths from header lengths so the header row never gets clipped.
		for i, c := range t.Columns {
			widths[i] = len(c.Header)
		}
	}
	for i, g := range goals {
		row := make([]string, len(t.Columns))
		for j, c := range t.Columns {
			row[j] = c.Cell(g)
			if len(row[j]) > widths[j] {
				widths[j] = len(row[j])
			}
		}
		cells[i] = row
	}

	var b strings.Builder

	if t.ShowHeader {
		headers := make([]string, len(t.Columns))
		for i, c := range t.Columns {
			headers[i] = c.Header
		}
		b.WriteString(padRow(headers, widths))
		b.WriteString("\n")
		rule := make([]string, len(t.Columns))
		for i, w := range widths {
			rule[i] = strings.Repeat("-", w)
		}
		b.WriteString(padRow(rule, widths))
		b.WriteString("\n")
	}

	for i, row := range cells {
		line := padRow(row, widths)
		if t.Colorize {
			line = UrgencyFor(goals[i].Safebuf).TextStyle().Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// RenderAs renders the goals in the requested output format for the global
// --format flag. "table" (and "") give the human-readable column table; "json"
// emits the raw goal objects (every API field) for scripting; "csv" emits the
// table's own columns as comma-separated rows with a header.
//
// json intentionally carries the full objects rather than the displayed
// columns, so a projected/bumped column value (e.g. the `tomorrow` view's
// baremin) surfaces only in table and csv output — csv reuses the Cell
// functions, json reflects the underlying goal.
func (t Table) RenderAs(format string, goals []Goal) (string, error) {
	switch format {
	case "", "table":
		return t.Render(goals), nil
	case "json":
		if goals == nil {
			goals = []Goal{} // marshal an empty list as [] rather than null
		}
		b, err := json.MarshalIndent(goals, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b) + "\n", nil
	case "csv":
		// ponytail: cells (baremin "+1", free-text comments) aren't sanitized for
		// spreadsheet formula injection — it's the user's own data on their own
		// machine, so they'd only be attacking themselves. Add ^[=+\-@] quoting if
		// csv ever carries another account's data.
		headers := make([]string, len(t.Columns))
		for i, c := range t.Columns {
			headers[i] = c.Header
		}
		rows := make([][]string, len(goals))
		for i, g := range goals {
			row := make([]string, len(t.Columns))
			for j, c := range t.Columns {
				row[j] = c.Cell(g)
			}
			rows[i] = row
		}
		return encodeCSV(headers, rows)
	default:
		return "", fmt.Errorf("unknown format %q (want table, json, or csv)", format)
	}
}

// encodeCSV renders a header row followed by data rows as a CSV string. The
// csv.Writer buffers and latches the first write error, surfaced by w.Error()
// after Flush — so one final check replaces per-row error handling. Shared by
// the goal, datapoint, and `next` csv output paths.
func encodeCSV(headers []string, rows [][]string) (string, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write(headers)
	for _, r := range rows {
		w.Write(r)
	}
	w.Flush()
	return buf.String(), w.Error()
}

// padRow joins cells with two-space separators, left-padding every column
// except the last to its measured width.
func padRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		if i == len(cells)-1 {
			parts[i] = cell
		} else {
			parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
		}
	}
	return strings.Join(parts, "  ")
}
