package main

import (
	"sort"
	"strings"
)

// Goal represents a Beeminder goal with relevant fields
type Goal struct {
	Slug        string                `json:"slug"`
	Title       string                `json:"title"`
	Fineprint   string                `json:"fineprint"` // User-provided description of what they're committing to
	GoalType    string                `json:"goal_type"` // Goal type (hustler, biker, fatloser, gainer, inboxer, drinker)
	Losedate    int64                 `json:"losedate"`
	Pledge      float64               `json:"pledge"`
	PledgeCap   *float64              `json:"pledge_cap"` // Pointer to handle null values from API
	Safebuf     int                   `json:"safebuf"`
	Limsum      string                `json:"limsum"`
	Baremin     string                `json:"baremin"`
	Autodata    string                `json:"autodata"`
	Autoratchet *float64              `json:"autoratchet"` // Pointer to handle null values from API
	Rate        *float64              `json:"rate"`        // Pointer to handle null values from API
	Runits      string                `json:"runits"`
	Gunits      string                `json:"gunits"`     // Goal units, like "hours" or "pushups" or "pages"
	Deadline    int                   `json:"deadline"`   // Seconds by which deadline differs from midnight
	Yaw         int                   `json:"yaw"`        // Good side of the bright red line (+1 = above, -1 = below)
	Dir         int                   `json:"dir"`        // Direction the bright red line is sloping (+1 = up, -1 = down)
	Kyoom       bool                  `json:"kyoom"`      // Whether goal is cumulative/auto-summing
	Tmin        string                `json:"tmin"`       // Min date for graph view (yyyy-mm-dd format)
	Tmax        string                `json:"tmax"`       // Max date for graph view (yyyy-mm-dd format)
	Curval      *float64              `json:"curval"`     // Most recent datapoint value
	Goalval     *float64              `json:"goalval"`    // End value of the goal (may be null if computed from goaldate+rate)
	Mathishard  []*float64            `json:"mathishard"` // [goaldate, goalval, rate] all filled in (may be null in error states)
	Roadall     [][]*float64          `json:"roadall"`    // Full piecewise bright line: rows of [t, v, r] with exactly one of v/r null per row (except the first row, which anchors the road start)
	Dueby       map[string]DuebyEntry `json:"dueby"`      // Per-daystamp deltas/totals, pre-rounded to the goal's display precision. Keys are YYYYMMDD strings.
	Datapoints  []Datapoint           `json:"datapoints,omitempty"`
}

// DuebyEntry is one entry in a goal's `dueby` map, keyed by daystamp.
// Beeminder pre-rounds FormattedDelta and FormattedTotal to the goal's
// configured Display Precision, so honouring those strings avoids the
// trailing-decimals problem we'd hit doing float arithmetic ourselves.
type DuebyEntry struct {
	Delta          float64 `json:"delta"`
	Total          float64 `json:"total"`
	FormattedDelta string  `json:"formatted_delta_for_beedroid"`
	FormattedTotal string  `json:"formatted_total_for_beedroid"`
}

// Datapoint represents a Beeminder datapoint
type Datapoint struct {
	ID        string  `json:"id"`
	Timestamp int64   `json:"timestamp"`
	Daystamp  string  `json:"daystamp"`
	Value     float64 `json:"value"`
	Comment   string  `json:"comment"`
}

// Charge represents a Beeminder charge response
type Charge struct {
	ID       string  `json:"id"`
	Amount   float64 `json:"amount"`
	Note     string  `json:"note"`
	Username string  `json:"username"`
}

// filterOutEndValueReached returns a new slice containing only goals whose
// end value has not yet been reached. Used by views that surface "next/most
// urgent" goals so completed goals (which can show a negative baremin and a
// past losedate) don't crowd out goals that still need attention.
func filterOutEndValueReached(goals []Goal) []Goal {
	out := make([]Goal, 0, len(goals))
	for _, g := range goals {
		if IsEndValueReached(g) {
			continue
		}
		out = append(out, g)
	}
	return out
}

// SortGoals sorts goals by: 1. Due ascending, 2. Stakes descending, 3. Name ascending
func SortGoals(goals []Goal) {
	sort.Slice(goals, func(i, j int) bool {
		// 1. Due ascending (losedate)
		if goals[i].Losedate != goals[j].Losedate {
			return goals[i].Losedate < goals[j].Losedate
		}
		// 2. Stakes descending (pledge)
		if goals[i].Pledge != goals[j].Pledge {
			return goals[i].Pledge > goals[j].Pledge
		}
		// 3. Name alphabetical ascending (slug)
		return goals[i].Slug < goals[j].Slug
	})
}

// SortGoalsBySlug sorts goals alphabetically by slug
func SortGoalsBySlug(goals []Goal) {
	sort.Slice(goals, func(i, j int) bool {
		return goals[i].Slug < goals[j].Slug
	})
}

