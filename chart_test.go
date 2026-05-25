package main

import (
	"strings"
	"testing"
	"time"
)

// roadall builds a Beeminder-style roadall row. Pass nil for v or r to leave
// that column unset (Beeminder rows past the anchor have exactly one of v/r).
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

	chart := renderGoalChart(goal, 80)
	if chart != "" {
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
			{
				Timestamp: yesterday.Unix(),
				Value:     5.0,
			},
			{
				Timestamp: now.Unix(),
				Value:     10.0,
			},
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
		t.Error("Expected non-empty chart for goal with datapoints")
	}

	// Check for key elements in the chart
	if !strings.Contains(chart, "Goal Progress Chart") {
		t.Error("Expected chart to contain 'Goal Progress Chart'")
	}
	if !strings.Contains(chart, "Do More") {
		t.Error("Expected chart to contain 'Do More'")
	}
	// asciigraph uses its own caption format
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
			{
				Timestamp: yesterday.Unix(),
				Value:     5.0,
			},
			{
				Timestamp: now.Unix(),
				Value:     3.0,
			},
		},
		Tmin: yesterday.Format("2006-01-02"),
		Tmax: now.Format("2006-01-02"),
	}

	chart := renderGoalChart(goal, 80)
	if chart == "" {
		t.Error("Expected non-empty chart for cumulative goal")
	}

	// Check that cumulative is mentioned
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
			{
				Timestamp: yesterday.Unix(),
				Value:     10.0,
			},
			{
				Timestamp: now.Unix(),
				Value:     5.0,
			},
		},
		Tmin: yesterday.Format("2006-01-02"),
		Tmax: now.Format("2006-01-02"),
	}

	chart := renderGoalChart(goal, 80)
	if chart == "" {
		t.Error("Expected non-empty chart for do less goal")
	}

	// Check that Do Less is mentioned
	if !strings.Contains(chart, "Do Less") {
		t.Error("Expected chart to indicate 'Do Less' goal")
	}
}

func TestRenderGoalChartWithFallbackTimeframe(t *testing.T) {
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	goal := Goal{
		Slug: "test-goal",
		Yaw:  1,
		Datapoints: []Datapoint{
			{
				Timestamp: thirtyDaysAgo.Unix(),
				Value:     5.0,
			},
			{
				Timestamp: now.Unix(),
				Value:     10.0,
			},
		},
		// No Tmin/Tmax - should use fallback
	}

	chart := renderGoalChart(goal, 80)
	if chart == "" {
		t.Error("Expected non-empty chart even without tmin/tmax")
	}
}

func TestGetRoadValueAtTime(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Anchor at v=0, then a rate-only segment 10 days later. The chart's
	// interpolator should resolve to v≈5 at day 5 (5 days at 1/day, with
	// d runits).
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), nil, fptr(1.0)),
		},
	}

	testTime := baseTime.AddDate(0, 0, 5)
	value := getRoadValueAtTime(goal, testTime)
	if value < 4.9 || value > 5.1 {
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
		t.Errorf("Expected 11 values, got %d", len(values))
	}

	// First value should be around 0
	if values[0] < -0.5 || values[0] > 0.5 {
		t.Errorf("Expected first value around 0, got %f", values[0])
	}

	// Last value should be around 10
	if values[10] < 9.5 || values[10] > 10.5 {
		t.Errorf("Expected last value around 10, got %f", values[10])
	}
}

func TestGetRoadValueAtTimeShortRoad(t *testing.T) {
	// Fewer than 2 rows is unusable — the function must short-circuit to 0
	// rather than deref Roadall[0].
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
	// Querying 20 days in should return the last materialised value (10).
	got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, 20))
	if got < 9.9 || got > 10.1 {
		t.Errorf("past end of road: expected ~10, got %f", got)
	}
}

func TestGetRoadValueAtTimeBeforeAnchor(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Anchor at v=0; rate-only segment 10 days later at 1/day.
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), nil, fptr(1.0)),
		},
	}
	// Five days before the anchor → extrapolate at -1/day → ~-5.
	got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, -5))
	if got < -5.1 || got > -4.9 {
		t.Errorf("before anchor extrapolation: expected ~-5, got %f", got)
	}
}

func TestGetRoadValueAtTimeUnknownRunits(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Rate-only segment with unrecognised runits → the function must bail
	// rather than treat the rate as gunits/day.
	goal := Goal{
		Runits: "lightyears",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), nil, fptr(1.0)),
		},
	}
	// Should return prevV (0) at any point past the anchor, not the
	// dimensionally-wrong extrapolation.
	got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, 5))
	if got != 0 {
		t.Errorf("unknown runits: expected 0 (bail), got %f", got)
	}
}

func TestGetRoadValueAtTimeAmbiguousRow(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Non-anchor row with both v and r set is ambiguous per Beeminder
	// spec; the walker must bail to the prior anchor (0) rather than
	// silently pick one interpretation.
	goal := Goal{
		Runits: "d",
		Roadall: [][]*float64{
			roadallRow(float64(baseTime.Unix()), fptr(0.0), nil),
			roadallRow(float64(baseTime.AddDate(0, 0, 10).Unix()), fptr(10.0), fptr(1.0)),
		},
	}
	got := getRoadValueAtTime(goal, baseTime.AddDate(0, 0, 5))
	if got != 0 {
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
	// numPoints==1 used to divide by (numPoints-1) and produce NaN; the
	// guard returns a single sample at startTime instead.
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

	goal := Goal{
		Roadall: [][]*float64{}, // No road data
	}

	values := getRoadValuesForTimeframe(goal, baseTime, endTime, 10)
	if len(values) != 10 {
		t.Errorf("Expected 10 values, got %d", len(values))
	}

	// All values should be 0
	for i, v := range values {
		if v != 0 {
			t.Errorf("Expected value at index %d to be 0, got %f", i, v)
		}
	}
}
