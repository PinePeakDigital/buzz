# Demo recording

Generates `demo.gif` — the animated demo shown in the project [README](../../README.md).

## How it works

The demo never touches a real Beeminder account. Instead:

- **`mockserver/`** — a tiny stand-in for the Beeminder API serving a fixed cast
  of fictional goals. Their `losedate`, the `dueby` 7-day forecast, and recent
  datapoints are computed relative to the current date at startup, so the
  recording always shows live-looking countdowns and a populated forecast.
- **`demo.tape`** — the [VHS](https://github.com/charmbracelet/vhs) script that
  drives the recording (launch the TUI, navigate, then `buzz list` and
  `buzz view`).
- **`record.sh`** — builds `buzz`, starts the mock server, points `buzz` at it
  via an isolated `$HOME/.buzzrc` (so the real `~/.buzzrc` is never touched),
  and runs VHS.

## Recording locally

```bash
brew install vhs   # also pulls ttyd + ffmpeg
./scripts/demo/record.sh
```

## In CI

- **On every PR** (`.github/workflows/ci.yml`): the GIF is recorded, uploaded as
  a workflow artifact, and (for same-repo PRs) attached to the PR's pre-release
  and embedded in a PR comment.
- **On release** (`.github/workflows/release.yml`): the GIF is re-recorded and
  committed back to `main` so the README always shows the latest UI.
