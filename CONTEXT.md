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
