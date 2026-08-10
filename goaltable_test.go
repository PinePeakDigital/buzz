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

// TestArchiveDot checks the slug dot gates on a still-future archivedate:
// future -> trailing " •", unset or past/now -> nothing.
func TestArchiveDot(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	plain := lipgloss.NewStyle()
	cases := map[string]struct {
		archivedate int64
		want        string
	}{
		"future":  {now.Unix() + 86400, " •"},
		"unset":   {0, ""},
		"past":    {now.Unix() - 86400, ""},
		"exactly": {now.Unix(), ""}, // boundary: not strictly future
	}
	for name, c := range cases {
		if got := archiveDot(Goal{Archivedate: c.archivedate}, now, plain); got != c.want {
			t.Errorf("%s: archiveDot = %q, want %q", name, got, c.want)
		}
	}
}

// TestTableArchiveDotAlignment guards that the trailing dot (multibyte, and in
// production coloured) doesn't knock the next column out of alignment: the
// marked and unmarked rows must line up. Byte-length padding would misalign it.
func TestTableArchiveDotAlignment(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	orange := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	tbl := Table{Columns: []Column{
		{Cell: func(g Goal) string { return g.Slug + archiveDot(g, now, orange) }},
		{Cell: func(g Goal) string { return g.Baremin }},
	}}
	goals := []Goal{
		{Slug: "abc", Archivedate: now.Unix() + 86400, Baremin: "+1"}, // marked
		{Slug: "xyz", Baremin: "+2"},                                  // plain
	}
	lines := strings.Split(strings.TrimRight(tbl.Render(goals), "\n"), "\n")
	if !strings.Contains(lines[0], "abc •") {
		t.Errorf("marked row = %q, want slug + dot", lines[0])
	}
	if strings.Contains(lines[1], "•") {
		t.Errorf("unscheduled row = %q, want no dot", lines[1])
	}
	// Equal display width => the Baremin column lines up in both rows, even
	// though the marked cell carries extra bytes (glyph + any colour codes).
	if lipgloss.Width(lines[0]) != lipgloss.Width(lines[1]) {
		t.Errorf("rows misaligned: %q (w=%d) vs %q (w=%d)",
			lines[0], lipgloss.Width(lines[0]), lines[1], lipgloss.Width(lines[1]))
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

// TestTableColorizeArchiveDot guards the filtered-view interaction: in a
// Colorize table (today/tomorrow/…), the plain archive dot must sit INSIDE the
// row's colour span — an empty-style dot that emitted its own reset would blank
// the urgency colour for the rest of the row — and must not break alignment.
func TestTableColorizeArchiveDot(t *testing.T) {
	orig := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(orig) })

	now := time.Unix(1_000_000_000, 0)
	tbl := Table{
		Colorize: true,
		Columns: []Column{
			{Cell: func(g Goal) string { return g.Slug + archiveDot(g, now, lipgloss.NewStyle()) }},
			{Cell: func(g Goal) string { return g.Baremin }},
		},
	}
	marked := Goal{Slug: "arch", Safebuf: 0, Archivedate: now.Unix() + 86400, Baremin: "+1"}
	plain := Goal{Slug: "keep", Safebuf: 0, Baremin: "+2"}
	out := tbl.Render([]Goal{marked, plain})

	markedLine, _, _ := strings.Cut(out, "\n")
	if !strings.Contains(markedLine, "•") {
		t.Fatalf("marked row missing dot: %q", markedLine)
	}
	if before, _, _ := strings.Cut(markedLine, "•"); strings.Contains(before, "\x1b[0m") {
		t.Errorf("colour reset before the dot; row colour would break mid-line: %q", markedLine)
	}
	plainLine, _, _ := strings.Cut(strings.SplitN(out, "\n", 2)[1], "\n")
	if lipgloss.Width(markedLine) != lipgloss.Width(plainLine) {
		t.Errorf("colourised rows misaligned: %q (w=%d) vs %q (w=%d)",
			markedLine, lipgloss.Width(markedLine), plainLine, lipgloss.Width(plainLine))
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
