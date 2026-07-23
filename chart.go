package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
)

// ansiPattern strips SGR colour codes so we can measure asciigraph's output in
// visible columns when aligning the x-axis beneath it.
var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

// chart dimensions
const (
	chartHeight   = 10
	minChartWidth = 40
	maxChartWidth = 80
)

// renderGoalChart renders an ASCII chart of a goal's progress: the goal's
// datapoints (blue) against its bright red line (red), over the goal's graph
// window — the user-set tmin/tmax axis limits where present, otherwise the
// goal's full history (initday..now). See chartTimeframe and defaultTimeframe
// for the exact window resolution. It returns "" when there is nothing
// chartable (no datapoints, or none inside the window).
func renderGoalChart(goal Goal, width int) string {
	if len(goal.Datapoints) == 0 {
		return ""
	}

	startTime, endTime := chartTimeframe(goal, time.Now())

	processed := processDatapoints(goal, startTime, endTime)
	if len(processed) == 0 {
		return ""
	}

	// Parse the bright red line once. A malformed roadall is surfaced loudly
	// (it almost certainly signals a parser or upstream bug); an absent one is
	// reported as "not populated" rather than silently drawn as a flat line at
	// zero. See docs/adr/0003-bright-red-line-parsing-failure-policy.md.
	brightLine, err := parseRoad(goal.Roadall, goal.Runits)
	if err != nil {
		return "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(0, 2).
			Render(fmt.Sprintf("⚠ Couldn't render the bright red line: %s", err)) + "\n"
	}
	if len(brightLine) == 0 {
		return "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 2).
			Render("The bright red line wasn't populated for this goal.") + "\n"
	}
	// Snap road knots onto the same local-midnight day grid the datapoints are
	// bucketed on, mirroring beebrain (stampIn dayparses road rows and data to
	// one day grid). Beeminder stores knot times mid-day (deadline-aligned), so
	// without this a road step and a same-day datapoint — e.g. a derailment's
	// road jump and its #DERAIL datapoint — draw risers several columns apart
	// instead of overlapping as they do on Beeminder's own graph.
	brightLine = daysnapRoad(brightLine, startTime.Location())

	chartWidth := width - 10 // leave room for padding and axis labels
	if chartWidth < minChartWidth {
		chartWidth = minChartWidth
	}
	if chartWidth > maxChartWidth {
		chartWidth = maxChartWidth
	}

	roadValues := roadValuesForTimeframe(brightLine, startTime, endTime, chartWidth)
	datapointValues, nodeCols := datapointSeries(processed, startTime, endTime, chartWidth)

	var chart strings.Builder
	chart.WriteString("\n")

	chartStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Padding(0, 2)

	goalType := "Do More"
	if goal.Yaw == -1 {
		goalType = "Do Less"
	}
	cumulativeStr := ""
	if goal.Kyoom {
		cumulativeStr = " (Cumulative)"
	}

	header := fmt.Sprintf("Goal Progress Chart - %s%s", goalType, cumulativeStr)
	chart.WriteString(chartStyle.Render(header) + "\n")

	timeframeInfo := fmt.Sprintf("Timeframe: %s to %s", startTime.Format("Jan 2"), endTime.Format("Jan 2, 2006"))
	chart.WriteString(chartStyle.Render(timeframeInfo) + "\n\n")

	// Plot the road first and the datapoints second: asciigraph lets a later
	// series overwrite an earlier one in shared cells, so this keeps the
	// datapoints (blue) drawn on top of the road (red) wherever they coincide.
	// The caption is rendered ourselves (below) so the date axis can sit
	// directly under the plot, above it.
	graphOutput := asciigraph.PlotMany([][]float64{roadValues, datapointValues},
		asciigraph.Height(chartHeight),
		asciigraph.Width(chartWidth),
		asciigraph.SeriesColors(asciigraph.Red, asciigraph.Blue),
	)

	// Indent the plot and date axis by 2 to match the padding the header,
	// caption, and review details use, so the chart isn't left-shifted from
	// the rest of the review UI. The gutter is measured on the un-indented
	// output, then plot and axis are shifted together.
	gutter := plotGutterWidth(graphOutput)
	// On sparse charts — where the datapoints are far enough apart to read as
	// separate steps — dot each datapoint on the blue line, mirroring Beeminder's
	// graph. The dots make it obvious the data lands on the bright red line at
	// each step, dispelling the illusion that the flat treads (which dip below the
	// rising line between points) mean the goal is off track. On dense charts the
	// nodes fill nearly every column and the dots would just smear the line into
	// noise, so they're skipped — and dense charts don't have the illusion anyway,
	// since the two lines merge.
	graphOutput = overlayDatapointMarkers(graphOutput, nodeCols, datapointValues, roadValues, gutter, chartWidth)
	chart.WriteString(indentLines(graphOutput, 2))
	chart.WriteString("\n")

	// Date axis aligned to the plot columns (asciigraph has no native x-axis).
	if axis := renderXAxis(startTime, endTime, gutter, chartWidth); axis != "" {
		chart.WriteString(indentLines(axis, 2) + "\n")
	}

	captionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 2)
	chart.WriteString(captionStyle.Render("Blue: datapoints, Red: bright red line") + "\n")

	return chart.String()
}

