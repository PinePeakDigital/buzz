# Context: buzz

`buzz` is a terminal UI (TUI) for [Beeminder](https://www.beeminder.com). Its
ubiquitous language therefore covers both the Beeminder domain (goals,
datapoints, rates) and the TUI's interaction model. This file records the latter
where it has been deliberately sharpened; the former follows Beeminder's own
terminology.

## UI interaction vocabulary

- **Mode** — the single foreground screen the app is currently showing. Exactly
  one mode is active at a time. The modes are **Browse**, **Goal detail**,
  **Datapoint input**, and **Create goal**. Modeled as the unexported `mode`
  enum on `appModel` and mutated only through transition methods (see
  [ADR-0002](./docs/adr/0002-mode-enum-with-guard-railed-transitions.md)).
  - **Browse** — the scrollable grid of goals. The default mode.
  - **Goal detail** — a single goal's detail popup, opened over the grid.
  - **Datapoint input** — the form for adding a datapoint, reachable *only* from
    Goal detail and returning there on cancel/submit.
  - **Create goal** — the new-goal form, reachable only from Browse.
- **Search** — a **filter layer**, *not* a mode. When active it filters the
  Browse grid by a query and persists underneath whatever mode is foreground
  (e.g. a Goal detail popup opened from filtered results keeps the search alive
  beneath it). Tracked by `searchActive` + `searchQuery`, orthogonal to `mode`.
- **Busy / in-flight** — a form is awaiting a Beeminder API response
  (`datapoint.submitting`, `createGoal.creating`). A *flag on the form*, not a
  mode; the screen looks the same, just locked against re-submission.

## Bright red line vocabulary

The **bright red line** is a goal's commitment line — the value Beeminder
expects you to stay on the right side of. "Road" and "the yellow brick road"
are Beeminder's *deprecated* names for it: use **bright red line** in all
user-facing text, and reserve "road" / `roadall` for the raw API field. It is
read two ways — its value at a moment, and its slope at a moment — by the module
that owns both (see #317 and
[ADR-0003](./docs/adr/0003-bright-red-line-parsing-failure-policy.md)).

- **`roadall`** — the API field encoding the bright red line: `[][]*float64`
  rows of `[t, v, r]`, the first an anchor (t, v set; r nil), each later row
  with *exactly one* of v/r null. (Observed shape: the API doc claims a final
  all-set `[goaldate, goalval, rate]` row, but real goals don't emit one — the
  validator stays strict and treats an all-set row as an anomaly to surface;
  see ADR-0003.)
- **Segment** — one piece of the bright red line between two boundaries, with
  known start/end time, start/end value, and a **slope**. Parsing `roadall`
  materialises a list of segments **all-or-nothing**: an *absent* `roadall`
  (length < 2) is a benign "not populated" state, but a *malformed* one yields
  no line and is surfaced loudly (it signals a parser or upstream bug).
- **Slope** — how fast the bright red line moves, in gunits/day, at a given
  moment: the slope of the segment containing it. Distinct from `g.Rate`, which
  is the rate at the *end* of the graph, not at any specific moment.
- **Value at time** — the bright red line's value at a moment: interpolated
  within its span, extrapolated backward before the start, held flat past the
  end. Value is defined for all t; slope only *within* the span (outside it,
  callers fall back to `g.Rate`).
