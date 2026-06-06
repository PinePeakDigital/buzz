package main

import (
	"strings"
	"testing"
	"time"
)

// roadallRow builds a Beeminder-style roadall row. Pass nil for v or r to
// leave that column unset (rows past the anchor have exactly one of v/r).
func roadallRow(t float64, v, r *float64) []*float64 {
	tp := t
	return []*float64{&tp, v, r}
}

func fptr(f float64) *float64 { return &f }

func TestRenderGoalChartWithNoDatapoints(t *testing.T) {
	goal := Goal{
		Slug:       "test-goal",
		Datapoints: []Datapoint{},
	}

	if chart := renderGoalChart(goal, 80); chart != "" {
		t.Error("Expected empty chart for goal with no datapoints")
	}
}

func TestRenderGoalChartWithDatapoints(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	goal := Goal{
		Slug: "test-goal",
		Yaw:  1, // Do more
		Datapoints: []Datapoint{
			{Timestamp: yesterday.Unix(), Value: 5.0},
			{Timestamp: now.Unix(), Value: 10.0},
		},
		Tmin: yesterday.Format("2006-01-02"),
		Tmax: now.Format("2006-01-02"),
		Roadall: [][]*float64{
			roadallRow(float64(yesterday.Unix()), fptr(0.0), nil),
			roadallRow(float64(now.Unix()), fptr(5.0), nil),
		},
	}

	chart := renderGoalChart(goal, 80)
	if chart == "" {
		t.Fatal("Expected non-empty chart for goal with datapoints")
	}
	if !strings.Contains(chart, "Goal Progress Chart") {
		t.Error("Expected chart to contain 'Goal Progress Chart'")
	}
	if !strings.Contains(chart, "Do More") {
		t.Error("Expected chart to contain 'Do More'")
	}
	if !strings.Contains(chart, "datapoints") && !strings.Contains(chart, "bright red line") {
		t.Error("Expected chart to contain caption")
	}
}

func TestRenderGoalChartCumulative(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	goal := Goal{
		Slug:  "test-goal",
		Yaw:   1, // Do more
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: yesterday.Unix(), Value: 5.0},
			{Timestamp: now.Unix(), Value: 3.0},
		},
		Tmin: yesterday.Format("2006-01-02"),
		Tmax: now.Format("2006-01-02"),
	}

	chart := renderGoalChart(goal, 80)
	if chart == "" {
		t.Fatal("Expected non-empty chart for cumulative goal")
	}
	if !strings.Contains(chart, "Cumulative") {
		t.Error("Expected chart to indicate cumulative goal")
	}
}

func TestRenderGoalChartDoLess(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	goal := Goal{
		Slug: "test-goal",
		Yaw:  -1, // Do less
		Datapoints: []Datapoint{
			{Timestamp: yesterday.Unix(), Value: 10.0},
			{Timestamp: now.Unix(), Value: 5.0},
		},
		Tmin: yesterday.Format("2006-01-02"),
		Tmax: now.Format("2006-01-02"),
	}

	chart := renderGoalChart(goal, 80)
	if chart == "" {
		t.Fatal("Expected non-empty chart for do less goal")
	}
	if !strings.Contains(chart, "Do Less") {
		t.Error("Expected chart to indicate 'Do Less' goal")
	}
}

func TestRenderGoalChartIncludesEndOfTmaxDay(t *testing.T) {
	// Regression: Tmax is a date, not an instant, so a datapoint logged late
	// on the user's local Tmax day must still fall inside the chart window.
	tmax := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	dpTime := time.Date(2024, 1, 15, 23, 30, 0, 0, time.Local)

	goal := Goal{
		Slug: "boundary",
		Yaw:  1,
		Tmin: tmax.AddDate(0, 0, -7).Format("2006-01-02"),
		Tmax: tmax.Format("2006-01-02"),
		Datapoints: []Datapoint{
			{Timestamp: dpTime.Unix(), Value: 1.0},
		},
	}

	if chart := renderGoalChart(goal, 80); chart == "" {
		t.Error("expected datapoint at 23:30 local on Tmax to be included; chart is empty")
	}
}

