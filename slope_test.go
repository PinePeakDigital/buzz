package main

import (
	"math"
	"testing"
	"time"
)

// TestSlopePerDayAt covers the Goal-level policy wrapper (not road.slopePerDayAt,
// which road_test.go pins). The wrapper resolves the segment from the parsed
// road and falls back to g.Rate when the road is absent, malformed, or t is
// outside its span — and reports unavailable only when there's no usable rate.
func TestSlopePerDayAt(t *testing.T) {
	// A "0 then 1/day for 10 days" road, anchored at roadBase (see road_test.go).
	inSpanRoad := [][]*float64{
		roadallRow(roadUnix(0), fptr(0), nil),
		roadallRow(roadUnix(10), nil, fptr(1)),
	}
	// A malformed road: a non-anchor row carrying both value and rate.
	malformedRoad := [][]*float64{
		roadallRow(roadUnix(0), fptr(0), nil),
		roadallRow(roadUnix(10), fptr(10), fptr(1)),
	}

	cases := []struct {
		name     string
		goal     Goal
		t        time.Time
		wantOK   bool
		wantRate float64 // checked only when wantOK
	}{
		{
			name:     "in-span road wins over g.Rate",
			goal:     Goal{Roadall: inSpanRoad, Runits: "d", Rate: fptr(99)},
			t:        roadDay(5),
			wantOK:   true,
			wantRate: 1.0,
		},
		{
			name:     "malformed road falls back to g.Rate (does not error)",
			goal:     Goal{Roadall: malformedRoad, Runits: "d", Rate: fptr(2)},
			t:        roadDay(5),
			wantOK:   true,
			wantRate: 2.0,
		},
		{
			name:     "t past the road's end falls back to g.Rate",
			goal:     Goal{Roadall: inSpanRoad, Runits: "d", Rate: fptr(2)},
			t:        roadDay(50),
			wantOK:   true,
			wantRate: 2.0,
		},
		{
			name:     "absent road falls back to g.Rate",
			goal:     Goal{Runits: "d", Rate: fptr(3)},
			t:        roadDay(5),
			wantOK:   true,
			wantRate: 3.0,
		},
		{
			name:     "weekly g.Rate fallback converts to per-day",
			goal:     Goal{Runits: "w", Rate: fptr(7)},
			t:        roadDay(5),
			wantOK:   true,
			wantRate: 1.0,
		},
		{
			name:   "nil g.Rate with no usable road is unavailable",
			goal:   Goal{Runits: "d"},
			t:      roadDay(5),
			wantOK: false,
		},
		{
			name:   "unknown runits with no usable road is unavailable",
			goal:   Goal{Runits: "lightyears", Rate: fptr(5)},
			t:      roadDay(5),
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := slopePerDayAt(tc.goal, tc.t)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if tc.wantOK && math.Abs(got-tc.wantRate) > 1e-9 {
				t.Errorf("slope = %v, want %v", got, tc.wantRate)
			}
		})
	}
}
