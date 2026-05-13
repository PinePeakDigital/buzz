package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestTableRenderEmpty exercises the degenerate cases: no columns and no
// goals. Both should produce an empty string (no header row when there are
// no columns; no rows when there are no goals).
func TestTableRenderEmpty(t *testing.T) {
	if got := (Table{}).Render(nil); got != "" {
		t.Errorf("empty table render = %q, want empty", got)
	}
	tbl := Table{
		Columns:    []Column{{Header: "Slug", Cell: func(g Goal) string { return g.Slug }}},
		ShowHeader: false,
	}
	if got := tbl.Render(nil); got != "" {
		t.Errorf("table with columns but no goals = %q, want empty", got)
	}
}

// TestTableRenderHeader checks the `buzz list`-style table: header row, rule
// row, two-space column separators, last column unpadded.
func TestTableRenderHeader(t *testing.T) {
	tbl := Table{
		ShowHeader: true,
		Columns: []Column{
			{Header: "Slug", Cell: func(g Goal) string { return g.Slug }},
			{Header: "Title", Cell: func(g Goal) string { return g.Title }},
			{Header: "Stakes", Cell: func(g Goal) string { return g.Slug + "$" }},
		},
	}
	goals := []Goal{
		{Slug: "abc", Title: "Short"},
		{Slug: "longslug", Title: "A longer title"},
	}
	got := tbl.Render(goals)

	// "Slug" widens to 8 (longslug), "Title" widens to 14, last column not padded.
	wantHeader := "Slug      Title           Stakes\n"
	wantRule := "--------  --------------  ---------\n"
	wantRow1 := "abc       Short           abc$\n"
	wantRow2 := "longslug  A longer title  longslug$\n"

	expected := wantHeader + wantRule + wantRow1 + wantRow2
	if got != expected {
		t.Errorf("table render mismatch\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

// TestTableRenderNoHeader checks the filtered-command style: no header rows,
// last column unpadded, every data row colourized when Colorize is set.
func TestTableRenderNoHeader(t *testing.T) {
	tbl := Table{
		Columns: []Column{
			{Cell: func(g Goal) string { return g.Slug }},
			{Cell: func(g Goal) string { return g.Baremin }},
		},
	}
	goals := []Goal{
		{Slug: "short", Baremin: "+1"},
		{Slug: "wider-slug", Baremin: "+0.5"},
	}
	got := tbl.Render(goals)
	// Slug column widens to 10, baremin column not padded (last column).
	want := "short       +1\nwider-slug  +0.5\n"
	if got != want {
		t.Errorf("no-header render mismatch\ngot:\n%q\nwant:\n%q", got, want)
	}
}

// TestTableRenderColorize asserts every data row gets the urgency style
// wrapper. lipgloss strips ANSI when the test process isn't a TTY, so we
// force TrueColor for the duration of this test and assert the colour
// prefixes differ between two rows with different urgencies.
func TestTableRenderColorize(t *testing.T) {
	orig := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(orig) })

	tbl := Table{
		Colorize: true,
		Columns: []Column{
			{Cell: func(g Goal) string { return g.Slug }},
		},
	}
	overdue := Goal{Slug: "overdue", Safebuf: 0}
	distant := Goal{Slug: "distant", Safebuf: 100}
	out := tbl.Render([]Goal{overdue, distant})

	if !strings.Contains(out, "overdue") || !strings.Contains(out, "distant") {
		t.Fatalf("expected both slugs in output, got %q", out)
	}
	overdueLine, _, _ := strings.Cut(out, "\n")
	distantLine, _, _ := strings.Cut(strings.SplitN(out, "\n", 2)[1], "\n")

	// The lines should differ because they wrap different colours; if
	// colourisation were a no-op they'd be byte-equal aside from the slug.
	if strings.ReplaceAll(overdueLine, "overdue", "") == strings.ReplaceAll(distantLine, "distant", "") {
		t.Errorf("colourised rows for different urgencies are identical; expected different ANSI wrappers\noverdue: %q\ndistant: %q",
			overdueLine, distantLine)
	}
}
