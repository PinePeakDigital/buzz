package main

import (
	"strings"
	"testing"
	"time"
)

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
		Roadall: [][]any{
			{float64(yesterday.Unix()), 0.0, 5.0},
			{float64(now.Unix()), 5.0, 5.0},
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
	goal := Goal{
		Roadall: [][]any{
			{float64(baseTime.Unix()), 0.0, 1.0}, // Start at 0, rate 1/day
			{float64(baseTime.AddDate(0, 0, 10).Unix()), 10.0, 1.0},
		},
	}

	// Test at day 5
	testTime := baseTime.AddDate(0, 0, 5)
	value := getRoadValueAtTime(goal, testTime)
	if value < 4.9 || value > 5.1 {
		t.Errorf("Expected value around 5.0, got %f", value)
	}
}

func TestGetRoadValueAtTimeWithDateStrings(t *testing.T) {
	goal := Goal{
		Roadall: [][]any{
			{"2024-01-01", 0.0, 1.0},
			{"2024-01-11", 10.0, 1.0},
		},
	}

	testTime := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)
	value := getRoadValueAtTime(goal, testTime)
	// Should be around 5.0 (5 days * 1/day rate)
	if value < 4.0 || value > 6.0 {
		t.Errorf("Expected value around 5.0, got %f", value)
	}
}

func TestGetRoadValuesForTimeframe(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := baseTime.AddDate(0, 0, 10)

	goal := Goal{
		Roadall: [][]any{
			{float64(baseTime.Unix()), 0.0, 1.0},
			{float64(endTime.Unix()), 10.0, 1.0},
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

func TestGetRoadValuesForTimeframeEmpty(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := baseTime.AddDate(0, 0, 10)

	goal := Goal{
		Roadall: [][]any{}, // No road data
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
