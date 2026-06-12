package main

import (
	"math"
	"testing"
	"time"
)

var roadBase = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func roadDay(d int) time.Time { return roadBase.AddDate(0, 0, d) }
func roadUnix(d int) float64  { return float64(roadDay(d).Unix()) }

// validRoad is a "0 then 1/day for 10 days" bright red line: anchor at value 0,
// a rate row climbing 1 gunit/day, ending at day 10 (value 10).
func validRoad() [][]*float64 {
	return [][]*float64{
		roadallRow(roadUnix(0), fptr(0), nil),
		roadallRow(roadUnix(10), nil, fptr(1)),
	}
}

func TestParseRoadAbsent(t *testing.T) {
	cases := []struct {
		name    string
		roadall [][]*float64
	}{
		{"nil", nil},
		{"empty", [][]*float64{}},
		{"anchor only", [][]*float64{roadallRow(roadUnix(0), fptr(5), nil)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := parseRoad(tc.roadall, "d")
			if err != nil {
				t.Fatalf("absent road must not error, got %v", err)
			}
			if len(r) != 0 {
				t.Errorf("absent road must be empty, got %d segments", len(r))
			}
		})
	}
}

func TestParseRoadMalformed(t *testing.T) {
	cases := []struct {
		name    string
		runits  string
		roadall [][]*float64
	}{
		{"row with both value and rate set", "d", [][]*float64{
			roadallRow(roadUnix(0), fptr(0), nil),
			roadallRow(roadUnix(10), fptr(10), fptr(1)),
		}},
		{"row with neither value nor rate", "d", [][]*float64{
			roadallRow(roadUnix(0), fptr(0), nil),
			roadallRow(roadUnix(10), nil, nil),
		}},
		{"anchor missing value", "d", [][]*float64{
			roadallRow(roadUnix(0), nil, fptr(1)),
			roadallRow(roadUnix(10), nil, fptr(1)),
		}},
		{"anchor with rate set", "d", [][]*float64{
			roadallRow(roadUnix(0), fptr(0), fptr(1)),
			roadallRow(roadUnix(10), nil, fptr(1)),
		}},
		{"short row", "d", [][]*float64{
			roadallRow(roadUnix(0), fptr(0), nil),
			{fptr(roadUnix(10))},
		}},
		{"row with nil time", "d", [][]*float64{
			roadallRow(roadUnix(0), fptr(0), nil),
			{nil, nil, fptr(1)},
		}},
		{"rate row with unknown runits", "lightyears", [][]*float64{
			roadallRow(roadUnix(0), fptr(0), nil),
			roadallRow(roadUnix(10), nil, fptr(1)),
		}},
		{"non-increasing time (equal to previous)", "d", [][]*float64{
			roadallRow(roadUnix(0), fptr(0), nil),
			roadallRow(roadUnix(0), fptr(5), nil),
		}},
		{"non-increasing time (earlier than previous)", "d", [][]*float64{
			roadallRow(roadUnix(10), fptr(0), nil),
			roadallRow(roadUnix(5), nil, fptr(1)),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseRoad(tc.roadall, tc.runits); err == nil {
				t.Errorf("malformed road must error, got nil")
			}
		})
	}
}

func TestRoadValueAt(t *testing.T) {
	r, err := parseRoad(validRoad(), "d")
	if err != nil || len(r) == 0 {
		t.Fatalf("validRoad parse: err=%v len=%d", err, len(r))
	}

	if got := r.valueAt(roadDay(5)); got < 4.9 || got > 5.1 {
		t.Errorf("valueAt(day 5) = %f, want ~5", got)
	}
	// Before the anchor: extrapolate backward along the first segment (slope
	// +1/day); the value goes negative only because the time delta is negative.
	if got := r.valueAt(roadDay(-5)); got < -5.1 || got > -4.9 {
		t.Errorf("valueAt(day -5) = %f, want ~-5", got)
	}
	// Past the end: hold the last value (10).
	if got := r.valueAt(roadDay(20)); got < 9.9 || got > 10.1 {
		t.Errorf("valueAt(day 20) = %f, want ~10", got)
	}
}

