package main

import (
	"math"
	"sort"
	"time"
)

// aggday (aggregation method) decides how multiple datapoints landing on the
// same day combine into the single value plotted for that day. This mirrors
// Beeminder: each day's datapoints are reduced to one value via the goal's
// aggday, and then — for cumulative (kyoom) goals — those daily values are
// summed into a running total (see processDatapoints).
//
// The reducers below are ported from beebrain's broad.js AGGR map; the names
// (including Beeminder's legacy aliases) are matched exactly so any goal renders
// the same way buzz's chart does as on beeminder.com.

// resolveAggday returns the goal's aggday method, falling back to Beeminder's
// per-kyoom default when the goal carries none: "sum" for cumulative goals,
// "last" otherwise (beebrain.js: `p.aggday = gol.kyoom ? "sum" : "last"`).
func resolveAggday(goal Goal) string {
	if goal.Aggday != "" {
		return goal.Aggday
	}
	return defaultAggday(goal)
}

func defaultAggday(goal Goal) string {
	if goal.Kyoom {
		return "sum"
	}
	return "last"
}

// aggregateDay reduces one day's datapoint values to a single value using the
// named aggday method. vals must be in datapoint order (ascending timestamp) so
// "first"/"last" pick the correct ends, and is always non-empty (a day exists
// only because it has at least one datapoint). An unrecognised method falls back
// to the goal's default.
func aggregateDay(goal Goal, name string, vals []float64) float64 {
	switch name {
	case "sum":
		return aggSum(vals)
	case "last":
		return vals[len(vals)-1]
	case "first":
		return vals[0]
	case "min":
		return aggMin(vals)
	case "max":
		return aggMax(vals)
	case "count":
		return float64(len(vals))
	case "mu", "truemean", "average":
		return aggMean(vals)
	case "mean", "munique", "uniqmean":
		return aggMean(aggDedup(vals))
	case "mutrim", "trimmean":
		return aggTrimMean(vals, 0.1)
	case "median":
		return aggMedian(vals)
	case "mode":
		return aggMode(vals)
	case "unary", "binary", "jolly":
		// 1 if any datapoint exists, else 0. vals is non-empty here, so it's 1;
		// the 0 is kept for clarity/parity with beebrain.
		if len(vals) > 0 {
			return 1
		}
		return 0
	case "unaryflat", "nonzero":
		for _, v := range vals {
			if v != 0 {
				return 1
			}
		}
		return 0
	case "triangle":
		s := aggSum(vals)
		return s * (s + 1) / 2
	case "square":
		s := aggSum(vals)
		return s * s
	case "clocky":
		return aggClocky(vals)
	case "skatesum":
		// Sum, capped at the goal's daily rate. When the rate isn't usable
		// (missing, or runits we can't convert), fall back to a plain sum rather
		// than cap at a dimensionally-wrong value.
		if r, ok := goalDailyRate(goal); ok {
			return math.Min(r, aggSum(vals))
		}
		return aggSum(vals)
	case "satsum", "cap1":
		return math.Min(1, aggSum(vals))
	case "sqrt":
		return math.Sqrt(aggSum(vals))
	case "countflat":
		return float64(countNonzero(vals))
	case "muflat":
		return aggMean(nonzero(vals))
	default:
		// Unknown method: render with the goal's default rather than nothing.
		def := defaultAggday(goal)
		if name == def {
			return aggSum(vals) // guard against recursion if a default is ever unknown
		}
		return aggregateDay(goal, def, vals)
	}
}

// goalDailyRate converts the goal's bright-line rate into gunits/day for the
// skatesum cap. ok is false when the rate is absent or expressed in runits we
// can't translate.
func goalDailyRate(g Goal) (float64, bool) {
	if g.Rate == nil || !isKnownRunits(g.Runits) {
		return 0, false
	}
	return ratePerDay(*g.Rate, g.Runits), true
}

// Day-bucketing: the other half of "aggregate datapoints by day". bucketByDay
// groups raw datapoints into timezone-aware calendar days; aggregateByDay then
// reduces each day via the goal's aggday. Keeping both halves here (beside
// aggregateDay/resolveAggday) means the whole "how a datapoint becomes one
// charted value for its day" story lives in one module — chart.go layers only
// windowing and the kyoom running total on top (see processDatapoints).

// startOfDay floors an instant to midnight of its own calendar day in loc.
// t is converted into loc first so the calendar day is read in loc — correct
// for any caller regardless of t's original zone.
func startOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// dayValue is one calendar day reduced to a single plotted value, positioned at
// the day's local-midnight boundary. aggregateByDay returns these in ascending
// day order, making the sort order the chart series depends on explicit in the
// result type rather than a comment on a bare []float64.
type dayValue struct {
	day   time.Time
	value float64
}

