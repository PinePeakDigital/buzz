package main

import (
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
// renderer calls it once per goal during measure and again during print, so it
// must be cheap and deterministic.
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
