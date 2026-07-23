package main

import (
	"math"
	"slices"
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

// chartTestRoad builds a minimal valid bright red line spanning [start, end]:
// a value-row road from 0 to 5. Value rows (not rate rows) are used because
// these chart fixtures don't set Runits, and value rows don't require known
// runits to parse. renderGoalChart now refuses to draw a chart without a road
// (see ADR-0003), so every fixture that expects a rendered chart needs one.
func chartTestRoad(start, end time.Time) [][]*float64 {
	return [][]*float64{
		roadallRow(float64(start.Unix()), fptr(0.0), nil),
		roadallRow(float64(end.Unix()), fptr(5.0), nil),
	}
}

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

func TestRenderGoalChartMalformedRoad(t *testing.T) {
	// A goal with in-window datapoints but a malformed roadall (a row carrying
	// both a value and a rate) must surface loudly rather than draw a chart —
	// the three-way render outcome from ADR-0003.
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	goal := Goal{
		Slug: "malformed",
		Yaw:  1,
		Datapoints: []Datapoint{
			{Timestamp: yesterday.Unix(), Value: 5.0},
			{Timestamp: now.Unix(), Value: 10.0},
		},
		Tmin: yesterday.Format("2006-01-02"),
		Tmax: now.Format("2006-01-02"),
		Roadall: [][]*float64{
			roadallRow(float64(yesterday.Unix()), fptr(0.0), nil),
			roadallRow(float64(now.Unix()), fptr(5.0), fptr(1.0)), // both set → malformed
		},
	}

	chart := renderGoalChart(goal, 80)
	if !strings.Contains(chart, "Couldn't render the bright red line") {
		t.Errorf("expected a malformed-road warning banner, got %q", chart)
	}
	if strings.Contains(chart, "Goal Progress Chart") {
		t.Error("expected no plotted chart when the road is malformed")
	}
}

func TestRenderGoalChartAbsentRoad(t *testing.T) {
	// A goal with in-window datapoints but no roadall must say the bright red
	// line wasn't populated rather than draw a flat zero line (ADR-0003).
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	goal := Goal{
		Slug: "absent",
		Yaw:  1,
		Datapoints: []Datapoint{
			{Timestamp: yesterday.Unix(), Value: 5.0},
			{Timestamp: now.Unix(), Value: 10.0},
		},
		Tmin: yesterday.Format("2006-01-02"),
		Tmax: now.Format("2006-01-02"),
		// No Roadall → absent.
	}

	chart := renderGoalChart(goal, 80)
	if !strings.Contains(chart, "wasn't populated") {
		t.Errorf("expected a 'not populated' notice, got %q", chart)
	}
	if strings.Contains(chart, "Goal Progress Chart") {
		t.Error("expected no plotted chart when the road is absent")
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
		Tmin:    yesterday.Format("2006-01-02"),
		Tmax:    now.Format("2006-01-02"),
		Roadall: chartTestRoad(yesterday, now),
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
		Tmin:    yesterday.Format("2006-01-02"),
		Tmax:    now.Format("2006-01-02"),
		Roadall: chartTestRoad(yesterday, now),
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
		Roadall: chartTestRoad(tmax.AddDate(0, 0, -7), dpTime),
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
		Roadall: chartTestRoad(thirtyDaysAgo, now),
	}

	if chart := renderGoalChart(goal, 80); chart == "" {
		t.Error("Expected non-empty chart even without tmin/tmax")
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
		Roadall: chartTestRoad(now.AddDate(0, 0, -20), now),
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
		Roadall: chartTestRoad(old, old.AddDate(0, 0, 1)),
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
		Roadall:    chartTestRoad(day, now),
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
		Roadall: chartTestRoad(last.AddDate(0, 0, -5), last),
	}
	if chart := renderGoalChart(goal, 100); chart == "" {
		t.Error("expected a chart for a goal with an explicit tmin and stale data")
	}
}

func TestProcessDatapointsCumulativeNoInWindowReturnsNil(t *testing.T) {
	// Invariant: when no day falls inside the window, a cumulative goal yields nil
	// even though earlier datapoints push the running total above zero — a lone
	// carry-over anchor must never draw a dataless flat line.
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 30)
	goal := Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: start.AddDate(0, 0, -10).Unix(), Daystamp: "20240522", Value: 5.0},
			{Timestamp: start.AddDate(0, 0, -5).Unix(), Daystamp: "20240527", Value: 7.0},
		},
	}
	if got := processDatapoints(goal, start, end); got != nil {
		t.Errorf("expected nil when no datapoints fall in the window, got %v", got)
	}
}

