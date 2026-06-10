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
//	GET /api/v1/users/{user}.json                 → account (timezone)
//	GET /api/v1/users/{user}/goals.json           → goal list (TUI dashboard, buzz list)
//	GET /api/v1/users/{user}/goals/{slug}.json    → one goal w/ datapoints (buzz view)
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
}

// demoGoals is the cast of fictional goals shown in the demo.
var demoGoals = []demoGoal{
	{Slug: "read", Title: "Read every day", GoalType: "hustler", Gunits: "pages", Runits: "d", Limsum: "+10 within 1 day", Rate: 10, Pledge: 5, Safebuf: 0, Yaw: 1, Dir: 1, perDay: 10, base: 1200},
	{Slug: "meditate", Title: "Daily meditation", GoalType: "hustler", Gunits: "minutes", Runits: "d", Limsum: "+00:20:00 within 1 day", Rate: 20, Pledge: 10, Safebuf: 1, Yaw: 1, Dir: 1, perDay: 20, timey: true, base: 3000},
	{Slug: "inbox", Title: "Inbox zero", GoalType: "inboxer", Gunits: "emails", Runits: "d", Limsum: "-3 within 2 days", Rate: -3, Pledge: 5, Safebuf: 3, Yaw: -1, Dir: -1, perDay: 3, base: 40},
	{Slug: "pushups", Title: "Push-ups", GoalType: "hustler", Gunits: "pushups", Runits: "w", Limsum: "+50 within 5 days", Rate: 50, Pledge: 30, Safebuf: 7, Yaw: 1, Dir: 1, perDay: 7, base: 900},
	{Slug: "screen", Title: "Limit screen time", GoalType: "drinker", Gunits: "hours", Runits: "d", Limsum: "+0 within 1 day", Rate: 2, Pledge: 90, Safebuf: 2, Yaw: -1, Dir: 1, perDay: 2, base: 14},
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
		out["datapoints"] = datapoints(g, now)
	}
	return out
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

// datapoints builds the three most recent days of entries.
func datapoints(g demoGoal, now time.Time) []map[string]any {
	out := make([]map[string]any, 0, 3)
	for i := 3; i >= 1; i-- {
		day := now.AddDate(0, 0, -i)
		out = append(out, map[string]any{
			"id":        fmt.Sprintf("%s-%d", g.Slug, i),
			"timestamp": day.Unix(),
			"daystamp":  day.Format("20060102"),
			"value":     float64(g.perDay),
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
