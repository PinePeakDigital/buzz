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
	if !strings.Contains(chart, "datapoints") || !strings.Contains(chart, "bright red line") {
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

func TestRenderGoalChartStaleGoalStillCharts(t *testing.T) {
	// A goal not updated within the last 30 days (and with no tmin/tmax) used to
	// render no chart at all: the fallback window was anchored at now, so every
	// datapoint fell before it. The default window now shifts back to the most
	// recent datapoint, so the goal still charts. This is the fix for graphs
	// appearing only on recently-touched goals and seeming random.
	old := time.Now().AddDate(0, 0, -60)
	goal := Goal{
		Slug:  "stale",
		Yaw:   1,
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: old.Unix(), Value: 5.0},
			{Timestamp: old.AddDate(0, 0, 1).Unix(), Value: 7.0},
		},
		// No Tmin/Tmax → data-aware default window, which now reaches the points.
	}
	if chart := renderGoalChart(goal, 100); chart == "" {
		t.Error("expected a chart for a stale goal whose datapoints predate 30 days")
	}
}

func TestChartTimeframeDefaultsStartToInitday(t *testing.T) {
	// With no user-set tmin/tmax, the window defaults to the goal's start
	// (initday) through now, charting the whole goal — matching Beeminder's
	// default of showing all data. initday carries a midday (deadline-aligned)
	// instant, which must be floored to the start of its local day.
	now := time.Date(2026, 6, 10, 23, 0, 0, 0, time.Local)
	initday := time.Date(2024, 1, 15, 16, 30, 0, 0, time.Local)
	goal := Goal{Slug: "wholehistory", Initday: initday.Unix()}

	start, end := chartTimeframe(goal, now)
	wantStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local) // floored to local midnight
	if !start.Equal(wantStart) {
		t.Errorf("start = %s, want goal-start day floored to local midnight %s", start, wantStart)
	}
	if !end.Equal(now) {
		t.Errorf("end = %s, want now %s", end, now)
	}
}

func TestRenderGoalChartIncludesSameDayDatapointBeforeInitdayInstant(t *testing.T) {
	// A brand-new goal whose initday instant is midday and whose only datapoint
	// was logged earlier the same day: flooring initday to the start of the day
	// must keep that datapoint inside the window so the goal still charts.
	day := time.Date(2026, 6, 10, 0, 0, 0, 0, time.Local)
	initday := day.Add(16 * time.Hour) // midday-ish initday instant
	dp := day.Add(6 * time.Hour)       // logged earlier the same day
	now := day.Add(20 * time.Hour)
	goal := Goal{
		Slug:       "newgoal",
		Yaw:        1,
		Kyoom:      true,
		Initday:    initday.Unix(),
		Datapoints: []Datapoint{{Timestamp: dp.Unix(), Value: 1.0}},
	}

	start, _ := chartTimeframe(goal, now)
	if dp.Before(start) {
		t.Fatalf("datapoint %s fell before window start %s", dp, start)
	}
	if chart := renderGoalChart(goal, 100); chart == "" {
		t.Error("expected a chart for a same-day datapoint logged before the initday instant")
	}
}

func TestChartTimeframeWidensEndForFutureDatapoint(t *testing.T) {
	// A datapoint timestamped after now (e.g. a scheduled/future-dated point)
	// would otherwise sit past the default end (now); the end widens to include
	// it.
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local)
	future := now.AddDate(0, 0, 3)
	goal := Goal{
		Slug:       "future",
		Initday:    now.AddDate(0, 0, -10).Unix(),
		Datapoints: []Datapoint{{Timestamp: future.Unix(), Value: 1.0}},
	}

	_, end := chartTimeframe(goal, now)
	if end.Before(future) {
		t.Errorf("end = %s, want >= future datapoint %s", end, future)
	}
}

