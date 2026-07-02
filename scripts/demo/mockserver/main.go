// Command mockserver is a tiny stand-in for the Beeminder API used only to
// record the buzz demo (see scripts/demo/record.sh). It serves a fixed set of
// fictional goals so the demo never touches a real Beeminder account.
//
// Goal data (losedate, the dueby forecast, and recent datapoints) is computed
// relative to the current date at startup, so the recorded demo always shows
// live-looking countdowns and a populated 7-Day Forecast no matter when it runs.
//
// It implements just the endpoints buzz hits for the demo:
//
//	GET  /api/v1/users/{user}.json                          → account (timezone)
//	GET  /api/v1/users/{user}/goals.json                    → goal list (TUI dashboard, buzz list, buzz today)
//	GET  /api/v1/users/{user}/goals/{slug}.json             → one goal w/ datapoints (buzz view)
//	POST /api/v1/users/{user}/goals/{slug}/datapoints.json  → acknowledge a datapoint (buzz add)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// demoGoal is the template for a fictional goal; the dynamic fields (losedate,
// dueby, datapoints) are filled in relative to "now" when the server starts.
type demoGoal struct {
	Slug     string
	Title    string
	GoalType string
	Gunits   string
	Runits   string
	Limsum   string
	Rate     float64
	Pledge   float64
	Safebuf  int
	Yaw      int
	Dir      int
	// perDay is the formatted amount due each day in the forecast. timey
	// switches the dueby strings to HH:MM:SS so we showcase a time-based goal.
	perDay int
	timey  bool
	// base is the running total at the start of the forecast.
	base int
	// kyoom marks a cumulative goal: the progress chart sums its datapoints into
	// a rising staircase (and exercises the step-rendering path). lvl is the
	// typical per-day metric value for non-cumulative goals, used to scale their
	// chart's datapoints and bright red line so the plot stays readable.
	kyoom bool
	lvl   int
	// historyDays overrides how far back this goal's chart reaches (initday and
	// datapoint history). 0 uses chartWindowDays. A long history makes a *dense*
	// chart where datapoint nodes fill nearly every column — so the demo shows
	// both the sparse chart (dotted datapoint markers) and the dense one (none).
	historyDays int
}

// chartWindowDays is the default reach of a goal's progress chart: initday and
// the datapoint history both span this many days before today, so `buzz review`
// shows a fully-populated chart. Goals can override it via historyDays.
const chartWindowDays = 14

// window returns how many days of history the goal's chart spans.
func (g demoGoal) window() int {
	if g.historyDays > 0 {
		return g.historyDays
	}
	return chartWindowDays
}

// demoGoals is the cast of fictional goals shown in the demo.
var demoGoals = []demoGoal{
	{Slug: "read", Title: "Read every day", GoalType: "hustler", Gunits: "pages", Runits: "d", Limsum: "+10 within 1 day", Rate: 10, Pledge: 5, Safebuf: 0, Yaw: 1, Dir: 1, perDay: 10, base: 1200, kyoom: true},
	{Slug: "meditate", Title: "Daily meditation", GoalType: "hustler", Gunits: "minutes", Runits: "d", Limsum: "+00:20:00 within 1 day", Rate: 20, Pledge: 10, Safebuf: 1, Yaw: 1, Dir: 1, perDay: 20, timey: true, base: 3000, kyoom: true},
	{Slug: "inbox", Title: "Inbox zero", GoalType: "inboxer", Gunits: "emails", Runits: "d", Limsum: "-3 within 2 days", Rate: -3, Pledge: 5, Safebuf: 3, Yaw: -1, Dir: -1, perDay: 3, base: 40, lvl: 6},
	// pushups carries a long history so its chart is dense — datapoint nodes fill
	// nearly every column, so the marker dots are suppressed (they'd smear the
	// line). Contrast with the sparse goals above, which get dotted.
	{Slug: "pushups", Title: "Push-ups", GoalType: "hustler", Gunits: "pushups", Runits: "w", Limsum: "+50 within 5 days", Rate: 50, Pledge: 30, Safebuf: 7, Yaw: 1, Dir: 1, perDay: 7, base: 900, kyoom: true, historyDays: 540},
	{Slug: "screen", Title: "Limit screen time", GoalType: "drinker", Gunits: "hours", Runits: "d", Limsum: "+0 within 1 day", Rate: 2, Pledge: 90, Safebuf: 2, Yaw: -1, Dir: 1, perDay: 2, base: 14, lvl: 2},
}