// chartTimeframe resolves the [start, end] window to chart from the goal's
// tmin/tmax (the graph axis limits the user set, parsed in the user's local
// zone), each falling back to defaultTimeframe independently when absent or
// unparseable.
//
// tmin and tmax are resolved separately rather than all-or-nothing because
// Beeminder force-nulls tmax once it falls into the past (gen_graph/writer.rb
// drops it; the goal model nils it on save), so tmax is null for virtually
// every goal — while tmin is commonly set. Gating on both would mean a user's
// explicit tmin was ignored on every goal, collapsing every chart onto the
// default window.
func chartTimeframe(goal Goal, now time.Time) (start, end time.Time) {
	defStart, defEnd := defaultTimeframe(goal, now)

	start = defStart
	if t, err := time.ParseInLocation("2006-01-02", goal.Tmin, time.Local); err == nil {
		start = t
	}

	end = defEnd
	if t, err := time.ParseInLocation("2006-01-02", goal.Tmax, time.Local); err == nil {
		// Extend to the last second of the Tmax calendar day. Build it as the
		// start of the next day minus one second (not +24h) so DST transitions —
		// where a local day is 23h or 25h — don't spill into the next day or clip
		// late ones.
		end = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location()).Add(-time.Second)
	}
	return start, end
}

// defaultTimeframe is the window charted when the goal carries no usable
// tmin/tmax. Beeminder leaves both null unless the user has set custom graph
// axis limits, so in practice this is the window almost every goal uses.
//
// The default start is the goal's own start (initday) — the date the bright red
// line begins — so the whole goal is charted, matching Beeminder's own default
// of showing all of a goal's data. The default end is now.
//
// When initday is unavailable, it falls back to the last 30 days, widened back
// to the most recent datapoint if that predates the window — otherwise a goal
// not updated within 30 days would have every datapoint fall outside the window
// and render no chart at all (graphs would appear only for recently-touched
// goals, seemingly at random).
func defaultTimeframe(goal Goal, now time.Time) (start, end time.Time) {
	end = now

	if goal.Initday > 0 {
		// initday marks a calendar day, so floor it to the start of that local
		// day. Using the raw instant (which Beeminder aligns to the goal's
		// deadline, often midday) would exclude a same-day datapoint logged
		// earlier in the day — e.g. a brand-new goal's only point.
		d := time.Unix(goal.Initday, 0).In(time.Local)
		start = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
	} else {
		start = now.AddDate(0, 0, -30)
		if last, ok := lastDatapointTime(goal); ok && last.Before(start) {
			start = last
		}
	}

	// A future-dated most-recent datapoint would otherwise sit past the window;
	// widen the end so it still shows.
	if last, ok := lastDatapointTime(goal); ok && last.After(end) {
		end = last
	}
	return start, end
}

// lastDatapointTime returns the timestamp of the goal's most recent datapoint.
// ok is false when the goal has no datapoints.
func lastDatapointTime(goal Goal) (t time.Time, ok bool) {
	if len(goal.Datapoints) == 0 {
		return time.Time{}, false
	}
	latest := goal.Datapoints[0].Timestamp
	for _, dp := range goal.Datapoints[1:] {
		if dp.Timestamp > latest {
			latest = dp.Timestamp
		}
	}
	// Return in the local zone: chartTimeframe resolves every other bound in
	// local time, and when this value becomes the window start/end it drives the
	// timeframe header and x-axis labels — a UTC instant could render the wrong
	// calendar day near midnight.
	return time.Unix(latest, 0).In(time.Local), true
}