func TestLastDatapointTime(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Empty: ok is false.
	if _, ok := lastDatapointTime(Goal{}); ok {
		t.Error("expected ok=false for a goal with no datapoints")
	}

	// Single datapoint.
	if got, ok := lastDatapointTime(Goal{Datapoints: []Datapoint{
		{Timestamp: base.Unix(), Value: 1},
	}}); !ok || !got.Equal(base) {
		t.Errorf("single: got %s ok=%v, want %s true", got, ok, base)
	}

	// Unsorted: must return the maximum timestamp, not the last element.
	want := base.AddDate(0, 0, 100)
	got, ok := lastDatapointTime(Goal{Datapoints: []Datapoint{
		{Timestamp: want.Unix(), Value: 1},
		{Timestamp: base.Unix(), Value: 2},
		{Timestamp: base.AddDate(0, 0, 50).Unix(), Value: 3},
	}})
	if !ok || !got.Equal(want) {
		t.Errorf("unsorted: got %s ok=%v, want %s true", got, ok, want)
	}
}

func TestChartTimeframeHonorsTminWithoutTmax(t *testing.T) {
	// Beeminder force-nulls tmax once it's in the past, so tmax is null on
	// virtually every goal while tmin is commonly set. The window must still
	// honor an explicit tmin rather than collapsing onto the default window.
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local)
	goal := Goal{
		Slug: "has-tmin",
		Tmin: "2024-08-02",
		// Tmax empty (null from the API).
	}
	start, end := chartTimeframe(goal, now)

	wantStart := time.Date(2024, 8, 2, 0, 0, 0, 0, time.Local)
	if !start.Equal(wantStart) {
		t.Errorf("start = %s, want explicit tmin %s", start, wantStart)
	}
	// With no tmax and no datapoints, the end falls back to now.
	if !end.Equal(now) {
		t.Errorf("end = %s, want fallback to now %s", end, now)
	}
}

func TestRenderGoalChartHonorsTminForStaleGoal(t *testing.T) {
	// Regression for the real-world "fam" goal: tmin set far in the past, tmax
	// null, last datapoint older than 30 days. Honoring tmin as the window start
	// (the end defaults to now) means the goal charts instead of going blank.
	last := time.Now().AddDate(0, 0, -40)
	goal := Goal{
		Slug:  "fam",
		Yaw:   1,
		Kyoom: true,
		Tmin:  last.AddDate(0, 0, -300).Format("2006-01-02"),
		Datapoints: []Datapoint{
			{Timestamp: last.AddDate(0, 0, -5).Unix(), Value: 2.0},
			{Timestamp: last.Unix(), Value: 3.0},
		},
	}
	if chart := renderGoalChart(goal, 100); chart == "" {
		t.Error("expected a chart for a goal with an explicit tmin and stale data")
	}
}

func TestProcessCumulativeNoInWindowDatapointsReturnsNil(t *testing.T) {
	// Invariant: when no datapoints fall inside the window, processCumulative
	// returns nil even though earlier datapoints push the running total above
	// zero — a lone carry-over anchor must never draw a dataless flat line.
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 30)
	goal := Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: start.AddDate(0, 0, -10).Unix(), Value: 5.0},
			{Timestamp: start.AddDate(0, 0, -5).Unix(), Value: 7.0},
		},
	}
	if got := processCumulative(goal, start, end); got != nil {
		t.Errorf("expected nil when no datapoints fall in the window, got %v", got)
	}
}

func TestProcessCumulativeValues(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 2)

	check := func(name string, got []timedValue, want []float64) {
		if len(got) != len(want) {
			t.Fatalf("%s: got %d values, want %d", name, len(got), len(want))
		}
		for i, w := range want {
			if got[i].value != w {
				t.Errorf("%s: value[%d] = %v, want %v", name, i, got[i].value, w)
			}
		}
	}

	// Two datapoints inside the window, nothing before it: the anchor carries 0,
	// then the running total reaches 5, then 8.
	check("in-window", processCumulative(Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: start.Unix(), Value: 5},
			{Timestamp: start.AddDate(0, 0, 1).Unix(), Value: 3},
		},
	}, start, end), []float64{0, 5, 8})

	// A datapoint before the window feeds the carry-over anchor (10) but isn't
	// itself plotted; in-window points continue the running total: 15, then 18.
	check("carry-over", processCumulative(Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: start.AddDate(0, 0, -1).Unix(), Value: 10},
			{Timestamp: start.Unix(), Value: 5},
			{Timestamp: start.AddDate(0, 0, 1).Unix(), Value: 3},
		},
	}, start, end), []float64{10, 15, 18})
}

