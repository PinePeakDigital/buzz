# Bright red line parsing: surface failures loudly; strict validator grounded in observed roadall, not the API doc

**Status:** accepted

## Context

[#317](https://github.com/PinePeakDigital/buzz/issues/317) deepens the three duplicated walkers of a goal's bright red line — `getRoadValueAtTime` and `segmentSlopePerSecond` (chart.go) and `roadallSlopePerDayAt` (slope.go) — into one module that parses the `roadall` API field once into materialised segments and answers value-at-time / slope-at-time. (The structural design — materialised `[]segment`, pure geometry with the `g.Rate` fallback kept caller-side, the value-after-rate gap closed — lives in the issue.)

Two questions that surfaced during design are not about structure but about **policy**, and both are easy for a future reader to "correct" in the wrong direction:

1. What should happen when the `roadall` can't be parsed.
2. What counts as a malformed row — because the API doc and reality disagree.

On (2): `docs/beeminder-api.md:457` (copied from Beeminder's own API docs) states every `roadall` ends with an all-set `[goaldate, goalval, rate]` row. The repo's test fixtures (`piecewiseRoadall`) assume the opposite — every non-anchor row has *exactly one* of value/rate null. The two contradict each other on the exact thing a loud validator keys off. A read-only audit of a live 60-goal account settled it empirically: **0/60** goals had any all-set row; every terminal row was a rate-row `[t, null, r]` to a far-future goaldate; no both-null interior rows, no rows with length ≠ 3, no short roads, only known runits (`d`, `w`). The fixtures are right; the doc is wrong — or describes `fullroad`/a different encoding — for what buzz actually receives.

**Correction (2026-06-11):** the original validator also required times to *strictly* increase (`curT > prevT`), and that sub-rule was the one invariant the audit above never checked. It is wrong. Re-auditing the same 60-goal account read-only found **3208 rows across 52/60 goals** where a row shares its predecessor's exact timestamp — a *vertical step*, the line jumping instantaneously (typically a rate-row immediately followed by a value-row at the same instant, e.g. `[t, null, 0.1], [t, 2.4, null]`). These are legitimate and common; the strict gate blanked the chart on nearly every goal with the banner "time must be greater than the previous row time". **Zero** rows had a strictly *earlier* time. So the rule is corrected to non-decreasing: equal times (vertical steps) are valid and materialise as zero-duration segments (`valueAt` returns a zero-duration segment's endpoint directly; `slopePerDayAt` skips zero-duration segments, so the value-step's inert 0 slope is never reported as a real rate — including when a step is the road's first segment, which 3/60 goals exhibit). The value-row slope is left 0 only to avoid a divide-by-zero. A strictly *earlier* time remains malformed and surfaced.

## Decision

**Surface parse failures; don't degrade silently.** A malformed `roadall` almost certainly means a bug in our parser (more likely) or upstream — not a condition to paper over.

- **Absent** bright red line (`roadall` length < 2): *not* an error. A view that would draw a graph instead shows "the bright red line wasn't populated for this goal"; non-view callers (e.g. `buzz tomorrow`'s baremin bump) fall back to `g.Rate` silently.
- **Malformed** `roadall` (present but structurally invalid): surfaced loudly and specifically — the chart replaces the line with a banner naming the defect; `buzz tomorrow` shows a per-goal `⚠` marker. Parsing is all-or-nothing: one bad row fails the whole parse.
- **Valid:** parsed into materialised segments.

So the parser returns three distinguishable outcomes and callers branch `err != nil → alarm; length 0 → "not populated"; else use it`.

**The validator stays strict, grounded in observed data rather than the doc.** A non-anchor row must have exactly one of value/rate set. The doc's all-set terminal row is treated as an *unobserved anomaly*: if it ever appears, the loud failure is the correct outcome (investigate it), not a false alarm to pre-accommodate. The code enforcing this **must carry a comment pointing at `docs/beeminder-api.md:457`** so a future reader does not loosen the rule on the doc's authority — that specific trap is why this ADR exists.

**Terminology.** "Road" / "the yellow brick road" are Beeminder's *deprecated* names for the bright red line. User-facing text says **bright red line**; "road" / `roadall` is reserved for the raw API field.

## Considered options

- **Silent degradation (status quo):** unparseable road → draw nothing / fall back to `g.Rate`. Rejected — hides a probable bug; the user sees a subtly-wrong or empty chart and never learns why.
- **Loosen the validator to accept the doc's all-set terminal row.** Rejected — the shape does not occur in real data, and accepting an unobserved shape on the word of a doc that just proved unreliable trades a real signal (loud-on-anomaly) for defensive handling of a phantom.
- **Treat an absent road as malformed too.** Rejected — a missing `roadall` is a legitimate "no data" state, distinct from corruption; conflating them would cry wolf.
- **Surface loudly + strict validator grounded in observed data** — chosen.

## Consequences

- One deliberate behaviour change vs. today: a malformed road that the old *lazy* code tolerated (drawing a partial line / silently falling back to `g.Rate`) now blanks the chart with an explicit banner. Acceptable because the audit shows real goals don't hit it; if one does, that is the bug-signal working as intended.
- If a future Beeminder change starts emitting the all-set terminal row, **every goal alarms at once** — a loud, immediate prompt to revisit this ADR, by design, rather than a silent drift.
- The grounding rests on one account (60 goals). Other users may have value-defined interior segments (`[t, v, null]` interior rows), which the strict rule already accepts. Re-running the audit against a second account before merge is cheap insurance.
- The "absent vs. malformed vs. valid" three-way contract becomes the test surface: each outcome is asserted at the parser, and each caller's branch (alarm / "not populated" / draw) is tested independently.