// timedValue is a datapoint reduced to the two things the chart cares about:
// when it landed and the value to plot (the raw value, or the running total
// for cumulative goals).
type timedValue struct {
	timestamp int64
	value     float64
}

// processDatapoints reduces a goal's datapoints to the series to plot within
// [startTime, endTime], sorted by time.
//
// Datapoints are first aggregated per calendar day using the goal's aggday
// method (see aggregateDay), producing one value per day positioned at that
// day's boundary — matching Beeminder, which plots one aggregated point per day
// rather than one per datapoint. This is why two datapoints on the same day
// share a column (e.g. a same-day "0 then 1" reads as a single riser at the
// start of the day, not a within-day ramp).
//
// For cumulative (kyoom) goals the plotted value is then the running total of
// the daily aggregates, accumulated across ALL days (including those before the
// window); a synthetic anchor is prepended at startTime carrying the total
// reached just before the window, so the in-window line begins where the goal
// actually stood rather than at zero. It returns nil when no day falls inside
// the window (so a pure carry-over never draws a dataless line).
//
// For non-cumulative goals each in-window day's aggregate is plotted directly.
func processDatapoints(goal Goal, startTime, endTime time.Time) []timedValue {
	loc := startTime.Location()

	// Drop datapoints after the window end (including future-dated ones) before
	// bucketing. This matches Beeminder — which filters data to "now" (asof)
	// before aggregating — and stops a day's aggregate from absorbing same-day
	// points logged after endTime when the window ends mid-day.
	endUnix := endTime.Unix()
	inRange := make([]Datapoint, 0, len(goal.Datapoints))
	for _, dp := range goal.Datapoints {
		if dp.Timestamp <= endUnix {
			inRange = append(inRange, dp)
		}
	}

	// Bucket + reduce per day in one call (aggday module). What's left here is
	// purely the charting layer: window filtering and the kyoom running total.
	days := aggregateByDay(goal, inRange, loc)
	if len(days) == 0 {
		return nil
	}

	// Days are compared against the start of startTime's calendar day, not the
	// startTime instant itself: when the window begins mid-day (e.g. a stale
	// goal whose window starts at its last datapoint's timestamp), that day's
	// midnight-anchored aggregate would otherwise fall just before the window
	// and be dropped.
	startDay := startOfDay(startTime, loc)

	var processed []timedValue
	running := 0.0 // cumulative total of daily aggregates (kyoom only)
	carry := 0.0   // running total reached just before the window (kyoom only)
	inWindow := false

	for _, d := range days {
		if d.day.After(endTime) {
			continue // future day: not plotted, and doesn't affect the in-window line
		}
		ad := d.value
		switch {
		case goal.Kyoom:
			running += ad
			if d.day.Before(startDay) {
				carry = running
			} else {
				processed = append(processed, timedValue{timestamp: d.day.Unix(), value: running})
				inWindow = true
			}
		case !d.day.Before(startDay):
			processed = append(processed, timedValue{timestamp: d.day.Unix(), value: ad})
			inWindow = true
		}
	}

	if !inWindow {
		return nil
	}
	if goal.Kyoom {
		// Anchor at the start of the window's day (not the raw startTime instant),
		// so it sorts at-or-before every day point — which sit at local midnight.
		// A mid-day startTime would otherwise place the anchor after the first
		// day point, breaking datapointSeries' ascending-order assumption.
		processed = append([]timedValue{{timestamp: startDay.Unix(), value: carry}}, processed...)
	}
	return processed
}