func TestProcessDatapointsCumulativeValues(t *testing.T) {
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

	// Two datapoints inside the window (one per day), nothing before it: the
	// anchor carries 0, then the running total reaches 5, then 8.
	check("in-window", processDatapoints(Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: start.Unix(), Daystamp: "20240101", Value: 5},
			{Timestamp: start.AddDate(0, 0, 1).Unix(), Daystamp: "20240102", Value: 3},
		},
	}, start, end), []float64{0, 5, 8})

	// A datapoint before the window feeds the carry-over anchor (10) but isn't
	// itself plotted; in-window points continue the running total: 15, then 18.
	check("carry-over", processDatapoints(Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: start.AddDate(0, 0, -1).Unix(), Daystamp: "20231231", Value: 10},
			{Timestamp: start.Unix(), Daystamp: "20240101", Value: 5},
			{Timestamp: start.AddDate(0, 0, 1).Unix(), Daystamp: "20240102", Value: 3},
		},
	}, start, end), []float64{10, 15, 18})
}

func TestProcessDatapointsNonCumulative(t *testing.T) {
	start := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 5)

	// Non-cumulative goal: only in-window days, ascending, the day's aggregate
	// (default aggday "last" → the single value).
	goal := Goal{
		Datapoints: []Datapoint{
			{Timestamp: start.AddDate(0, 0, 3).Unix(), Daystamp: "20240113", Value: 30},  // in window, later
			{Timestamp: start.AddDate(0, 0, -2).Unix(), Daystamp: "20240108", Value: 99}, // before window → excluded
			{Timestamp: start.AddDate(0, 0, 1).Unix(), Daystamp: "20240111", Value: 10},  // in window, earlier
			{Timestamp: end.AddDate(0, 0, 2).Unix(), Daystamp: "20240117", Value: 88},    // after window → excluded
		},
	}

	got := processDatapoints(goal, start, end)
	if len(got) != 2 {
		t.Fatalf("expected 2 in-window days, got %d", len(got))
	}
	if got[0].timestamp >= got[1].timestamp {
		t.Error("expected days sorted ascending by time")
	}
	if got[0].value != 10 || got[1].value != 30 {
		t.Errorf("expected per-day (un-summed) values [10 30], got [%v %v]", got[0].value, got[1].value)
	}
}

func TestProcessDatapointsMidDayWindowStartAndNoDaystamp(t *testing.T) {
	// Two subtle paths at once: a window that starts mid-day (as stale goals do,
	// anchored at the last datapoint's timestamp), and datapoints carrying no
	// Daystamp (so the day is derived from the timestamp). The day whose midnight
	// precedes the mid-day startTime must still count as in-window, and the kyoom
	// anchor must sort at-or-before the first day point.
	loc := time.UTC
	day0 := time.Date(2024, 5, 10, 0, 0, 0, 0, loc)
	start := day0.Add(12 * time.Hour) // window starts at noon on day 0
	end := day0.AddDate(0, 0, 2)

	goal := Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: day0.Add(14 * time.Hour).Unix(), Value: 5},                 // no Daystamp → day 0
			{Timestamp: day0.AddDate(0, 0, 1).Add(9 * time.Hour).Unix(), Value: 3}, // day 1
		},
	}
	got := processDatapoints(goal, start, end)

	// anchor (carry 0) + day0 cumulative 5 + day1 cumulative 8.
	want := []float64{0, 5, 8}
	if len(got) != len(want) {
		t.Fatalf("got %d points, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].value != w {
			t.Errorf("value[%d] = %v, want %v", i, got[i].value, w)
		}
	}
	// The series must stay sorted ascending by time for datapointSeries.
	for i := 1; i < len(got); i++ {
		if got[i].timestamp < got[i-1].timestamp {
			t.Errorf("processed not ascending at %d: %d < %d", i, got[i].timestamp, got[i-1].timestamp)
		}
	}
}