func TestRenderGoalChartWithFallbackTimeframe(t *testing.T) {
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	goal := Goal{
		Slug: "test-goal",
		Yaw:  1,
		Datapoints: []Datapoint{
			{Timestamp: thirtyDaysAgo.Unix(), Value: 5.0},
			{Timestamp: now.Unix(), Value: 10.0},
		},
		// No Tmin/Tmax - should use fallback
	}

	if chart := renderGoalChart(goal, 80); chart == "" {
		t.Error("Expected non-empty chart even without tmin/tmax")
	}
}

func TestGetRoadValueAtTime(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Anchor at v=0, then a rate-only segment 10 days later at 1/day; day 5
	// should interpolate to ~5.
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), nil, fptr(1.0)),
		},
	}

	if value := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, 5)); value < 4.9 || value > 5.1 {
		t.Errorf("Expected value around 5.0, got %f", value)
	}
}

func TestGetRoadValuesForTimeframe(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := baseTime.AddDate(0, 0, 10)

	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(endTime.Unix()), fptr(10.0), nil),
		},
	}

	values := getRoadValuesForTimeframe(goal, baseTime, endTime, 11)
	if len(values) != 11 {
		t.Fatalf("Expected 11 values, got %d", len(values))
	}
	if values[0] < -0.5 || values[0] > 0.5 {
		t.Errorf("Expected first value around 0, got %f", values[0])
	}
	if values[10] < 9.5 || values[10] > 10.5 {
		t.Errorf("Expected last value around 10, got %f", values[10])
	}
}

func TestGetRoadValueAtTimeShortRoad(t *testing.T) {
	// Fewer than 2 rows is unusable — short-circuit to 0 rather than deref.
	if v := getRoadValueAtTime(Goal{}, time.Now()); v != 0 {
		t.Errorf("empty roadall: expected 0, got %f", v)
	}
	single := Goal{Roadall: [][]*float64{roadallRow(0, fptr(5.0), nil)}}
	if v := getRoadValueAtTime(single, time.Now()); v != 0 {
		t.Errorf("single-row roadall: expected 0, got %f", v)
	}
}

func TestGetRoadValueAtTimePastEndOfRoad(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), fptr(10.0), nil),
		},
	}
	if got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, 20)); got < 9.9 || got > 10.1 {
		t.Errorf("past end of road: expected ~10, got %f", got)
	}
}

func TestGetRoadValueAtTimeBeforeAnchorAmbiguousRow(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Row 1 has both v and r set (malformed). The before-anchor branch must
	// bail rather than extrapolate from one interpretation.
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), fptr(10.0), fptr(1.0)),
		},
	}
	if got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, -5)); got != 0 {
		t.Errorf("before-anchor ambiguous row: expected 0 (anchor value), got %f", got)
	}
}

func TestGetRoadValueAtTimeBeforeAnchorUnknownRunits(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Backward extrapolation with runits we can't translate must bail to the
	// anchor value rather than apply a dimensionally-wrong slope.
	goal := Goal{
		Runits: "lightyears",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), nil, fptr(1.0)),
		},
	}
	if got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, -5)); got != 0 {
		t.Errorf("before-anchor unknown runits: expected 0 (anchor value), got %f", got)
	}
}

func TestGetRoadValueAtTimeBeforeAnchor(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), nil, fptr(1.0)),
		},
	}
	// Five days before the anchor → extrapolate at -1/day → ~-5.
	if got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, -5)); got < -5.1 || got > -4.9 {
		t.Errorf("before anchor extrapolation: expected ~-5, got %f", got)
	}
}

func TestGetRoadValueAtTimeUnknownRunits(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Rate-only segment with unrecognised runits → bail rather than treat the
	// rate as gunits/day.
	goal := Goal{
		Runits: "lightyears",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), nil, fptr(1.0)),
		},
	}
	if got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, 5)); got != 0 {
		t.Errorf("unknown runits: expected 0 (bail), got %f", got)
	}
}

func TestGetRoadValueAtTimeAmbiguousRow(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Non-anchor row with both v and r set is ambiguous; bail to the prior
	// anchor (0) rather than guess.
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), fptr(10.0), fptr(1.0)),
		},
	}
	if got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, 5)); got != 0 {
		t.Errorf("ambiguous row: expected prior anchor value 0, got %f", got)
	}
}