// datapointSeries maps processed datapoints onto chartWidth evenly-spaced
// columns and fills the gaps: each datapoint lands in the column matching its
// position in the timeframe, and columns before the first / after the last hold
// that endpoint's value.
//
// Interior gaps are filled with a step-after staircase: each datapoint's value
// is held across the gap until the next datapoint's column, where the line jumps.
// This matches Beeminder's "steppy" line, the default connecting line for nearly
// every goal type — see beebrain's bgraph.js (the line is hold-then-jump, and its
// only alternative, `nosteppy`, is hardcoded off) and the per-type `steppy =>
// true` defaults in beeminder's goal_type.rb. It is NOT a per-goal-type choice:
// Beeminder never draws a diagonal connect-the-dots data line. (Cumulative goals
// carry a running total, so their staircase climbs; non-cumulative goals hold a
// raw value flat between points — both step.)
//
// The second return value is the distinct columns carrying an actual datapoint
// (the staircase's nodes, ascending) — as opposed to the gap-fill columns
// between them. overlayDatapointMarkers dots these nodes on sparse charts.
func datapointSeries(processed []timedValue, startTime, endTime time.Time, chartWidth int) ([]float64, []int) {
	values := make([]float64, chartWidth)
	hasDatapoint := make([]bool, chartWidth)
	duration := endTime.Sub(startTime).Seconds()

	for _, dp := range processed {
		col := 0
		if duration > 0 {
			progress := time.Unix(dp.timestamp, 0).Sub(startTime).Seconds() / duration
			col = int(progress * float64(chartWidth-1))
		}
		if col < 0 {
			col = 0
		}
		if col >= chartWidth {
			col = chartWidth - 1
		}
		// processed is time-sorted, so a later datapoint correctly overwrites
		// an earlier one sharing a column.
		values[col] = dp.value
		hasDatapoint[col] = true
	}

	firstDP, lastDP := -1, -1
	for i := 0; i < chartWidth; i++ {
		if hasDatapoint[i] {
			if firstDP == -1 {
				firstDP = i
			}
			lastDP = i
		}
	}
	if firstDP < 0 {
		return values, nil
	}

	for i := 0; i < firstDP; i++ {
		values[i] = values[firstDP]
	}
	for i := lastDP + 1; i < chartWidth; i++ {
		values[i] = values[lastDP]
	}

	prevDP := firstDP
	for i := firstDP + 1; i <= lastDP; i++ {
		if !hasDatapoint[i] {
			continue
		}
		// Hold the previous datapoint's value across the gap; the jump to this
		// datapoint's value lands at column i.
		for j := prevDP + 1; j < i; j++ {
			values[j] = values[prevDP]
		}
		prevDP = i
	}

	nodes := make([]int, 0)
	for i := 0; i < chartWidth; i++ {
		if hasDatapoint[i] {
			nodes = append(nodes, i)
		}
	}
	return values, nodes
}

// markerGlyph is the dot drawn on each datapoint node of a sparse chart,
// mirroring the dots on Beeminder's own graph.
const markerGlyph = '●'

