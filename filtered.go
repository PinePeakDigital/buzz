package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Filtered list views: `buzz all`, `buzz today`, `buzz tomorrow`, `buzz due`,
// `buzz less`. They share an orchestration helper (load → filter → sort →
// render via the goaltable.Table) and a small family of filter predicates +
// the tomorrow-view projection (goalByEndOfTomorrowAt), which bumps both
// baremin and losedate together for due-today goals.

// isDoLessFilter returns true if the goal is a do-less type goal
func isDoLessFilter(g Goal) bool {
	return IsDoLessGoal(g)
}

// allGoalsFilter returns true for all goals
func allGoalsFilter(g Goal) bool {
	return true
}

// handleAllCommand outputs all goals
func handleAllCommand() {
	handleFilteredCommand("all", allGoalsFilter)
}

// handleTodayCommand outputs all goals that are due today
func handleTodayCommand() {
	handleFilteredCommand("today", isDueTodayFilter)
}

// handleTomorrowCommand outputs all goals that are due tomorrow. Goals that
// are already due today are included with their baremin bumped by one day's
// rate, so the user sees the total amount they would need to do for the goal
// to not be due tomorrow. The displayed deadline is bumped to match — the
// bumped baremin is what's needed by *tomorrow's* deadline, not today's.
func handleTomorrowCommand() {
	now := time.Now()
	// Memoize the vended pair per goal: losedateFor is called O(n log n) times
	// while sorting and again per deadline column, and each goalByEndOfTomorrowAt
	// runs the relatively expensive bumpedBaremin projection (string parsing +
	// roadall slope lookup). Caching by slug computes it once per goal. Both
	// columns still derive from the same pair, so the bumped baremin and bumped
	// losedate can't disagree (see goalByEndOfTomorrowAt).
	views := make(map[string]tomorrowView)
	viewFor := func(g Goal) tomorrowView {
		if v, ok := views[g.Slug]; ok {
			return v
		}
		v := goalByEndOfTomorrowAt(g, now)
		views[g.Slug] = v
		return v
	}
	filter := func(g Goal) bool { return isDueTomorrowFilterAt(g, now) }
	bareminFor := func(g Goal) string { return viewFor(g).markedBaremin() }
	losedateFor := func(g Goal) int64 { return viewFor(g).losedate }
	handleFilteredCommandWithDisplay("tomorrow", filter, bareminFor, losedateFor)
}

// handleLessCommand outputs all do-less type goals
func handleLessCommand() {
	handleFilteredCommand("do-less", isDoLessFilter)
}

// handleDueCommand outputs all goals due within the specified duration
func handleDueCommand() {
	// Check arguments: buzz due <duration>
	if len(os.Args) < 3 {
		fmt.Println("Error: Missing required duration argument")
		fmt.Println("Usage: buzz due <duration>")
		fmt.Println("  Examples: buzz due 10m, buzz due 1h, buzz due 5d, buzz due 1w")
		fmt.Println("  Supported units: m (minutes), h (hours), d (days), w (weeks)")
		os.Exit(1)
	}

	durationStr := os.Args[2]

	// Parse the duration
	duration, ok := ParseDuration(durationStr)
	if !ok {
		fmt.Printf("Error: Invalid duration format: %s\n", durationStr)
		fmt.Println("Usage: buzz due <duration>")
		fmt.Println("  Examples: buzz due 10m, buzz due 1h, buzz due 5d, buzz due 1w")
		fmt.Println("  Supported units: m (minutes), h (hours), d (days), w (weeks)")
		os.Exit(1)
	}

	// Create filter function that captures the duration
	isDueWithinFilter := func(g Goal) bool {
		return IsDueWithin(g.Losedate, duration)
	}

	// Format the filter name for display
	filterName := fmt.Sprintf("due within %s", durationStr)
	handleFilteredCommand(filterName, isDueWithinFilter)
}