// ParseLimsumValue extracts the delta value from limsum string
// e.g., "+2 within 1 day" -> "2", "+1 in 3 hours" -> "1", "0 today" -> "0"
// Time formats are preserved: "+00:05 within 1 day" -> "00:05", "+1:30 in 2 hours" -> "1:30"
func ParseLimsumValue(limsum string) string {
	if limsum == "" {
		return "0"
	}
	var value string
	// Split on " within "
	parts := strings.Split(limsum, " within ")
	if len(parts) == 2 {
		value = parts[0]
	} else {
		// Split on " in "
		parts = strings.Split(limsum, " in ")
		if len(parts) == 2 {
			value = parts[0]
		} else {
			// Handle "0 today" or similar cases - extract just the number/value at the start
			fields := strings.Fields(limsum)
			if len(fields) > 0 {
				value = fields[0]
			} else {
				// If format doesn't match, return "0" as fallback
				return "0"
			}
		}
	}
	// Strip leading plus sign
	cleaned := strings.TrimPrefix(value, "+")
	// Return "0" if the cleaned value is empty
	if cleaned == "" {
		return "0"
	}
	return cleaned
}

// ParseBareminValue extracts the delta value from baremin string
// e.g., "+2 in 3 days" -> "2", "-1.5 in 2 hours" -> "-1.5", "3:00 in 1 day" -> "3:00"
func ParseBareminValue(baremin string) string {
	if baremin == "" {
		return "0"
	}
	var value string
	// Split on " in "
	parts := strings.Split(baremin, " in ")
	if len(parts) == 2 {
		value = parts[0]
	} else {
		// Handle edge cases - extract just the number/value at the start
		fields := strings.Fields(baremin)
		if len(fields) > 0 {
			value = fields[0]
		} else {
			return "0"
		}
	}

	// Remove leading "+" if present (but keep "-" for negative values)
	value = strings.TrimPrefix(value, "+")

	// Return "0" if the value is empty after cleanup
	if value == "" {
		return "0"
	}

	return value
}

// IsEndValueReached reports whether the goal's current value has already met or passed
// its end value (goalval). When this is true the bright red line has plateaued and the
// goal effectively has no remaining work, so it shouldn't be surfaced as "due".
//
// Returns false when the end value can't be determined (e.g., goalval and mathishard
// are both nil, or direction is unknown), so callers don't accidentally hide goals.
//
// Do-less goals are excluded: their goalval is an ongoing cap, not an endpoint to
// reach, so curval crossing it indicates a problem state (at/over cap) rather than
// completion. Hiding such goals would mask the very situations they're meant to flag.
func IsEndValueReached(goal Goal) bool {
	if IsDoLessGoal(goal) {
		return false
	}
	if goal.Curval == nil {
		return false
	}
	goalval := resolveGoalval(goal)
	if goalval == nil {
		return false
	}
	switch {
	case goal.Dir > 0:
		return *goal.Curval >= *goalval
	case goal.Dir < 0:
		return *goal.Curval <= *goalval
	default:
		return false
	}
}

// resolveGoalval returns the goal's end value, preferring the direct goalval field and
// falling back to mathishard[1] (which Beeminder fills in even when goalval itself is
// the computed-of-three value).
func resolveGoalval(goal Goal) *float64 {
	if goal.Goalval != nil {
		return goal.Goalval
	}
	if len(goal.Mathishard) >= 2 && goal.Mathishard[1] != nil {
		return goal.Mathishard[1]
	}
	return nil
}

// IsDoLess checks if a goal is a "do-less" type goal based on goal_type string.
// In Beeminder, do-less goals have goal_type "drinker".
// The naming comes from Beeminder's internal convention where goal types
// are represented by descriptive shorthand names (e.g., "hustler" for do-more,
// "biker" for odometer, "fatloser" for weight loss, "drinker" for do-less).
func IsDoLess(goalType string) bool {
	return goalType == "drinker"
}

// IsDoLessGoal checks if a goal is a "do-less" type goal.
// A goal is considered "do-less" if:
//  1. Its goal_type is "drinker" (the standard do-less type), OR
//  2. It has the WEEN platonic goal type attributes (yaw = -1 and dir = 1),
//     which represents a do-less goal where you must stay below an upward-sloping
//     line (e.g., limit cigarettes, reduce social media usage). This handles custom goals
//     that are configured to behave like do-less goals.
func IsDoLessGoal(goal Goal) bool {
	// Check for the standard "drinker" goal type
	if goal.GoalType == "drinker" {
		return true
	}
	// Check for the WEEN platonic goal type (yaw = -1, dir = 1)
	// This handles custom goals configured as do-less
	if goal.Yaw == -1 && goal.Dir == 1 {
		return true
	}
	return false
}