// overlayDatapointMarkers dots each datapoint node on the blue line, but only
// when the chart is sparse enough that the nodes read as separate steps —
// roughly half the columns or fewer carrying a datapoint. It returns graph
// unchanged when the chart is too dense (the dots would just smear the merged
// line), when there's nothing to mark, or when the gutter couldn't be located.
//
// A marker replaces the blue glyph asciigraph already drew at the node's cell:
// asciigraph plots series[x]'s value at column x via the same value→row mapping
// recomputed here, so the target cell is already blue and swapping only its rune
// keeps the colour. The mapping mirrors asciigraph.PlotMany — min/max across
// both series, chartHeight rows, its sign-aware rounding; TestGoalChartMarkers
// guards it against drift if the library changes.
//
// asciigraph draws each segment's endpoints in the segment's left column, which
// steers where a marker sits:
//
//   - Stepped node (its value differs from the previous column's): the riser and
//     its top corner are drawn one column to the left, so the marker goes there —
//     on the corner — and the vertical riser leads straight into the dot rather
//     than the line cornering past it to a dot on the tread.
//   - Flat node (same value as the previous column, e.g. a kyoom 0-day) has no
//     riser; the marker stays in its own column, on the horizontal run.
//   - A terminal node (x == chartWidth-1) is likewise drawn one column to its
//     left (the last column is never drawn), so it's retargeted there. Nodes
//     anchor at local midnight, so one rarely lands on the last column, but the
//     guard keeps the helper correct.
//
// As a backstop, a node whose projected cell still isn't on the drawn line (a
// space) is skipped rather than dotting empty space.
func overlayDatapointMarkers(graph string, nodeCols []int, datapointValues, roadValues []float64, gutter, chartWidth int) string {
	if gutter < 0 || len(nodeCols) == 0 || len(nodeCols) > chartWidth/2 {
		return graph
	}

	minimum, maximum := math.Inf(1), math.Inf(-1)
	for _, series := range [][]float64{roadValues, datapointValues} {
		for _, v := range series {
			minimum = math.Min(minimum, v)
			maximum = math.Max(maximum, v)
		}
	}
	interval := math.Abs(maximum - minimum)
	ratio := 1.0
	if interval != 0 {
		ratio = float64(chartHeight) / interval
	}
	intmin2 := int(asciiRound(minimum * ratio))
	rows := int(math.Abs(float64(int(asciiRound(maximum*ratio)) - intmin2)))

	lines := strings.Split(graph, "\n")
	for _, x := range nodeCols {
		if x < 0 || x >= len(datapointValues) {
			continue
		}
		row := rows - (int(asciiRound(datapointValues[x]*ratio)) - intmin2)
		if row < 0 || row >= len(lines) {
			continue
		}
		// A step into this node puts its riser + corner one column to the left;
		// sit the dot on the corner so the riser runs straight into it. A flat node
		// has no riser and keeps its own column. The terminal column is never drawn,
		// so a node there also shifts left.
		col := x
		if col > 0 && (datapointValues[x] != datapointValues[x-1] || col == chartWidth-1) {
			col--
		}
		lines[row] = replaceCellGlyph(lines[row], gutter+1+col, markerGlyph)
	}
	return strings.Join(lines, "\n")
}

// asciiRound mirrors asciigraph's own rounding (round-half-up by magnitude) so a
// marker lands on the exact cell asciigraph drew the blue line in.
func asciiRound(input float64) float64 {
	sign := 1.0
	if input < 0 {
		sign, input = -1, -input
	}
	if _, frac := math.Modf(input); frac >= 0.5 {
		return math.Ceil(input) * sign
	}
	return math.Floor(input) * sign
}

// replaceCellGlyph replaces the rune at visible column targetCol in an
// ANSI-coloured line, copying SGR escape sequences (which occupy no visible
// column) verbatim so the replacement inherits the cell's colour. A space at the
// target — meaning the projected node isn't on the drawn line — is left alone,
// so a marker never floats in empty space.
func replaceCellGlyph(line string, targetCol int, glyph rune) string {
	var b strings.Builder
	col := 0
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			b.WriteRune(r)
			for i++; i < len(runes); i++ {
				b.WriteRune(runes[i])
				if runes[i] == 'm' {
					break
				}
			}
			continue
		}
		if col == targetCol && r != ' ' {
			b.WriteRune(glyph)
		} else {
			b.WriteRune(r)
		}
		col++
	}
	return b.String()
}

// indentLines prefixes n spaces to each non-empty line. Used to align the plot
// and date axis with the 2-space padding the surrounding review UI uses.
func indentLines(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if ln != "" {
			lines[i] = pad + ln
		}
	}
	return strings.Join(lines, "\n")
}

// plotGutterWidth returns the visible column index of asciigraph's y-axis
// (the `┤`/`┼` rune), which is exactly one column left of the plot area. Those
// axis runes appear only in the gutter, never in the plotted line, so the
// first one on any row marks the boundary. Returns -1 if not found.
func plotGutterWidth(graph string) int {
	for _, line := range strings.Split(graph, "\n") {
		plain := ansiPattern.ReplaceAllString(line, "")
		for i, r := range []rune(plain) {
			if r == '┤' || r == '┼' {
				return i
			}
		}
	}
	return -1
}