func main() {
	port := flag.String("port", "7180", "port to listen on")
	user := flag.String("user", "demo", "username the demo config authenticates as")
	flag.Parse()

	now := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, *user, now)
	})

	addr := "127.0.0.1:" + *port
	// ReadHeaderTimeout guards against a stalled client holding the connection
	// open; harmless for a localhost demo server but keeps linters quiet.
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("mock Beeminder API listening on http://%s (user %q)", addr, *user)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("mockserver: %v", err)
	}
}

func handle(w http.ResponseWriter, r *http.Request, user string, now time.Time) {
	path := r.URL.Path
	log.Printf("%s %s", r.Method, r.URL.RequestURI())
	prefix := "/api/v1/users/" + user

	switch {
	case path == prefix+".json":
		writeJSON(w, map[string]any{"username": user, "timezone": "America/New_York"})

	case r.Method == http.MethodPost && strings.HasPrefix(path, prefix+"/goals/") && strings.HasSuffix(path, "/datapoints.json"):
		// Acknowledge a new datapoint without persisting it. `buzz add` runs
		// last in the demo, so nothing else needs to reflect the change — this
		// keeps the mock stateless.
		writeJSON(w, map[string]any{
			"id":        "demo-new",
			"timestamp": now.Unix(),
			"daystamp":  now.Format("20060102"),
			"value":     1.0,
			"comment":   "logged via buzz",
		})

	case path == prefix+"/goals.json":
		list := make([]map[string]any, 0, len(demoGoals))
		for _, g := range demoGoals {
			list = append(list, goalJSON(g, now, false))
		}
		writeJSON(w, list)

	case strings.HasPrefix(path, prefix+"/goals/") && strings.HasSuffix(path, ".json"):
		slug := strings.TrimSuffix(strings.TrimPrefix(path, prefix+"/goals/"), ".json")
		for _, g := range demoGoals {
			if g.Slug == slug {
				writeJSON(w, goalJSON(g, now, true))
				return
			}
		}
		http.Error(w, `{"errors":"goal not found"}`, http.StatusNotFound)

	default:
		http.Error(w, `{"errors":"not found"}`, http.StatusNotFound)
	}
}

// goalJSON builds the API JSON for a goal, filling dynamic fields relative to
// now. When withData is true it also includes recent datapoints (as the
// single-goal endpoint does).
func goalJSON(g demoGoal, now time.Time, withData bool) map[string]any {
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Deadline at end of the day the goal is due (today + safebuf days).
	losedate := startOfToday.AddDate(0, 0, g.Safebuf+1).Add(-time.Second).Unix()

	out := map[string]any{
		"slug":      g.Slug,
		"title":     g.Title,
		"goal_type": g.GoalType,
		"gunits":    g.Gunits,
		"runits":    g.Runits,
		"limsum":    g.Limsum,
		"rate":      g.Rate,
		"pledge":    g.Pledge,
		"safebuf":   g.Safebuf,
		"yaw":       g.Yaw,
		"dir":       g.Dir,
		"losedate":  losedate,
		"dueby":     dueby(g, startOfToday),
	}
	if withData {
		// The progress chart (buzz review) needs the goal's history, start date,
		// cumulative flag, and bright red line. These ride on the single-goal
		// detail response, mirroring the real API.
		initday := startOfToday.AddDate(0, 0, -g.window())
		out["datapoints"] = datapoints(g, now)
		out["kyoom"] = g.kyoom
		out["initday"] = initday.Unix()
		out["roadall"] = roadall(g, initday, startOfToday)
	}
	return out
}