// aggregateByDay is the aggday module's end-to-end "datapoints → one value per
// day" reduction: it buckets datapoints into timezone-aware calendar days and
// reduces each day to a single value via the goal's aggday. The result is in
// ascending day order. Callers layer windowing and (for kyoom goals) the
// running total on top — see processDatapoints.
func aggregateByDay(goal Goal, datapoints []Datapoint, loc *time.Location) []dayValue {
	aggday := resolveAggday(goal)
	buckets := bucketByDay(datapoints, loc)
	out := make([]dayValue, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, dayValue{day: b.day, value: aggregateDay(goal, aggday, b.values)})
	}
	return out
}

// dayBucket is one calendar day's worth of datapoint values, in ascending
// timestamp order, tagged with the day's start instant (local midnight).
type dayBucket struct {
	day    time.Time
	values []float64
}

// bucketByDay groups datapoints into calendar days, ascending. Each day's
// values stay in datapoint (ascending-timestamp) order so order-sensitive
// aggdays (first/last) pick the right ends.
//
// The day is taken from the datapoint's Beeminder daystamp when present (it
// already accounts for the goal's deadline), otherwise from the timestamp. Both
// are resolved in loc — the same zone the chart window uses — so day boundaries
// line up with the window and x-axis.
func bucketByDay(datapoints []Datapoint, loc *time.Location) []dayBucket {
	sorted := append([]Datapoint(nil), datapoints...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Timestamp < sorted[j].Timestamp
	})

	index := make(map[string]int)
	var buckets []dayBucket
	for _, dp := range sorted {
		day := dayStart(dp, loc)
		key := day.Format("2006-01-02")
		if i, ok := index[key]; ok {
			buckets[i].values = append(buckets[i].values, dp.Value)
		} else {
			index[key] = len(buckets)
			buckets = append(buckets, dayBucket{day: day, values: []float64{dp.Value}})
		}
	}

	// Buckets are first-seen in ascending-timestamp order, which is already
	// ascending-day order; sort defensively in case daystamps and timestamps
	// disagree near a boundary.
	sort.SliceStable(buckets, func(i, j int) bool {
		return buckets[i].day.Before(buckets[j].day)
	})
	return buckets
}

// dayStart returns the local-midnight instant of the datapoint's day, preferring
// its daystamp (YYYYMMDD) and falling back to its timestamp.
func dayStart(dp Datapoint, loc *time.Location) time.Time {
	if len(dp.Daystamp) == 8 {
		if t, err := time.ParseInLocation("20060102", dp.Daystamp, loc); err == nil {
			return t
		}
	}
	return startOfDay(time.Unix(dp.Timestamp, 0).In(loc), loc)
}

func aggSum(a []float64) float64 {
	s := 0.0
	for _, v := range a {
		s += v
	}
	return s
}

func aggMean(a []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	return aggSum(a) / float64(len(a))
}

func aggMin(a []float64) float64 {
	m := a[0]
	for _, v := range a[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func aggMax(a []float64) float64 {
	m := a[0]
	for _, v := range a[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// aggDedup drops duplicate values, preserving first-seen order (for uniqmean).
func aggDedup(a []float64) []float64 {
	seen := make(map[float64]bool, len(a))
	out := make([]float64, 0, len(a))
	for _, v := range a {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// aggMedian is the middle value of the sorted list, or the mean of the two
// middle values when the count is even.
func aggMedian(a []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	b := append([]float64(nil), a...)
	sort.Float64s(b)
	l := len(b)
	if l%2 == 0 {
		return (b[l/2-1] + b[l/2]) / 2
	}
	return b[(l-1)/2]
}

// aggMode returns the most common value, breaking ties toward the value that
// first reaches the highest tally (matching beebrain's tie-break closely enough
// — aggday=mode is vanishingly rare).
func aggMode(a []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	tally := make(map[float64]int, len(a))
	maxTally := 1
	item := a[0]
	for _, v := range a {
		tally[v]++
		if tally[v] > maxTally {
			maxTally = tally[v]
			item = v
		}
	}
	return item
}

// aggTrimMean is the mean after dropping the lowest and highest trim-fraction of
// the sorted values.
func aggTrimMean(a []float64, trim float64) float64 {
	b := append([]float64(nil), a...)
	sort.Float64s(b)
	n := int(math.Floor(float64(len(b)) * trim))
	t := b[n : len(b)-n]
	if len(t) == 0 {
		return 0
	}
	return aggSum(t) / float64(len(t))
}

// aggClocky sums the differences of consecutive pairs, e.g. [1,2,6,9] →
// (2-1)+(9-6) = 4. A trailing unpaired value is ignored. Used for timer-style
// goals that log start/stop timestamps.
func aggClocky(a []float64) float64 {
	s := 0.0
	for i := 1; i < len(a); i += 2 {
		s += a[i] - a[i-1]
	}
	return s
}

func countNonzero(a []float64) int {
	n := 0
	for _, v := range a {
		if v != 0 {
			n++
		}
	}
	return n
}

func nonzero(a []float64) []float64 {
	out := make([]float64, 0, len(a))
	for _, v := range a {
		if v != 0 {
			out = append(out, v)
		}
	}
	return out
}