func TestProcessDatapointsExcludesPointsAfterWindowEnd(t *testing.T) {
	// A datapoint logged later the same day, after a mid-day endTime, must not
	// leak into that day's aggregate (it's effectively in the future relative to
	// the charted window). Mirrors Beeminder filtering data to "now" before
	// aggregating.
	loc := time.UTC
	day := time.Date(2024, 7, 1, 0, 0, 0, 0, loc)
	start := day
	end := day.Add(15 * time.Hour) // window ends at 3pm

	goal := Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: day.Add(10 * time.Hour).Unix(), Daystamp: "20240701", Value: 5},  // before end → counts
			{Timestamp: day.Add(20 * time.Hour).Unix(), Daystamp: "20240701", Value: 99}, // after end → excluded
		},
	}
	got := processDatapoints(goal, start, end)

	// anchor 0 + the day's aggregate of just the 5 (the 99 is excluded), not 104.
	want := []float64{0, 5}
	if len(got) != len(want) {
		t.Fatalf("got %d points, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].value != w {
			t.Errorf("value[%d] = %v, want %v (point after endTime must be excluded)", i, got[i].value, w)
		}
	}
}

func TestProcessDatapointsAggregatesSameDay(t *testing.T) {
	// The crux of the aggday work: multiple datapoints on one day collapse to a
	// single plotted point per day, using the goal's aggday.
	start := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 3)
	noon := func(d int) int64 {
		return time.Date(2024, 3, 1+d, 12, 0, 0, 0, time.UTC).Unix()
	}

	// A cumulative goal whose default aggday is "sum": two same-day points sum
	// within the day, then accumulate across days.
	kyoom := processDatapoints(Goal{
		Kyoom: true,
		Datapoints: []Datapoint{
			{Timestamp: noon(0), Daystamp: "20240301", Value: 1},
			{Timestamp: noon(0) + 60, Daystamp: "20240301", Value: 2}, // same day → sums to 3
			{Timestamp: noon(1), Daystamp: "20240302", Value: 4},      // running 3 → 7
		},
	}, start, end)
	// anchor 0, day1 cumulative 3, day2 cumulative 7
	wantK := []float64{0, 3, 7}
	if len(kyoom) != len(wantK) {
		t.Fatalf("kyoom: got %d points, want %d (%v)", len(kyoom), len(wantK), kyoom)
	}
	for i, w := range wantK {
		if kyoom[i].value != w {
			t.Errorf("kyoom value[%d] = %v, want %v", i, kyoom[i].value, w)
		}
	}
	// Both day-1 points must collapse to one column (the day boundary), not two.
	if kyoom[1].timestamp != kyoom[0].timestamp {
		// anchor and day1 both sit at the window start (2024-03-01).
		t.Errorf("expected the day-1 point at the window-start day boundary, got %d vs anchor %d", kyoom[1].timestamp, kyoom[0].timestamp)
	}

	// A non-cumulative goal with an explicit aggday="max": the day's value is the
	// largest datapoint, with no accumulation.
	maxGoal := processDatapoints(Goal{
		Aggday: "max",
		Datapoints: []Datapoint{
			{Timestamp: noon(0), Daystamp: "20240301", Value: 1},
			{Timestamp: noon(0) + 60, Daystamp: "20240301", Value: 9},
			{Timestamp: noon(0) + 120, Daystamp: "20240301", Value: 4},
		},
	}, start, end)
	if len(maxGoal) != 1 || maxGoal[0].value != 9 {
		t.Errorf("aggday=max: want a single day valued 9, got %v", maxGoal)
	}
}

