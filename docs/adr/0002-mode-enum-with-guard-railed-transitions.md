# Collapse appModel UI mode booleans into a `mode` enum with guard-railed transitions

**Status:** accepted

## Context

After #246/#293 (see [ADR-0001](./0001-extract-input-forms-not-bisect-appmodel.md)), `appModel` still encoded "what screen is the UI showing" as a set of overlapping booleans — `searchMode`, `showModal` (+ `modalGoal`), `inputMode`, and `showCreateModal`. Handlers were full of guards like `if showModal && inputMode && !submitting` and `if !showModal && !showCreateModal && !searchMode`. The booleans were *mostly* mutually exclusive, so invalid combinations were representable but never intended — a classic booleans-as-state-machine smell that #292 set out to remove.

Grilling the design surfaced two things the naive "swap 4 bools for 1 enum" framing missed:

1. **Search is not a peer mode — it's a filter layer.** Pressing Enter while searching opens the goal-detail modal *over the filtered list*, so `searchMode` and `showModal` legitimately coexist. The old Esc handler checked `searchMode` first, so Esc on a modal-opened-over-search wiped the search and left the modal open — a latent bug. Search therefore stays orthogonal to the mode (its own `searchActive` + `searchQuery`), and Esc on the modal now closes the modal while preserving the search.
2. **"Saving" is a busy flag, not a mode.** The in-flight `datapoint.submitting` / `createGoal.creating` flags stay on the form structs. Folding them into the enum would double every form state (input vs input-while-saving) and reintroduce the combinatorial mess.

## Decision

Introduce an unexported `mode` enum on `appModel` with exactly four foreground states: `modeBrowse`, `modeGoalDetail`, `modeDatapointInput` (reachable only from `modeGoalDetail`), and `modeCreateGoal`. Remove `showModal`, `inputMode`, and `showCreateModal`. Rename `searchMode` → `searchActive` to reflect that it is a filter layer, not a mode. Keep `modalGoal`.

Crucially, **the mode is mutated only through a small set of transition methods** on `appModel` (`openGoalDetail`, `startDatapointInput`, `exitDatapointInput`, `closeModal`, `openCreateGoal`, `closeCreateGoal`, `enterSearch`, `exitSearch`). Each sets the mode *and* its required companions atomically (e.g. `openGoalDetail(goal)` is the only door into `modeGoalDetail`, and it attaches the goal). Handlers call these instead of poking fields. This makes the invariant "`modalGoal` is non-nil ⟺ mode is `modeGoalDetail`/`modeDatapointInput`" hold by construction rather than by convention, and gives a small, directly unit-testable surface — the testability win ADR-0001 was built toward.

## Considered options

- **Plain `mode` field, handlers assign it directly** — rejected. Trades 4 booleans for 1 field but keeps the invariants informal; a future edit could enter `modeGoalDetail` without attaching a goal (nil-pointer crash). The "impossible states" guarantee would be convention-only.
- **Search as a fifth peer mode in a flat enum** — rejected. Would force either forbidding "open a goal from filtered results" or duplicating states (`SearchThenModal`). Search genuinely persists underneath the modal.
- **Folding "saving" into the enum** (`modeDatapointSaving`, …) — rejected. Doubles the form states.
- **`mode` enum + guard-railed transition methods + search as an orthogonal filter layer** — chosen.

## Consequences

- Invalid mode/companion combinations are no longer representable through normal code paths; tests target the transition methods directly.
- One deliberate behavior change: Esc on a goal-detail modal opened over a search now closes the modal and **keeps** the search (previously it cleared the search and left the modal open).
- `searchActive` remains orthogonal to `mode`; any mode except `modeCreateGoal` can have a search active underneath. `modeCreateGoal` is only reachable from `modeBrowse` with no active search (letter keys are consumed as search text while searching, so `n` cannot open the create form mid-search).
- Rendering functions in `grid.go` keep their boolean parameters; `tui.go` derives them from the mode at the call site, keeping the view layer decoupled from the `mode` type.
