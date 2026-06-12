package main

import (
	"fmt"
	"time"
)

// The bright red line — Beeminder's commitment line, delivered as the `roadall`
// API field (rows of [t, value, rate]). parseRoad materialises that raw matrix
// into segments once; valueAt and slopePerDayAt then answer from the segments
// rather than re-walking the raw rows. See CONTEXT.md for vocabulary and
// docs/adr/0003-bright-red-line-parsing-failure-policy.md for the failure
// policy this implements.

// roadSegment is one piece of the bright red line between two boundaries, with
// both endpoint values materialised and the slope precomputed.
type roadSegment struct {
	startT, endT float64 // unix seconds
	startV, endV float64 // gunits
	slopePerDay  float64 // gunits/day
}

// road is a parsed bright red line: segments in ascending time. An empty road
// means the goal's roadall was absent — a benign "not populated" state, not a
// malformed one (see parseRoad).
type road []roadSegment

// parseRoad materialises a goal's roadall into segments. Three outcomes:
//
//   - (segments, nil) — a well-formed road.
//   - (nil, nil)      — absent roadall (fewer than 2 rows): a benign "not
//     populated" state. Callers fall back to g.Rate or show a message.
//   - (nil, error)    — present but malformed: almost certainly a bug in this
//     parser or upstream, surfaced loudly. Parsing is all-or-nothing: one bad
//     row fails the whole road.
//
// Validator: row 0 is the anchor [t, value, null]; every later row sets exactly
// one of value/rate. NOTE: docs/beeminder-api.md:457 claims roadall ends with
// an all-set [goaldate, goalval, rate] row, but a read-only audit of a live
// 60-goal account found zero such rows — every terminal row is a rate-row. The
// validator stays strict and treats an all-set row as an unobserved anomaly to
// surface; do NOT loosen it on the doc's authority (see ADR-0003).
func parseRoad(roadall [][]*float64, runits string) (road, error) {
	if len(roadall) < 2 {
		return nil, nil // absent — not an error
	}

	first := roadall[0]
	if len(first) < 3 || first[0] == nil || first[1] == nil || first[2] != nil {
		return nil, fmt.Errorf("road row 0: anchor must be [time, value, null]")
	}
	prevT, prevV := *first[0], *first[1]

	segs := make(road, 0, len(roadall)-1)
	for i := 1; i < len(roadall); i++ {
		cur := roadall[i]
		if len(cur) < 3 || cur[0] == nil {
			return nil, fmt.Errorf("road row %d: missing time", i)
		}
		// Per the Beeminder spec a non-anchor row sets exactly one of
		// value/rate. Both nil or both set is ambiguous.
		if (cur[1] == nil) == (cur[2] == nil) {
			return nil, fmt.Errorf("road row %d: must set exactly one of value or rate", i)
		}
		curT := *cur[0]
		// Times must strictly increase: an equal or earlier boundary would
		// produce a zero- or negative-duration segment, after which valueAt /
		// slopePerDayAt pick the wrong branch. Per ADR-0003 that's a malformed
		// road, surfaced rather than silently materialised.
		if curT <= prevT {
			return nil, fmt.Errorf("road row %d: time must be greater than the previous row time", i)
		}

		var curV, slopePerDay float64
		if cur[1] != nil {
			// Value row: slope derived from the materialised endpoints. curT >
			// prevT is guaranteed above, so the divisor is always positive.
			curV = *cur[1]
			slopePerDay = (curV - prevV) / (curT - prevT) * 86400.0
		} else {
			// Rate row: the slope is the row's rate (in gunits/day); the end
			// value is materialised from it so a following value-or-rate row
			// has a known anchor — this is what closes the value-after-rate
			// gap the old split walkers had.
			if !isKnownRunits(runits) {
				return nil, fmt.Errorf("road row %d: unknown runits %q for a rate row", i, runits)
			}
			slopePerDay = ratePerDay(*cur[2], runits)
			curV = prevV + slopePerDay/86400.0*(curT-prevT)
		}

		segs = append(segs, roadSegment{startT: prevT, endT: curT, startV: prevV, endV: curV, slopePerDay: slopePerDay})
		prevT, prevV = curT, curV
	}
	return segs, nil
}

// valueAt returns the bright red line's value at time t. It is defined for all
// t: interpolated within the road, extrapolated backward along the first
// segment before the start, and held flat at the last value past the end.
func (r road) valueAt(t time.Time) float64 {
	if len(r) == 0 {
		return 0
	}
	target := float64(t.Unix())

	first := r[0]
	if target <= first.startT {
		// Before the start: extrapolate backward along the first segment.
		return first.startV + first.slopePerDay/86400.0*(target-first.startT)
	}
	for _, seg := range r {
		if target <= seg.endT {
			if seg.endT == seg.startT {
				return seg.endV
			}
			frac := (target - seg.startT) / (seg.endT - seg.startT)
			return seg.startV + frac*(seg.endV-seg.startV)
		}
	}
	// Past the end: hold the last value.
	return r[len(r)-1].endV
}

// slopePerDayAt returns the slope (gunits/day) of the segment containing t, with
// ok=true only when t falls within the road's span [start, end]. Outside the
// span ok is false and the caller falls back to g.Rate (the bright line's slope
// is only defined where the road actually runs).
func (r road) slopePerDayAt(t time.Time) (float64, bool) {
	if len(r) == 0 {
		return 0, false
	}
	target := float64(t.Unix())
	if target < r[0].startT {
		return 0, false
	}
	for _, seg := range r {
		if target <= seg.endT {
			return seg.slopePerDay, true
		}
	}
	return 0, false // past the end
}