func TestProcessDatapointsNonCumulative(t *testing.T) {
	start := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 5)

	// Non-cumulative goal: only in-window points, sorted ascending, raw values.
	goal := Goal{
		Datapoints: []Datapoint{
			{Timestamp: start.AddDate(0, 0, 3).Unix(), Value: 30},  // in window, later
			{Timestamp: start.AddDate(0, 0, -2).Unix(), Value: 99}, // before window → excluded
			{Timestamp: start.AddDate(0, 0, 1).Unix(), Value: 10},  // in window, earlier
			{Timestamp: end.AddDate(0, 0, 2).Unix(), Value: 88},    // after window → excluded
		},
	}

	got := processDatapoints(goal, start, end)
	if len(got) != 2 {
		t.Fatalf("expected 2 in-window datapoints, got %d", len(got))
	}
	if got[0].timestamp >= got[1].timestamp {
		t.Error("expected datapoints sorted ascending by timestamp")
	}
	if got[0].value != 10 || got[1].value != 30 {
		t.Errorf("expected raw (un-summed) values [10 30], got [%v %v]", got[0].value, got[1].value)
	}
}

func TestDatapointSeriesInterpolation(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 10)

	// Endpoints at the first/last columns; interior columns linearly interpolated.
	got := datapointSeries([]timedValue{
		{timestamp: start.Unix(), value: 0},
		{timestamp: end.Unix(), value: 100},
	}, start, end, 11, false)
	if len(got) != 11 {
		t.Fatalf("expected 11 columns, got %d", len(got))
	}
	if got[0] != 0 || got[10] != 100 {
		t.Errorf("endpoints: got[0]=%v got[10]=%v, want 0 and 100", got[0], got[10])
	}
	if got[5] < 49.9 || got[5] > 50.1 {
		t.Errorf("midpoint interpolation: got[5]=%v, want ~50", got[5])
	}

	// A single datapoint fills the whole row with its value (no gaps, no NaN).
	single := datapointSeries([]timedValue{
		{timestamp: start.AddDate(0, 0, 5).Unix(), value: 7},
	}, start, end, 11, false)
	for i, v := range single {
		if v != 7 {
			t.Errorf("single datapoint flat-fill: col %d = %v, want 7", i, v)
		}
	}
}

// TestDatapointSeriesCumulativeSteps guards the cumulative-goal fix: a kyoom
// goal's line must step (hold the previous total, then jump at the datapoint),
// not draw a diagonal ramp between points. Reproduces the integrations-goal case
// where a value-0 anchor and a same-window value-1 point produced a misleading
// diagonal instead of Beeminder's vertical riser.
func TestDatapointSeriesCumulativeSteps(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 10)

	// Anchor of 0 at the window start, then a cumulative total of 1 at the
	// midpoint. Columns 0..mid-1 must stay flat at 0 (no ramp), the jump lands at
	// the midpoint column, and everything after holds 1.
	got := datapointSeries([]timedValue{
		{timestamp: start.Unix(), value: 0},
		{timestamp: start.AddDate(0, 0, 5).Unix(), value: 1},
	}, start, end, 11, true)
	if len(got) != 11 {
		t.Fatalf("expected 11 columns, got %d", len(got))
	}
	for i := 0; i < 5; i++ {
		if got[i] != 0 {
			t.Errorf("cumulative step: col %d = %v, want 0 (flat, no diagonal)", i, got[i])
		}
	}
	for i := 5; i < 11; i++ {
		if got[i] != 1 {
			t.Errorf("cumulative step: col %d = %v, want 1 (held after jump)", i, got[i])
		}
	}
}
