package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

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

// TestArchiveMarker checks the slug marker gates on a still-future archivedate:
// future -> prefixed, unset or past/now -> plain.
func TestArchiveMarker(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	cases := map[string]struct {
		archivedate int64
		want        string
	}{
		"future":  {now.Unix() + 86400, "(A) "},
		"unset":   {0, ""},
		"past":    {now.Unix() - 86400, ""},
		"exactly": {now.Unix(), ""}, // boundary: not strictly future
	}
	for name, c := range cases {
		if got := archiveMarker(Goal{Archivedate: c.archivedate}, now); got != c.want {
			t.Errorf("%s: archiveMarker = %q, want %q", name, got, c.want)
		}
	}
}

// TestTableArchiveMarkerAlignment guards that the slug marker doesn't knock the
// next column out of alignment: the marked and unmarked rows must line up.
func TestTableArchiveMarkerAlignment(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	tbl := Table{Columns: []Column{
		{Cell: func(g Goal) string { return archiveMarker(g, now) + g.Slug }},
		{Cell: func(g Goal) string { return g.Baremin }},
	}}
	goals := []Goal{
		{Slug: "abc", Archivedate: now.Unix() + 86400, Baremin: "+1"}, // marked
		{Slug: "xyz", Baremin: "+2"},                                  // plain
	}
	lines := strings.Split(strings.TrimRight(tbl.Render(goals), "\n"), "\n")
	if !strings.HasPrefix(lines[0], "(A) abc") {
		t.Errorf("marked row = %q, want (A) prefix", lines[0])
	}
	if strings.HasPrefix(lines[1], "(A)") {
		t.Errorf("unscheduled row = %q, want no marker", lines[1])
	}
	// Equal total width => the Baremin column lines up in both rows.
	if len(lines[0]) != len(lines[1]) {
		t.Errorf("rows misaligned: %q (w=%d) vs %q (w=%d)",
			lines[0], len(lines[0]), lines[1], len(lines[1]))
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

// TestTableRenderAs covers the --format json/csv paths: json carries the raw
// goal objects, csv reuses the column headers/cells, table is unchanged, and an
// unknown format errors. Empty goals must still produce valid output ([] / a
// header row) rather than "null".
func TestTableRenderAs(t *testing.T) {
	tbl := Table{
		Columns: []Column{
			{Header: "Slug", Cell: func(g Goal) string { return g.Slug }},
			{Header: "Baremin", Cell: func(g Goal) string { return g.Baremin }},
		},
	}
	goals := []Goal{{Slug: "run", Baremin: "+1"}, {Slug: "read", Baremin: "+2"}}

	// table == Render
	if got, err := tbl.RenderAs("table", goals); err != nil || got != tbl.Render(goals) {
		t.Errorf("RenderAs(table) = %q, %v; want == Render", got, err)
	}

	// json: valid array of the full goal objects (raw slug field present)
	jsonOut, err := tbl.RenderAs("json", goals)
	if err != nil {
		t.Fatalf("RenderAs(json) error: %v", err)
	}
	var decoded []Goal
	if err := json.Unmarshal([]byte(jsonOut), &decoded); err != nil {
		t.Fatalf("json output not valid: %v\n%s", err, jsonOut)
	}
	if len(decoded) != 2 || decoded[0].Slug != "run" {
		t.Errorf("json roundtrip mismatch: %+v", decoded)
	}

	// csv: header row from Column.Header, then one row per goal
	csvOut, err := tbl.RenderAs("csv", goals)
	if err != nil {
		t.Fatalf("RenderAs(csv) error: %v", err)
	}
	wantCSV := "Slug,Baremin\nrun,+1\nread,+2\n"
	if csvOut != wantCSV {
		t.Errorf("csv output = %q, want %q", csvOut, wantCSV)
	}

	// empty goals: json emits [] not null; csv emits just the header
	if got, _ := tbl.RenderAs("json", nil); got != "[]\n" {
		t.Errorf("RenderAs(json, nil) = %q, want %q", got, "[]\n")
	}
	if got, _ := tbl.RenderAs("csv", nil); got != "Slug,Baremin\n" {
		t.Errorf("RenderAs(csv, nil) = %q, want header only", got)
	}

	// unknown format is an error
	if _, err := tbl.RenderAs("yaml", goals); err == nil {
		t.Error("RenderAs(yaml) = nil error, want error")
	}
}
