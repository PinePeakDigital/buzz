package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
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

	chartWidth := width - 10 // leave room for padding and axis labels
	if chartWidth < minChartWidth {
		chartWidth = minChartWidth
	}
	if chartWidth > maxChartWidth {
		chartWidth = maxChartWidth
	}

	roadValues := getRoadValuesForTimeframe(goal, startTime, endTime, chartWidth)
	datapointValues := datapointSeries(processed, startTime, endTime, chartWidth)

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
// For cumulative (kyoom) goals the plotted value is the running total, so the
// sum is accumulated across ALL datapoints (including those before the window)
// and a synthetic anchor is prepended at startTime carrying the total reached
// just before the window — otherwise the in-window line would start from zero
// instead of where the goal actually stood.
func processDatapoints(goal Goal, startTime, endTime time.Time) []timedValue {
	if goal.Kyoom {
		return processCumulative(goal, startTime, endTime)
	}

	var processed []timedValue
	for _, dp := range goal.Datapoints {
		dpTime := time.Unix(dp.Timestamp, 0)
		if !dpTime.Before(startTime) && !dpTime.After(endTime) {
			processed = append(processed, timedValue{timestamp: dp.Timestamp, value: dp.Value})
		}
	}
	sort.Slice(processed, func(i, j int) bool {
		return processed[i].timestamp < processed[j].timestamp
	})
	return processed
}

// processCumulative builds the in-window plot series for a cumulative (kyoom)
// goal: it sums every datapoint in chronological order so each in-window point
// carries the running total, prepends a synthetic anchor at the window start
// holding the total reached just before it, and returns nil when no datapoints
// fall inside the window (so a pure carry-over never draws a dataless line).
func processCumulative(goal Goal, startTime, endTime time.Time) []timedValue {
	sorted := make([]Datapoint, len(goal.Datapoints))
	copy(sorted, goal.Datapoints)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp < sorted[j].Timestamp
	})

	sum := 0.0
	startSum := 0.0
	var processed []timedValue
	for _, dp := range sorted {
		dpTime := time.Unix(dp.Timestamp, 0)
		switch {
		case dpTime.Before(startTime):
			sum += dp.Value
			startSum = sum
		case !dpTime.After(endTime):
			sum += dp.Value
			processed = append(processed, timedValue{timestamp: dp.Timestamp, value: sum})
		}
		// Datapoints after endTime are ignored.
	}

	// No datapoints inside the window means nothing to chart — even if earlier
	// datapoints pushed the running total above zero. (renderGoalChart's
	// contract is to return empty when none fall inside the window; a lone
	// carry-over anchor would otherwise draw a flat, dataless line.)
	if len(processed) == 0 {
		return nil
	}

	// Prepend an anchor at the window start carrying the running total so far,
	// so the line begins where the goal actually stood rather than at zero.
	return append([]timedValue{{timestamp: startTime.Unix(), value: startSum}}, processed...)
}

// datapointSeries maps processed datapoints onto chartWidth evenly-spaced
// columns and fills the gaps: each datapoint lands in the column matching its
// position in the timeframe, columns before the first / after the last hold
// that endpoint's value, and interior gaps are linearly interpolated so the
// plotted line is continuous.
func datapointSeries(processed []timedValue, startTime, endTime time.Time, chartWidth int) []float64 {
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
		return values
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
		if i > prevDP+1 {
			startVal := values[prevDP]
			endVal := values[i]
			for j := prevDP + 1; j < i; j++ {
				ratio := float64(j-prevDP) / float64(i-prevDP)
				values[j] = startVal + ratio*(endVal-startVal)
			}
		}
		prevDP = i
	}

	return values
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