func TestDatapointSeriesStep(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 10)

	// Step-after: the first value is held flat across the gap and the line jumps
	// to the second value only at its column — never a linear ramp between them.
	// Matches Beeminder's steppy line, which is not interpolated.
	got, _ := datapointSeries([]timedValue{
		{timestamp: start.Unix(), value: 0},
		{timestamp: end.Unix(), value: 100},
	}, start, end, 11)
	if len(got) != 11 {
		t.Fatalf("expected 11 columns, got %d", len(got))
	}
	if got[0] != 0 || got[10] != 100 {
		t.Errorf("endpoints: got[0]=%v got[10]=%v, want 0 and 100", got[0], got[10])
	}
	for i := 1; i < 10; i++ {
		if got[i] != 0 {
			t.Errorf("step hold: col %d = %v, want 0 (held until the jump, not interpolated)", i, got[i])
		}
	}

	// A single datapoint fills the whole row with its value (no gaps, no NaN).
	single, _ := datapointSeries([]timedValue{
		{timestamp: start.AddDate(0, 0, 5).Unix(), value: 7},
	}, start, end, 11)
	for i, v := range single {
		if v != 7 {
			t.Errorf("single datapoint flat-fill: col %d = %v, want 7", i, v)
		}
	}
}

// TestDatapointSeriesCumulativeSteps guards the original integrations-goal case:
// a kyoom goal's line must step (hold the previous total, then jump at the
// datapoint), not draw a diagonal ramp between points — Beeminder's vertical
// riser. Stepping is universal (see TestDatapointSeriesStep), but a running
// total exercises it on accumulated values.
func TestDatapointSeriesCumulativeSteps(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 10)

	// Anchor of 0 at the window start, then a cumulative total of 1 at the
	// midpoint. Columns 0..mid-1 must stay flat at 0 (no ramp), the jump lands at
	// the midpoint column, and everything after holds 1.
	got, _ := datapointSeries([]timedValue{
		{timestamp: start.Unix(), value: 0},
		{timestamp: start.AddDate(0, 0, 5).Unix(), value: 1},
	}, start, end, 11)
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

// kyoomDailyGoal builds a cumulative do-more goal with one +1 datapoint per day
// over the last `days` days, its bright red line rising alongside. Used to flip
// between a sparse chart (few days → nodes read as separate steps) and a dense
// one (many days → nodes fill nearly every column).
func kyoomDailyGoal(days int) Goal {
	now := time.Now()
	start := now.AddDate(0, 0, -(days - 1))
	dps := make([]Datapoint, days)
	for i := 0; i < days; i++ {
		dps[i] = Datapoint{Timestamp: start.AddDate(0, 0, i).Unix(), Value: 1.0}
	}
	return Goal{
		Slug:       "daily",
		Yaw:        1,
		Kyoom:      true,
		Datapoints: dps,
		Tmin:       start.Format("2006-01-02"),
		Tmax:       now.Format("2006-01-02"),
		Roadall: [][]*float64{
			roadallRow(float64(start.Unix()), fptr(0.0), nil),
			roadallRow(float64(now.Unix()), fptr(float64(days)), nil),
		},
	}
}