// roadall builds a two-row bright red line running from the goal's start to a
// week past today. Row 0 anchors the road (t, v set); row 1 is the second
// segment endpoint, matching Beeminder's roadall encoding where a non-anchor row
// sets exactly one of value/rate.
//
// Cumulative goals anchor at 0 and rise at the goal's rate, so the summed
// datapoint staircase climbs alongside the line.
//
// The demo's non-cumulative goals are all Do Less (yaw -1), where the safe side
// is *below* the bright red line. So their road is a flat horizontal cap sitting
// a few units above the datapoints' typical level — the data oscillates safely
// under it and never crosses, which reads as an on-track goal. (An inclined road
// the data dipped across looked like uncaught derails, since the chart doesn't
// draw the derail region.)
func roadall(g demoGoal, initday, startOfToday time.Time) [][]any {
	end := startOfToday.AddDate(0, 0, 7)
	if g.kyoom {
		return [][]any{
			{float64(initday.Unix()), 0.0, nil},
			{float64(end.Unix()), nil, g.Rate},
		}
	}
	// Flat cap above the data. Datapoints peak around lvl+1 (see datapoints'
	// wave, which spans lvl-2..lvl+1), so a cap at lvl+3 keeps the line clear of
	// the highest point with visible headroom.
	capLine := float64(g.lvl) + 3
	return [][]any{
		{float64(initday.Unix()), capLine, nil},
		{float64(end.Unix()), capLine, nil},
	}
}

// dueby builds a seven-day forecast (today..+6) with running totals, formatted
// the way Beeminder pre-formats them (HH:MM:SS for time goals, integers
// otherwise).
func dueby(g demoGoal, startOfToday time.Time) map[string]any {
	out := map[string]any{}
	total := g.base
	for i := 0; i < 7; i++ {
		day := startOfToday.AddDate(0, 0, i)
		total += g.perDay
		delta := (i + 1) * g.perDay // cumulative amount due by that day
		out[day.Format("20060102")] = map[string]any{
			"delta":                        float64(delta),
			"total":                        float64(total),
			"formatted_delta_for_beedroid": "+" + amount(delta, g.timey),
			"formatted_total_for_beedroid": amount(total, g.timey),
		}
	}
	return out
}

// datapoints builds a daily history (g.window() days long) ending yesterday, so
// the progress chart has shape. Cumulative goals get an uneven per-day stream
// (with a couple of zero days) that the chart sums into a staircase; non-
// cumulative goals get values oscillating around their typical daily level.
func datapoints(g demoGoal, now time.Time) []map[string]any {
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	window := g.window()
	initday := startOfToday.AddDate(0, 0, -window)

	// Day-to-day texture. The cumulative goals are all Do More (yaw +1), where the
	// safe side is *above* the bright red line (which rises at the goal's rate).
	// Each day's multiplier averages above 1, and the running sum stays above the
	// day index throughout, so the summed staircase rides above the road with a
	// growing safety buffer — an on-track Do More goal — while the varied steps
	// and a zero day keep the staircase shape. The non-kyoom wave nudges values
	// around the goal's level.
	kyoomMult := []float64{2, 1, 2, 1, 2, 0, 2, 1, 2, 1, 2, 1, 2}
	wave := []float64{0, 1, -1, 0, 1, -2, 1, 0, -1, 1, 0, 1, -1}

	out := make([]map[string]any, 0, window-1)
	for i := 1; i < window; i++ {
		day := initday.AddDate(0, 0, i)
		var value float64
		if g.kyoom {
			value = float64(g.perDay) * kyoomMult[(i-1)%len(kyoomMult)]
		} else {
			value = float64(g.lvl) + wave[(i-1)%len(wave)]
			if value < 0 {
				value = 0
			}
		}
		out = append(out, map[string]any{
			"id":        fmt.Sprintf("%s-%d", g.Slug, i),
			"timestamp": day.Unix(),
			"daystamp":  day.Format("20060102"),
			"value":     value,
			"comment":   "logged via buzz",
		})
	}
	return out
}

// amount formats a count as an integer, or as HH:MM:SS (treating the count as
// minutes) for time-based goals.
func amount(n int, timey bool) string {
	if !timey {
		return fmt.Sprintf("%d", n)
	}
	h := n / 60
	m := n % 60
	return fmt.Sprintf("%02d:%02d:00", h, m)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode error: %v", err)
	}
}
