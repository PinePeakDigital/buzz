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
// moment, expressed in the goal's gunits per day, and a bool reporting whether
// a slope is available.
//
// For goals with a piecewise schedule (rate changes scheduled in the future),
// g.Rate reports the rate at the *end* of the graph rather than the rate of
// the segment the user is currently on — so a "1 h/day now, dropping to
// 0.1 h/day later" goal would otherwise be bumped by the wrong amount. This is
// the policy wrapper over the parsed bright red line (road.go): resolve the
// segment containing t from the road and use its slope; fall back to g.Rate
// when the road is absent, malformed, or t is outside its span.
func slopePerDayAt(g Goal, t time.Time) (float64, bool) {
	// A malformed road falls back to g.Rate here rather than erroring — the
	// chart (renderGoalChart) is where a malformed road is surfaced loudly;
	// this slope path stays tolerant so a single bad goal doesn't break the
	// tomorrow view's bump for every other goal.
	if road, err := parseRoad(g.Roadall, g.Runits); err == nil && len(road) > 0 {
		if slope, ok := road.slopePerDayAt(t); ok {
			return slope, true
		}
	}
	if g.Rate == nil || !isKnownRunits(g.Runits) {
		return 0, false
	}
	return ratePerDay(*g.Rate, g.Runits), true
}
