# Extract input forms as a deep module; do not bisect appModel into domain/UI halves

**Status:** accepted

In addressing #246 ("Separate UI state from domain state in appModel"), we rejected the issue's own proposed direction — splitting `appModel` into a domain struct (goals/config/selection) plus UI sub-states — because it fails the deletion test: navigation handlers inherently need `goals` *and* `cursor` *and* `width` (for column layout) *and* `scrollRow` together, so bisecting the struct only relocates fields without concentrating any complexity. Instead we extracted the real deep module: a generic `form` (a `[]field` with per-field rune filters, a focus index, and shared `handleRune`/`backspace`/`tab` behavior), with thin typed wrappers (`datapointForm`, `createGoalForm`) that embed it, add named accessors, lifecycle flags, and a `validate()`. This concentrates the char-filter ↔ deletion ↔ focus logic in one place whose small interface *is* the unit-test surface — which is the concrete cost #246 identified (interactive handlers were untestable because constructing an `appModel` required dozens of unrelated fields).

## Considered options

- **Bisect `appModel` into domain vs UI structs** (the issue's proposal) — rejected; shallow reorg, fights Bubble Tea idioms, and the issue itself flagged it as low-confidence.
- **Three concrete form structs with duplicated cycling/backspace logic** — rejected; three adapters of the same pattern is a real seam asking for one deep module, not three shallow copies.
- **Generic `form` + thin typed wrappers** — chosen.

## Consequences

- `appModel` still holds `goals`, `cursor`, and the UI mode booleans (`searchMode`, `showModal`, `inputMode`, `showCreateModal`, `modalGoal`). A reader expecting a clean domain/UI split after #246 will not find one — that was deliberate.
- `search` was left as plain `appModel` state (single field, no focus/validation) rather than forced into the form abstraction.
- The mode booleans remain a booleans-as-state-machine smell, deferred to a focused follow-up: collapse them into a `Mode` enum (#292).