func TestRoadValuesForTimeframe(t *testing.T) {
	r, err := parseRoad(validRoad(), "d")
	if err != nil || len(r) == 0 {
		t.Fatalf("validRoad parse: err=%v len=%d", err, len(r))
	}

	values := roadValuesForTimeframe(r, roadDay(0), roadDay(10), 11)
	if len(values) != 11 {
		t.Fatalf("want 11 values, got %d", len(values))
	}
	if values[0] < -0.5 || values[0] > 0.5 {
		t.Errorf("first sample = %f, want ~0", values[0])
	}
	if values[10] < 9.5 || values[10] > 10.5 {
		t.Errorf("last sample = %f, want ~10", values[10])
	}

	// numPoints == 1 must not divide by (numPoints-1): one sample at startTime.
	single := roadValuesForTimeframe(r, roadDay(5), roadDay(10), 1)
	if len(single) != 1 || single[0] < 4.9 || single[0] > 5.1 {
		t.Errorf("single-point sample = %v, want [~5]", single)
	}
}

func TestRoadSlopePerDayAt(t *testing.T) {
	// Two rate segments: 1/day until day 10, then 0.1/day until day 20.
	twoSeg, _ := parseRoad([][]*float64{
		roadallRow(roadUnix(0), fptr(0), nil),
		roadallRow(roadUnix(10), nil, fptr(1)),
		roadallRow(roadUnix(20), nil, fptr(0.1)),
	}, "d")

	if got, ok := twoSeg.slopePerDayAt(roadDay(5)); !ok || math.Abs(got-1.0) > 1e-9 {
		t.Errorf("slope in first segment = %v,%v, want 1,true", got, ok)
	}
	if got, ok := twoSeg.slopePerDayAt(roadDay(15)); !ok || math.Abs(got-0.1) > 1e-9 {
		t.Errorf("slope in second segment = %v,%v, want 0.1,true", got, ok)
	}
	// Outside the span: no slope (caller falls back to g.Rate).
	if _, ok := twoSeg.slopePerDayAt(roadDay(-1)); ok {
		t.Errorf("slope before start should be unavailable")
	}
	if _, ok := twoSeg.slopePerDayAt(roadDay(21)); ok {
		t.Errorf("slope past end should be unavailable")
	}

	// Weekly runits convert to per-day: 7/week == 1/day.
	weekly, _ := parseRoad([][]*float64{
		roadallRow(roadUnix(0), fptr(0), nil),
		roadallRow(roadUnix(10), nil, fptr(7)),
	}, "w")
	if got, ok := weekly.slopePerDayAt(roadDay(5)); !ok || math.Abs(got-1.0) > 1e-9 {
		t.Errorf("weekly slope = %v,%v, want 1,true", got, ok)
	}

	// Value-only segment: slope derived from Δvalue/Δtime.
	valueSeg, _ := parseRoad([][]*float64{
		roadallRow(roadUnix(0), fptr(0), nil),
		roadallRow(roadUnix(10), fptr(30), nil),
	}, "d")
	if got, ok := valueSeg.slopePerDayAt(roadDay(5)); !ok || math.Abs(got-3.0) > 1e-9 {
		t.Errorf("value-segment slope = %v,%v, want 3,true", got, ok)
	}
}

// TestRoadSlopePerDayAtClosesValueAfterRateGap pins the behavior the old split
// walkers couldn't: a value row that follows a rate row. The rate row's end
// value is materialised, so the value row's slope resolves from Δvalue/Δtime
// instead of bailing to g.Rate.
func TestRoadSlopePerDayAtClosesValueAfterRateGap(t *testing.T) {
	// anchor 0 → rate 1/day to day 10 (value 10) → value 25 at day 20.
	// The value segment runs 10→25 over 10 days = 1.5/day.
	r, err := parseRoad([][]*float64{
		roadallRow(roadUnix(0), fptr(0), nil),
		roadallRow(roadUnix(10), nil, fptr(1)),
		roadallRow(roadUnix(20), fptr(25), nil),
	}, "d")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, ok := r.slopePerDayAt(roadDay(15)); !ok || math.Abs(got-1.5) > 1e-9 {
		t.Errorf("value-after-rate slope = %v,%v, want 1.5,true", got, ok)
	}
}
