package main

import (
	"math"
	"testing"
)

func TestResolveAggdayDefaults(t *testing.T) {
	if got := resolveAggday(Goal{Kyoom: true}); got != "sum" {
		t.Errorf("kyoom default: got %q, want \"sum\"", got)
	}
	if got := resolveAggday(Goal{Kyoom: false}); got != "last" {
		t.Errorf("non-kyoom default: got %q, want \"last\"", got)
	}
	if got := resolveAggday(Goal{Kyoom: true, Aggday: "max"}); got != "max" {
		t.Errorf("explicit aggday: got %q, want \"max\"", got)
	}
}

func TestAggregateDayMethods(t *testing.T) {
	vals := []float64{1, 2, 2, 5} // ascending-timestamp order

	cases := map[string]float64{
		"sum":       10,
		"last":      5,
		"first":     1,
		"min":       1,
		"max":       5,
		"count":     4,
		"truemean":  2.5,                  // mean of all
		"uniqmean":  float64(1+2+5) / 3.0, // mean of unique {1,2,5}
		"median":    2,                    // sorted {1,2,2,5} → (2+2)/2
		"mode":      2,
		"binary":    1,
		"nonzero":   1,
		"triangle":  55, // 10*11/2
		"square":    100,
		"cap1":      1,
		"sqrt":      math.Sqrt(10),
		"countflat": 4, // all nonzero
		"muflat":    2.5,
	}
	for name, want := range cases {
		if got := aggregateDay(Goal{}, name, vals); math.Abs(got-want) > 1e-9 {
			t.Errorf("aggday=%s: got %v, want %v", name, got, want)
		}
	}

	// median with an odd count returns the true middle of the sorted input
	// (inputs need not be pre-sorted).
	if got := aggregateDay(Goal{}, "median", []float64{5, 1, 3}); got != 3 {
		t.Errorf("median odd-count: got %v, want 3", got)
	}

	// trimmean drops the lowest/highest trim-fraction before averaging. With <10
	// values floor(0.1*n)=0, so it degenerates to the plain mean...
	if got := aggregateDay(Goal{}, "trimmean", vals); math.Abs(got-2.5) > 1e-9 {
		t.Errorf("trimmean small list (no trim): got %v, want 2.5", got)
	}
	// ...and with >=10 values the single lowest and highest are dropped: trimming
	// 1 and 100 from {1,2,3,4,5,6,7,8,9,100} leaves {2..9}, mean 5.5.
	big := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100}
	if got := aggregateDay(Goal{}, "trimmean", big); math.Abs(got-5.5) > 1e-9 {
		t.Errorf("trimmean trimming extremes: got %v, want 5.5", got)
	}

	// nonzero / countflat / muflat ignore zeros.
	withZeros := []float64{0, 0, 4}
	if got := aggregateDay(Goal{}, "nonzero", withZeros); got != 1 {
		t.Errorf("nonzero with a nonzero present: got %v, want 1", got)
	}
	if got := aggregateDay(Goal{}, "nonzero", []float64{0, 0}); got != 0 {
		t.Errorf("nonzero all-zero: got %v, want 0", got)
	}
	if got := aggregateDay(Goal{}, "countflat", withZeros); got != 1 {
		t.Errorf("countflat: got %v, want 1", got)
	}
	if got := aggregateDay(Goal{}, "muflat", withZeros); got != 4 {
		t.Errorf("muflat: got %v, want 4 (mean of nonzero)", got)
	}
}

func TestAggregateDayClocky(t *testing.T) {
	// Sum of differences of consecutive pairs; trailing unpaired value ignored.
	if got := aggClocky([]float64{1, 2, 6, 9}); got != 4 {
		t.Errorf("clocky [1,2,6,9]: got %v, want 4", got)
	}
	if got := aggClocky([]float64{1, 2, 6}); got != 1 {
		t.Errorf("clocky [1,2,6] (odd): got %v, want 1", got)
	}
}

func TestAggregateDaySkatesum(t *testing.T) {
	rate := 5.0
	g := Goal{Rate: &rate, Runits: "d"} // 5/day

	// Sum (1+2+3=6) capped at the daily rate (5).
	if got := aggregateDay(g, "skatesum", []float64{1, 2, 3}); got != 5 {
		t.Errorf("skatesum capped: got %v, want 5", got)
	}
	// Under the cap, the sum passes through.
	if got := aggregateDay(g, "skatesum", []float64{1, 2}); got != 3 {
		t.Errorf("skatesum under cap: got %v, want 3", got)
	}
	// Unusable rate → fall back to a plain sum rather than a wrong cap.
	if got := aggregateDay(Goal{}, "skatesum", []float64{1, 2, 3}); got != 6 {
		t.Errorf("skatesum no rate: got %v, want 6 (plain sum)", got)
	}
}

func TestAggregateDayUnknownFallsBackToDefault(t *testing.T) {
	// Unknown method renders with the goal's default (sum for kyoom, last else).
	if got := aggregateDay(Goal{Kyoom: true}, "bogus", []float64{1, 2, 3}); got != 6 {
		t.Errorf("unknown on kyoom: got %v, want 6 (sum)", got)
	}
	if got := aggregateDay(Goal{Kyoom: false}, "bogus", []float64{1, 2, 3}); got != 3 {
		t.Errorf("unknown on non-kyoom: got %v, want 3 (last)", got)
	}
}