// handleFilteredCommand is a shared helper that outputs all goals matching the given filter
// filterName is used in messages (e.g., "today", "tomorrow", or "do-less")
// filter is a function that takes a Goal and returns true if the goal matches
func handleFilteredCommand(filterName string, filter func(Goal) bool) {
	handleFilteredCommandWithDisplay(filterName, filter,
		func(g Goal) string { return g.Baremin },
		func(g Goal) int64 { return g.Losedate },
	)
}

// sortGoalsByDisplayedLosedate reorders goals in place so the slice ends up
// sorted by the timestamp that losedateFor would render. SliceStable preserves
// the input order for ties so any prior sort (e.g. SortGoals's pledge/slug
// tiebreakers) survives.
func sortGoalsByDisplayedLosedate(goals []Goal, losedateFor func(Goal) int64) {
	sort.SliceStable(goals, func(i, j int) bool {
		return losedateFor(goals[i]) < losedateFor(goals[j])
	})
}

// handleFilteredCommandWithDisplay is the most general filtered-output helper:
// the caller can override both the displayed baremin string and the deadline
// timestamp used for the timeframe/absolute-deadline columns. The tomorrow
// view uses this to bump both for due-today goals so the bumped baremin and
// the displayed deadline are aligned to the same target moment.
func handleFilteredCommandWithDisplay(filterName string, filter func(Goal) bool, bareminFor func(Goal) string, losedateFor func(Goal) int64) {
	// Load config
	if !ConfigExists() {
		fmt.Println("Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)

	// Fetch goals
	goals, err := client.FetchGoals(context.Background())
	if err != nil {
		fmt.Printf("Error: Failed to fetch goals: %s\n", redactError(err))
		os.Exit(1)
	}

	// Sort goals (by due date ascending, then by stakes descending, then by name)
	SortGoals(goals)

	// Filter goals that match the criteria
	var filteredGoals []Goal
	for _, goal := range goals {
		if filter(goal) {
			filteredGoals = append(filteredGoals, goal)
		}
	}

	// If no matching goals, exit
	if len(filteredGoals) == 0 {
		fmt.Printf("No %s goals found.\n", filterName)
		return
	}

	// SortGoals ordered by each goal's own losedate, but the tomorrow view may
	// show a bumped losedate for due-today goals. Re-sort by the displayed
	// losedate so the rendered order matches the deadline column. SliceStable
	// preserves the SortGoals tiebreakers (pledge desc, slug asc) when
	// displayed losedates are equal.
	sortGoalsByDisplayedLosedate(filteredGoals, losedateFor)

	table := Table{
		Colorize: true,
		Columns: []Column{
			{Cell: func(g Goal) string { return g.Slug }},
			{Cell: func(g Goal) string { return bareminFor(g) }},
			{Cell: func(g Goal) string {
				if IsEndValueReached(g) {
					return "COMPLETE"
				}
				return FormatDueDate(losedateFor(g))
			}},
			{Cell: func(g Goal) string { return FormatAbsoluteDeadline(losedateFor(g)) }},
		},
	}
	fmt.Print(table.Render(filteredGoals))

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}

// tomorrowView is what a goal shows in the "due tomorrow" view: the baremin
// string, the losedate timestamp, and whether the goal's bright red line failed
// to parse. baremin and losedate are vended together so they always reflect the
// same horizon — a bumped amount never appears beside an un-bumped deadline, or
// vice-versa.
type tomorrowView struct {
	baremin  string
	losedate int64
	// roadMalformed is true when parseRoad rejected the goal's roadall. The
	// bump silently falls back to g.Rate in that case (slopePerDayAt stays
	// tolerant), so the view flags it with a ⚠ rather than presenting a
	// fallback number as if it were authoritative. An absent road (benign,
	// "not populated") is NOT malformed and carries no marker. See ADR-0003.
	roadMalformed bool
}

// markedBaremin is the baremin string the tomorrow view prints, prefixed with a
// ⚠ when the goal's bright red line is malformed — the bumped amount silently
// fell back to g.Rate, so it's flagged rather than shown as if authoritative
// (#325 / ADR-0003).
func (v tomorrowView) markedBaremin() string {
	if v.roadMalformed {
		return "⚠ " + v.baremin
	}
	return v.baremin
}

// goalByEndOfTomorrowAt vends the baremin, losedate, and road-malformed flag a
// goal should show in the "due tomorrow" view, projected to tomorrow's deadline.
// The due-today gate and `now` are evaluated once here, so baremin and losedate
// can't disagree — the coupling the caller previously had to enforce by wiring
// two closures identically now holds by construction.
//
// A goal due *later today* is bumped: its baremin gains one day's worth of rate
// (so the amount reflects what's required to avoid a beemergency tomorrow) and
// its losedate advances one calendar day. A goal not due later today — overdue
// (losedate already past) or already due tomorrow-or-later — is left as-is:
// baremin reflects what the API reports for tomorrow's deadline already, and
// keeping the real losedate preserves the OVERDUE indicator. See bumpedBaremin
// for the baremin projection and dueLaterTodayAt for the gate.
func goalByEndOfTomorrowAt(g Goal, now time.Time) tomorrowView {
	// A malformed roadall is surfaced regardless of the gate: the bump path
	// swallows the parse error (falling back to g.Rate), so this is the tomorrow
	// view's chance to signal it. An absent road parses without error and is not
	// flagged.
	_, roadErr := parseRoad(g.Roadall, g.Runits)
	malformed := roadErr != nil

	if !dueLaterTodayAt(g, now) {
		return tomorrowView{baremin: stripTimeWindowSuffix(g.Baremin), losedate: g.Losedate, roadMalformed: malformed}
	}
	// Advance the deadline by one calendar day in the caller's local zone so the
	// displayed wall-clock deadline stays correct across DST transitions.
	losedate := time.Unix(g.Losedate, 0).In(now.Location()).AddDate(0, 0, 1).Unix()
	return tomorrowView{baremin: bumpedBaremin(g, now), losedate: losedate, roadMalformed: malformed}
}

// Tomorrow-view baremin bumping. bumpedBaremin projects a goal's "what's needed
// by tomorrow's deadline" baremin string from the API's "what's needed by
// today's deadline" string, by adding one day's worth of rate. Beeminder's
// `dueby` map already carries pre-rounded per-daystamp deltas; we use that
// when available and fall back to local arithmetic otherwise.

// bumpedBaremin returns the bumped baremin string for a goal due later today,
// adding one day's worth of rate so the displayed amount reflects what's
// required to avoid a beemergency tomorrow. It assumes the due-today gate has
// already passed (goalByEndOfTomorrowAt is the only caller).
//
// The per-day slope is taken from the bright-line segment containing `now`
// (via roadall) rather than from g.Rate, because g.Rate reports the goal's
// end-of-graph rate which can differ from the current segment's rate for
// goals with piecewise schedules.
//
// Two baremin value shapes are recognised: plain numeric (e.g. "+2") and the
// colon-separated time format used by hour-valued goals — both "HH:MM" (e.g.
// "+00:25") and "HH:MM:SS" (e.g. "+00:25:00"). For both, the slope is
// interpreted as units-of-baremin per day before being added. The output
// preserves whichever colon format the input used. If the slope can't be
// determined or the value can't be parsed, the original Baremin string is
// returned with any trailing time-window qualifier removed. The returned
// string never carries that qualifier (e.g. " in 1 day", " within 1 day",
// " today") — every row in the tomorrow view shares the same horizon, so
// the suffix is just noise.
func bumpedBaremin(g Goal, now time.Time) string {
	if bumped, ok := bareminFromDueby(g, now); ok {
		return stripTimeWindowSuffix(bumped)
	}
	perDay, ok := slopePerDayAt(g, now)
	if !ok {
		return stripTimeWindowSuffix(g.Baremin)
	}
	value := ParseBareminValue(g.Baremin)

	// Colon-separated time formats — HH:MM or HH:MM:SS. The base value and
	// the rate share the same hour unit, so add in seconds and reformat in
	// whichever colon format the input used.
	if strings.Contains(value, ":") {
		baseSeconds, includeSeconds, ok := parseTimeValue(value)
		if !ok {
			return stripTimeWindowSuffix(g.Baremin)
		}
		// Guard against overflow when converting deltaSeconds to int on
		// 32-bit systems. 1e9 seconds is ~31 years per day, well past any
		// realistic Beeminder rate; treat anything above as a malformed
		// rate and fall back to the un-bumped baremin.
		deltaSeconds := perDay * 3600
		if deltaSeconds > 1e9 || deltaSeconds < -1e9 {
			return stripTimeWindowSuffix(g.Baremin)
		}
		totalSeconds := baseSeconds + int(math.Round(deltaSeconds))
		return formatTimeValue(totalSeconds, includeSeconds)
	}

	// Plain numeric.
	base, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return stripTimeWindowSuffix(g.Baremin)
	}
	total := base + perDay
	sign := "+"
	if total < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%g", sign, total)
}

// stripTimeWindowSuffix removes Beeminder's trailing time-window phrase from a
// baremin/limsum string (e.g. " in 1 day", " within 1 day", " in 3 hours",
// " today"), leaving just the leading signed value. Used by the tomorrow view,
// where every row shares the same horizon so the suffix only adds noise.
func stripTimeWindowSuffix(s string) string {
	if s == "" {
		return s
	}
	for _, sep := range []string{" within ", " in "} {
		if i := strings.LastIndex(s, sep); i >= 0 {
			return s[:i]
		}
	}
	// Handle "+0 today" / "0 today" style: drop the trailing " today" word.
	if strings.HasSuffix(s, " today") {
		return strings.TrimSuffix(s, " today")
	}
	return s
}

// bareminFromDueby returns the formatted delta for tomorrow's daystamp,
// taken from Beeminder's pre-rounded `dueby` map. Beeminder rounds those
// strings to the goal's Display Precision, so honouring them sidesteps
// float-formatting noise (e.g. "+10788.140000000001") that arises when we
// add today's baremin to a per-day rate ourselves. Returns ok=false when
// dueby is absent or the tomorrow entry is missing/empty — callers fall
// back to local arithmetic in those cases.
func bareminFromDueby(g Goal, now time.Time) (string, bool) {
	if len(g.Dueby) == 0 {
		return "", false
	}
	entry, ok := g.Dueby[tomorrowDaystampFor(g, now)]
	if !ok || entry.FormattedDelta == "" {
		return "", false
	}
	return entry.FormattedDelta, true
}

// todayDaystampFor returns the YYYYMMDD daystamp of the goal's current
// Beeminder day. Beeminder shifts day boundaries by the goal's `deadline`
// seconds — positive deadlines push the boundary past midnight (e.g. +10800 =
// 3am cutoff), negative deadlines pull it before (e.g. -10800 = 9pm cutoff).
// Subtracting the deadline from `now` produces a wall-clock time inside the
// user's calendar date that matches the goal's current daystamp.
func todayDaystampFor(g Goal, now time.Time) string {
	shifted := now.Add(-time.Duration(g.Deadline) * time.Second).In(now.Location())
	return time.Date(shifted.Year(), shifted.Month(), shifted.Day(), 0, 0, 0, 0, now.Location()).Format("20060102")
}

// tomorrowDaystampFor returns the YYYYMMDD daystamp representing the day after
// the goal's current Beeminder day (see todayDaystampFor for the deadline-shift
// rationale).
func tomorrowDaystampFor(g Goal, now time.Time) string {
	today, _ := time.ParseInLocation("20060102", todayDaystampFor(g, now), now.Location())
	return today.AddDate(0, 0, 1).Format("20060102")
}