func TestGetRoadValuesForTimeframeSinglePoint(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := baseTime.AddDate(0, 0, 10)
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(endTime.Unix()), fptr(10.0), nil),
		},
	}
	// numPoints==1 must not divide by (numPoints-1); it returns a single
	// sample at startTime.
	values := getRoadValuesForTimeframe(goal, baseTime.AddDate(0, 0, 5), endTime, 1)
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if values[0] < 4.9 || values[0] > 5.1 {
		t.Errorf("numPoints=1 sample: expected ~5, got %f", values[0])
	}
}

func TestGetRoadValuesForTimeframeEmpty(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := baseTime.AddDate(0, 0, 10)

	goal := Goal{Roadall: [][]*float64{}} // No road data

	values := getRoadValuesForTimeframe(goal, baseTime, endTime, 10)
	if len(values) != 10 {
		t.Fatalf("Expected 10 values, got %d", len(values))
	}
	for i, v := range values {
		if v != 0 {
			t.Errorf("Expected value at index %d to be 0, got %f", i, v)
		}
	}
}

func TestRenderXAxisAlignment(t *testing.T) {
	start := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	gutter, chartWidth := 7, 80

	axis := renderXAxis(start, end, gutter, chartWidth)
	if axis == "" {
		t.Fatal("expected a non-empty axis")
	}
	lines := strings.Split(axis, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected tick row + label row, got %d lines", len(lines))
	}
	tickRow, labelRow := lines[0], lines[1]

	// Measure tick positions in runes ('┬' is multi-byte, so byte offsets lie).
	ticks := []rune(tickRow)
	first, last := -1, -1
	for i, r := range ticks {
		if r == '┬' {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	// First tick sits at the first plot column (gutter+1); last at the last.
	if first != gutter+1 {
		t.Errorf("first tick at col %d, want %d", first, gutter+1)
	}
	if last != gutter+chartWidth {
		t.Errorf("last tick at col %d, want %d", last, gutter+chartWidth)
	}
	// Endpoints' dates appear, in order.
	if !strings.Contains(labelRow, "May 7") {
		t.Errorf("label row missing start date: %q", labelRow)
	}
	if !strings.Contains(labelRow, "Jun 6") {
		t.Errorf("label row missing end date: %q", labelRow)
	}
	// Nothing spills into the y-axis gutter.
	if strings.TrimSpace(labelRow[:gutter+1]) != "" {
		t.Errorf("label row writes into the gutter: %q", labelRow[:gutter+1])
	}
}

func TestRenderXAxisNoGutter(t *testing.T) {
	if got := renderXAxis(time.Now(), time.Now().AddDate(0, 0, 1), -1, 80); got != "" {
		t.Errorf("expected empty axis when gutter not found, got %q", got)
	}
}

func TestRenderGoalChartHasDateAxis(t *testing.T) {
	now := time.Now()
	goal := Goal{
		Slug: "axis", Yaw: 1,
		Datapoints: []Datapoint{
			{Timestamp: now.AddDate(0, 0, -20).Unix(), Value: 1},
			{Timestamp: now.Unix(), Value: 5},
		},
	}
	chart := renderGoalChart(goal, 100)
	if chart == "" {
		t.Fatal("expected a chart")
	}
	if !strings.Contains(chart, "┬") {
		t.Error("expected an x-axis tick row in the rendered chart")
	}
}

func TestChartTimeframeTmaxAcrossDSTFallBack(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tz data unavailable")
	}
	orig := time.Local
	time.Local = loc
	defer func() { time.Local = orig }()

	// 2023-11-05 is a 25h day in New York (DST ends 02:00). The old
	// midnight+24h-1s math landed at 22:59:59 local, wrongly excluding a
	// datapoint logged at 23:30 that same day.
	g := Goal{Tmin: "2023-10-29", Tmax: "2023-11-05"}
	_, end := chartTimeframe(g, time.Now())

	dp := time.Date(2023, 11, 5, 23, 30, 0, 0, loc)
	if dp.After(end) {
		t.Errorf("23:30 on a 25h DST day excluded: end=%s dp=%s", end, dp)
	}
	// End must stay within the Tmax calendar day, not spill into the next.
	if end.Day() != 5 {
		t.Errorf("end spilled past the Tmax day: %s", end)
	}
}