// renderXAxis builds a date axis (a tick row and a label row) aligned beneath a
// chartWidth-wide plot whose first column sits at gutter+1. Ticks are spaced to
// fit the width; the first label is left-aligned to its tick, the last
// right-aligned, the rest centred, and any label that would collide with the
// previous one is dropped. Returns "" when the gutter couldn't be located.
func renderXAxis(start, end time.Time, gutter, chartWidth int) string {
	if gutter < 0 || chartWidth < 2 {
		return ""
	}

	plotStart := gutter + 1
	total := plotStart + chartWidth

	// One tick per ~18 columns, clamped so labels ("Jan 2") have room.
	ticks := chartWidth/18 + 1
	if ticks < 2 {
		ticks = 2
	}
	if ticks > 6 {
		ticks = 6
	}

	tickRow := make([]rune, total)
	labelRow := make([]rune, total)
	for i := range tickRow {
		tickRow[i] = ' '
		labelRow[i] = ' '
	}

	span := end.Sub(start)
	lastLabelEnd := -1
	for i := 0; i < ticks; i++ {
		f := float64(i) / float64(ticks-1)
		col := plotStart + int(math.Round(f*float64(chartWidth-1)))
		if col >= total {
			col = total - 1
		}
		tickRow[col] = '┬'

		label := []rune(start.Add(time.Duration(float64(span) * f)).Format("Jan 2"))
		var pos int
		switch i {
		case 0:
			pos = col // left-align under the first tick
		case ticks - 1:
			pos = col - len(label) + 1 // right-align under the last tick
		default:
			pos = col - len(label)/2 // centre on the tick
		}
		if pos < 0 {
			pos = 0
		}
		if pos+len(label) > total {
			pos = total - len(label)
		}
		if pos <= lastLabelEnd {
			continue // would collide with the previous label
		}
		copy(labelRow[pos:], label)
		lastLabelEnd = pos + len(label) - 1
	}

	return strings.TrimRight(string(tickRow), " ") + "\n" + strings.TrimRight(string(labelRow), " ")
}

// roadValuesForTimeframe samples a parsed bright red line into numPoints
// chart columns spanning [startTime, endTime].
//
// Each column is sampled at its RIGHT edge (the next column's instant): a
// column stands for the time span up to the next column, and datapointSeries
// assigns a day to the column at-or-before it (int truncation). Sampling the
// right edge makes a day-snapped road knot change value in that same column,
// so a road riser and a same-day data riser land in the same column instead
// of one apart.
//
// The last column has no next column; its sample is clamped to endTime so the
// rightmost column — the one read as "now" — never shows a value extrapolated
// a column-width past the window's end (real roads almost always keep running
// past now, so an unclamped sample would be visibly in the future).
func roadValuesForTimeframe(r road, startTime, endTime time.Time, numPoints int) []float64 {
	values := make([]float64, numPoints)
	if numPoints == 1 {
		values[0] = r.valueAt(startTime)
		return values
	}

	duration := endTime.Sub(startTime)
	for i := 0; i < numPoints; i++ {
		t := startTime.Add(time.Duration(float64(duration) * float64(i+1) / float64(numPoints-1)))
		if t.After(endTime) {
			t = endTime
		}
		values[i] = r.valueAt(t)
	}
	return values
}

// daysnapRoad floors every segment boundary to local midnight in loc, putting
// road knots on the same day grid the datapoints are bucketed on (beebrain's
// daysnap equivalent). Flooring is monotone, so segment order is preserved;
// a vertical step's two equal boundaries stay equal, and a sub-day segment
// collapsing to zero duration is handled by valueAt like any vertical step.
//
// slopePerDay is recomputed from the snapped boundaries (0 for zero-duration
// steps, matching parseRoad's vertical-step convention): valueAt's before-start
// extrapolation branch reads it directly, so leaving the pre-snap slope in
// place would extrapolate along a slope inconsistent with the segment's own
// snapped endpoints.
func daysnapRoad(r road, loc *time.Location) road {
	snapped := make(road, len(r))
	for i, seg := range r {
		seg.startT = floorUnixToDay(seg.startT, loc)
		seg.endT = floorUnixToDay(seg.endT, loc)
		if seg.endT == seg.startT {
			seg.slopePerDay = 0
		} else {
			seg.slopePerDay = (seg.endV - seg.startV) / (seg.endT - seg.startT) * 86400.0
		}
		snapped[i] = seg
	}
	return snapped
}

// floorUnixToDay floors a unix-seconds instant to midnight of its calendar day
// in loc.
func floorUnixToDay(t float64, loc *time.Location) float64 {
	d := time.Unix(int64(t), 0).In(loc)
	return float64(time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc).Unix())
}
