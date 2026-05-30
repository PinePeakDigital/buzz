---
name: update-readme
description: "Update README.md to reflect the current feature set, commands, and installation options of the buzz TUI. Use when README has drifted from the code or after shipping user-visible changes."
---

# Update README

You are a documentation maintainer for the buzz Beeminder TUI. Your role is to keep `README.md` accurate and helpful for end users by reconciling it against the current state of the code and release artifacts.

## What You Do

- Inspect what the application currently does (commands, flags, keybindings, configuration)
- Compare the live behavior to what `README.md` claims
- Update sections that have drifted (installation, usage, keybindings, configuration, screenshots)
- Preserve the existing tone and section structure unless asked to restructure

## What You Don't Do

- Add aspirational features that don't exist yet
- Duplicate content already covered by `DEVELOPMENT.md` or `docs/*` — link to those instead
- Rewrite the whole README when a targeted edit suffices
- Add badges, marketing copy, or emoji unless the user requests them

## Workflow

### Step 1: Read the current README

```bash
cat README.md
```

### Step 2: Discover the real surface area

Identify what the app actually exposes today:

```bash
# CLI subcommands and flags. Flags are defined via flag.NewFlagSet(...) and then
# <flagSet>.Bool/String/Int(...) (e.g. nextFlags.Bool("watch", ...)), so match the
# FlagSet constructor and its method calls — a `flag.String(...)` search misses them.
grep -nE 'flag\.NewFlagSet|\.(Bool|String|Int|Int64|Uint|Float64|Duration)\("|case "' main.go *.go

# Keybindings (Bubble Tea handlers)
grep -nE 'tea\.KeyMsg|key\.Matches|"q"|"j"|"k"|ctrl\+' *.go

# Config keys
grep -nE 'viper|os\.Getenv|json:"' config.go

# Install options surfaced by release tooling
ls scripts/ && cat scripts/*.sh 2>/dev/null | head -50
```

Cross-check the latest release for binary names and platforms:

```bash
gh release view --json name,tagName,assets -q '.assets[].name'
```

### Step 3: Identify drift

For each README section, list concrete drift items (e.g. "README mentions `--token` but the flag is now `--auth-token`", "Homebrew tap path changed", "Missing the new `uncle` command"). Walk through:

- Installation methods (bin, Homebrew, direct download)
- Quickstart / first run
- Keybindings table
- Configuration (env vars, config file path)
- Commands / subcommands
- Screenshot or asciinema, if any

### Step 4: Edit README.md

Apply targeted edits with the `Edit` tool. Keep changes minimal and grounded in what you found in Step 2.

### Step 5: Verify

```bash
# Render-check that markdown is well-formed
grep -nE '^#{1,6} ' README.md

# Make sure every code block has a language tag
awk '/^```/{n++; if(n%2==1 && $0=="```") print NR": untagged fence"}' README.md
```

### Step 6: Summarize the change

Tell the user the concrete drift items you fixed and anything you intentionally left alone (and why).

## Tips

- The buzz binary is built with `go build`; running `./buzz --help` is the fastest way to confirm flags.
- For keybindings, `model.go` and `handlers.go` are the authoritative source.
- If a feature was added recently, `git log -- README.md` will show whether docs were updated in the same PR.
- Don't invent install methods — only document the ones present in `scripts/` or recent releases.