// TestGoalChartMarkers covers the sparse/dense gate: a short-history chart dots
// each datapoint node on the blue line (mirroring Beeminder's graph), while a
// long-history chart — where nodes fill nearly every column and dots would just
// smear the line — draws none. It also guards that the dot lands on the blue
// line rather than floating in empty space, which would mean the asciigraph
// value→row projection has drifted.
func TestGoalChartMarkers(t *testing.T) {
	const width = 80
	goal := kyoomDailyGoal(6)
	sparse := renderGoalChart(goal, width)

	// Every datapoint node must be dotted, not just "at least one": a drifted
	// projection that lands markers on spaces (silently dropped by
	// replaceCellGlyph) would still leave a stray one, so assert the exact count.
	// Derive the expected node count from the same pipeline renderGoalChart uses.
	start, end := chartTimeframe(goal, time.Now())
	_, nodes := datapointSeries(processDatapoints(goal, start, end), start, end, width-10)
	if got := strings.Count(sparse, string(markerGlyph)); got != len(nodes) {
		t.Errorf("marker count = %d, want one per node (%d):\n%s", got, len(nodes), sparse)
	}

	// Each marker must sit ON the blue line: the SGR immediately governing the
	// marker cell (the last colour code before it on its row) must be blue. A
	// projection off by a row could land a dot on the red line while blue merely
	// appears elsewhere on the row — the weaker "blue somewhere before" check
	// would miss that.
	blue := "\x1b[94m"
	for _, ln := range strings.Split(sparse, "\n") {
		idx := strings.IndexRune(ln, markerGlyph)
		if idx < 0 {
			continue
		}
		sgrs := ansiPattern.FindAllString(ln[:idx], -1)
		if len(sgrs) == 0 || sgrs[len(sgrs)-1] != blue {
			t.Errorf("marker not governed by the blue SGR (last code before it = %v): %q", sgrs, ln)
		}
	}

	dense := renderGoalChart(kyoomDailyGoal(200), width)
	if strings.ContainsRune(dense, markerGlyph) {
		t.Errorf("dense chart should not dot datapoints (nodes fill the width), got:\n%s", dense)
	}
}

// TestGoalChartMarkerRiser guards that the vertical riser runs straight into each
// dot: on this strictly-rising staircase the marker sits on the step's corner, so
// the cell directly below it is a riser or bottom corner (│ ╯ ╰), not the tread
// beside a corner. Without the corner-column shift the dot would sit one column
// right (on the tread) with only empty space beneath it.
func TestGoalChartMarkerRiser(t *testing.T) {
	plain := ansiPattern.ReplaceAllString(renderGoalChart(kyoomDailyGoal(6), 60), "")
	grid := strings.Split(plain, "\n")

	connected := 0
	total := 0
	for r, ln := range grid {
		for c, ch := range []rune(ln) {
			if ch != markerGlyph {
				continue
			}
			total++
			// Look at the cell directly below (same visible column, next row).
			if r+1 < len(grid) {
				below := []rune(grid[r+1])
				if c < len(below) && strings.ContainsRune("│╯╰", below[c]) {
					connected++
				}
			}
		}
	}
	// Every marker but the first (which sits at the origin with nothing beneath)
	// must have the riser leading into it.
	if total == 0 || connected < total-1 {
		t.Errorf("riser leads into %d/%d markers, want >= %d:\n%s", connected, total, total-1, plain)
	}
}

func TestReplaceCellGlyph(t *testing.T) {
	// Colour runs must survive: only the rune at the target visible column
	// changes; SGR escapes (which occupy no column) stay put.
	line := "ab\x1b[94mcd\x1b[0mef"
	got := replaceCellGlyph(line, 3, '●') // visible cols: a0 b1 c2 d3 e4 f5
	want := "ab\x1b[94mc●\x1b[0mef"
	if got != want {
		t.Errorf("replaceCellGlyph = %q, want %q", got, want)
	}
	// A space at the target is left alone — a marker never floats off the line.
	if got := replaceCellGlyph("a c", 1, '●'); got != "a c" {
		t.Errorf("replaceCellGlyph over a space = %q, want unchanged", got)
	}
}

