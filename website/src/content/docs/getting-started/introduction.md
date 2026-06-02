---
title: Introduction
description: What buzz is and how the docs are organized.
---

**buzz** is a terminal user interface (TUI) and command-line tool for
[Beeminder](https://beeminder.com), built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

View your goals in a colorful grid, navigate with arrow keys, and add datapoints
directly from your terminal — or skip the TUI entirely and use the scriptable
subcommands.

## Two ways to use buzz

Running `buzz` with no arguments launches the **interactive TUI**: a goal grid
you navigate with the keyboard, color-coded by deadline urgency.

Every common action is also available as a **one-shot subcommand** — `buzz
today`, `buzz add`, `buzz ratchet`, and more — which print plain output and exit.
These are designed for scripts, cron jobs, and status bars.

## Where to go next

- [Installation](/getting-started/installation/) — install buzz via `bin`,
  Homebrew, a direct download, or `go install`.
- [Authentication](/getting-started/authentication/) — connect buzz to your
  Beeminder account.
- [Configuration](/getting-started/configuration/) — optional settings such as
  request logging.
- [Commands overview](/commands/overview/) — the full command-line reference.
- [Using the TUI](/guides/tui/) — keyboard navigation and the goal grid.
