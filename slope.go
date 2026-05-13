package main

import "time"

// Bright-line slope geometry. The "slope" of a goal is how fast its bright red
// line is moving in gunits/day at a given moment. For a flat-rate goal that's
// just `g.Rate` normalised to per-day; for a piecewise goal it's the rate of
// the segment containing `now`, which is what `roadall` encodes.
//
// `g.Rate` reports the rate at the *end* of the graph, not at any specific
// moment, so for piecewise schedules it gives the wrong answer when used as
// "today's slope". `slopePerDayAt` resolves the correct segment from
// `Roadall` and falls back to `g.Rate` only when roadall isn't usable.

// isKnownRunits reports whether the given runits string is one of the values
// Beeminder uses (y/m/w/d/h). Callers that need a dimensionally-correct
// per-day conversion should bail out for anything else.
func isKnownRunits(runits string) bool {
	switch runits {
	case "y", "m", "w", "d", "h":
		return true
	}
	return false
}

// ratePerDay converts a rate expressed in the given runits into an equivalent
// amount per day. Supports the runits Beeminder reports: y, m, w, d, h. For
// unrecognised units the rate is returned unchanged.
func ratePerDay(rate float64, runits string) float64 {
	switch runits {
	case "y":
		return rate / 365.0
	case "m":
		return rate / 30.0
	case "w":
		return rate / 7.0
	case "d":
		return rate
	case "h":
		return rate * 24.0
	default:
		return rate
	}
}

// slopePerDayAt returns the slope of the goal's bright line at the given
// moment, expressed in the goal's gunits per day.
//
// For goals with a piecewise schedule (rate changes scheduled in the future),
// g.Rate reports the rate at the *end* of the graph rather than the rate of
// the segment the user is currently on — so a "1 h/day now, dropping to
// 0.1 h/day later" goal would otherwise be bumped by the wrong amount. Resolve
// the current segment from g.Roadall and use its slope; fall back to g.Rate
// only when roadall isn't usable.
func slopePerDayAt(g Goal, t time.Time) (float64, bool) {
	if slope, ok := roadallSlopePerDayAt(g, t); ok {
		return slope, true
	}
	if g.Rate == nil {
		return 0, false
	}
	if !isKnownRunits(g.Runits) {
		return 0, false
	}
	return ratePerDay(*g.Rate, g.Runits), true
}

// roadallSlopePerDayAt walks Beeminder's piecewise bright line (roadall) and
// returns the slope of the segment containing t, in gunits/day. Rows are
// [t, v, r] triples with exactly one of v/r null per row past the first
// anchor row. When the row's rate is given, it's converted via ratePerDay;
// when only values are given, the slope is computed directly from
// (Δvalue / Δtime) so the goal's runits don't matter.
//
// Returns ok=false for a missing/short roadall, unparseable rows, or when t
// is past the goal's end date (caller falls back to g.Rate in that case).
func roadallSlopePerDayAt(g Goal, t time.Time) (float64, bool) {
	if len(g.Roadall) < 2 {
		return 0, false
	}
	target := float64(t.Unix())
	for i := 1; i < len(g.Roadall); i++ {
		cur := g.Roadall[i]
		// A malformed boundary row makes the road ambiguous; fail fast so
		// the caller falls back to g.Rate rather than silently advancing
		// to a later segment that doesn't actually contain `target`.
		if len(cur) < 3 || cur[0] == nil {
			return 0, false
		}
		// Per Beeminder spec, non-anchor rows must have exactly one of
		// value/rate set. Both nil or both set means the row is ambiguous.
		if (cur[1] == nil) == (cur[2] == nil) {
			return 0, false
		}
		prev := g.Roadall[i-1]
		if len(prev) < 3 || prev[0] == nil {
			return 0, false
		}
		// Validate prev's shape: the first row is the start anchor (value
		// set, rate nil); subsequent rows follow the same one-of-v-or-r
		// constraint as cur.
		if i == 1 {
			if prev[1] == nil || prev[2] != nil {
				return 0, false
			}
		} else if (prev[1] == nil) == (prev[2] == nil) {
			return 0, false
		}
		// `target` is before the road even starts — don't fall through to
		// segment 1 just because target <= cur[0].
		if target < *prev[0] {
			return 0, false
		}
		if target > *cur[0] {
			continue
		}
		// `t` falls in the segment ending at this row.
		if cur[2] != nil {
			if !isKnownRunits(g.Runits) {
				return 0, false
			}
			return ratePerDay(*cur[2], g.Runits), true
		}
		// Rate not specified — derive from (Δvalue / Δtime).
		if prev[1] == nil || cur[1] == nil {
			return 0, false
		}
		seconds := *cur[0] - *prev[0]
		if seconds <= 0 {
			return 0, false
		}
		return (*cur[1] - *prev[1]) / seconds * 86400.0, true
	}
	return 0, false
}