func TestAsciiRound(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{{0.4, 0}, {0.5, 1}, {1.5, 2}, {-0.5, -1}, {-1.4, -1}, {2.0, 2}}
	for _, c := range cases {
		if got := asciiRound(c.in); got != c.want {
			t.Errorf("asciiRound(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestRoadStepAlignsWithSameDayDatapoint guards the derailment picture: a
// derailment writes a vertical road step at a mid-day knot time (Beeminder
// stores knot times at the goal deadline's time-of-day) plus a datapoint whose
// daystamp is that same day. Beeminder day-snaps both onto one day grid so the
// two risers overlap; buzz must land the road's value change in the same
// column as the datapoint node (daysnapRoad + right-edge sampling), or the
// chart draws two separate vertical lines a few columns apart.
func TestRoadStepAlignsWithSameDayDatapoint(t *testing.T) {
	loc := time.Local
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 10)

	stepDay := start.AddDate(0, 0, 7)
	stepAt := float64(stepDay.Add(11 * time.Hour).Unix()) // mid-day derail knot

	r, err := parseRoad([][]*float64{
		roadallRow(float64(start.Unix()), fptr(30), nil),
		roadallRow(stepAt, nil, fptr(30)),
		roadallRow(stepAt, fptr(10091), nil), // vertical derail step
		roadallRow(float64(end.AddDate(0, 0, 300).Unix()), nil, fptr(30)),
	}, "d")
	if err != nil || len(r) == 0 {
		t.Fatalf("derail road parse: err=%v len=%d", err, len(r))
	}
	r = daysnapRoad(r, loc)

	for _, width := range []int{70, 71} { // 71: step day lands exactly on a column instant
		roadValues := roadValuesForTimeframe(r, start, end, width)

		// The datapoint the derailment wrote, bucketed to its daystamp's midnight.
		processed := []timedValue{
			{timestamp: start.Unix(), value: 0},
			{timestamp: stepDay.Unix(), value: 9881},
		}
		_, nodes := datapointSeries(processed, start, end, width)
		nodeCol := nodes[len(nodes)-1]

		stepCol := slices.IndexFunc(roadValues, func(v float64) bool { return v > 5000 })
		if stepCol != nodeCol {
			t.Errorf("width %d: road step first shows in column %d, datapoint node in column %d — risers won't overlap", width, stepCol, nodeCol)
		}
	}
}

// TestDaysnapRoad pins the properties daysnapRoad's doc comment claims:
// boundaries floor to local midnight, a vertical step's equal boundaries stay
// equal, segments stay contiguous and ordered, a sub-day sloped segment
// collapses to a zero-duration step, and slopePerDay is recomputed from the
// snapped boundaries (0 for zero-duration, per parseRoad's convention).
func TestDaysnapRoad(t *testing.T) {
	loc := time.Local
	day := func(d int, hour int) float64 {
		return float64(time.Date(2026, 7, 1+d, hour, 0, 0, 0, loc).Unix())
	}
	midnight := func(d int) float64 { return day(d, 0) }

	r := road{
		// sloped, mid-day boundaries spanning days 0..2
		{startT: day(0, 11), startV: 0, endV: 4, endT: day(2, 11), slopePerDay: 2},
		// vertical step at a mid-day instant
		{startT: day(2, 11), startV: 4, endV: 100, endT: day(2, 11), slopePerDay: 0},
		// sub-day sloped segment: 11:00 → 20:00 same day
		{startT: day(2, 11), startV: 100, endV: 103, endT: day(2, 20), slopePerDay: 8},
	}
	s := daysnapRoad(r, loc)

	for i, want := range []struct{ startT, endT, slope float64 }{
		{midnight(0), midnight(2), 2},
		{midnight(2), midnight(2), 0},
		{midnight(2), midnight(2), 0}, // sub-day segment collapsed to a step
	} {
		if s[i].startT != want.startT || s[i].endT != want.endT {
			t.Errorf("seg %d boundaries = (%f, %f), want (%f, %f)", i, s[i].startT, s[i].endT, want.startT, want.endT)
		}
		if math.Abs(s[i].slopePerDay-want.slope) > 1e-9 {
			t.Errorf("seg %d slopePerDay = %f, want %f (recomputed from snapped boundaries)", i, s[i].slopePerDay, want.slope)
		}
	}
	// Contiguity preserved across the chain.
	for i := 1; i < len(s); i++ {
		if s[i].startT != s[i-1].endT {
			t.Errorf("seg %d not contiguous: startT %f != prev endT %f", i, s[i].startT, s[i-1].endT)
		}
	}
}