// getRoadValuesForTimeframe samples the bright red line at numPoints evenly
// distributed instants across [startTime, endTime] — one per chart column.
func getRoadValuesForTimeframe(goal Goal, startTime, endTime time.Time, numPoints int) []float64 {
	values := make([]float64, numPoints)
	if len(goal.Roadall) == 0 {
		return values
	}
	if numPoints == 1 {
		values[0] = getRoadValueAtTime(goal, startTime)
		return values
	}

	duration := endTime.Sub(startTime)
	for i := 0; i < numPoints; i++ {
		t := startTime.Add(time.Duration(float64(duration) * float64(i) / float64(numPoints-1)))
		values[i] = getRoadValueAtTime(goal, t)
	}
	return values
}

// getRoadValueAtTime interpolates the bright red line's value at time t.
//
// Beeminder's roadall is a piecewise schedule: the first row is the anchor
// (t, v set, r nil), and each subsequent row has exactly one of v/r null —
// either the value at that t, or the rate (per runits) used to get there. To
// interpolate we walk forward, materialising each row's value from the prior
// anchor and the row's rate when the row's own value is missing.
func getRoadValueAtTime(goal Goal, t time.Time) float64 {
	if len(goal.Roadall) < 2 {
		return 0
	}

	target := float64(t.Unix())

	// Resolve the anchor (row 0): must have t and v set.
	first := goal.Roadall[0]
	if len(first) < 3 || first[0] == nil || first[1] == nil {
		return 0
	}
	prevT := *first[0]
	prevV := *first[1]

	// Before the road starts: extrapolate backwards along the first segment's
	// slope so the chart can still draw a value.
	if target < prevT {
		slope, ok := segmentSlopePerSecond(goal, 1, prevT, prevV)
		if !ok {
			return prevV
		}
		return prevV + slope*(target-prevT)
	}

	for i := 1; i < len(goal.Roadall); i++ {
		cur := goal.Roadall[i]
		if len(cur) < 3 || cur[0] == nil {
			return prevV
		}
		curT := *cur[0]

		// Per the Beeminder spec a non-anchor row has exactly one of v/r set.
		// Both nil or both set is ambiguous — bail at the prior anchor rather
		// than guess an interpretation (matches slope.go's validation).
		if (cur[1] == nil) == (cur[2] == nil) {
			return prevV
		}

		var curV float64
		switch {
		case cur[1] != nil:
			curV = *cur[1]
		case cur[2] != nil:
			// ratePerDay passes unknown runits through unchanged, so a
			// per-week rate would be misread as per-day. Bail rather than
			// draw a dimensionally-wrong road.
			if !isKnownRunits(goal.Runits) {
				return prevV
			}
			rps := ratePerDay(*cur[2], goal.Runits) / 86400.0
			curV = prevV + rps*(curT-prevT)
		}

		if target <= curT {
			if curT == prevT {
				return curV
			}
			frac := (target - prevT) / (curT - prevT)
			return prevV + frac*(curV-prevV)
		}

		prevT = curT
		prevV = curV
	}

	// Past the end of the road: hold the last materialised value.
	return prevV
}

// segmentSlopePerSecond returns the slope (gunits/second) of the roadall
// segment ending at index i, given the prior anchor (prevT, prevV). Used to
// extrapolate before the start of the road. ok is false when the segment is
// missing, malformed, ambiguous, or expressed in runits we can't translate.
func segmentSlopePerSecond(goal Goal, i int, prevT, prevV float64) (float64, bool) {
	if i >= len(goal.Roadall) {
		return 0, false
	}
	cur := goal.Roadall[i]
	if len(cur) < 3 || cur[0] == nil {
		return 0, false
	}
	// Ambiguous rows (both v/r nil or both set) are malformed per the spec —
	// bail rather than pick an interpretation. Mirrors getRoadValueAtTime.
	if (cur[1] == nil) == (cur[2] == nil) {
		return 0, false
	}
	if cur[2] != nil {
		if !isKnownRunits(goal.Runits) {
			return 0, false
		}
		return ratePerDay(*cur[2], goal.Runits) / 86400.0, true
	}
	dt := *cur[0] - prevT
	if dt == 0 {
		return 0, false
	}
	return (*cur[1] - prevV) / dt, true
}
