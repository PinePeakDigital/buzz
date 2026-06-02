---
title: Using the TUI
description: Navigate the interactive goal grid, create goals, and add datapoints.
---

Running `buzz` with no arguments launches the interactive terminal UI: a colorful
goal grid you drive entirely from the keyboard.

## Navigation

| Key | Action |
| --- | --- |
| **Arrow keys** or **h j k l** | Navigate the goal grid spatially (vim-style) |
| **Page Up / Page Down** or **u / d** | Scroll when there are many goals |
| **/** | Enter search/filter mode |
| **n** | Create a new goal |
| **Escape** | Exit search mode or close modals |
| **Enter** | View goal details and add datapoints |
| **q** or **Ctrl+C** | Quit |

## The goal grid

Goals are displayed in a colorful grid based on their deadline urgency:

| Color | Meaning |
| --- | --- |
| **Red** | Overdue or due today (`safebuf < 1`) |
| **Orange** | Due within 1 day (`safebuf < 2`) |
| **Blue** | Due within 2 days (`safebuf < 3`) |
| **Green** | Due within 3–6 days (`safebuf < 7`) |
| **Gray** | Due in 7+ days |

## Creating goals

1. Press <kbd>n</kbd> to open the goal creation modal.
2. Fill in the required fields:
   - **Slug** — unique identifier for the goal (alphanumeric, dashes, underscores)
   - **Title** — display name for the goal
   - **Goal Type** — type of goal (e.g. hustler, biker, fatloser, gainer)
   - **Goal Units** — units for the goal (e.g. workouts, pages, pounds)
   - **Exactly 2 of 3** parameters: `goaldate`, `goalval`, `rate` (use "null" to skip one)
3. Use <kbd>Tab</kbd> / <kbd>Shift</kbd>+<kbd>Tab</kbd> to navigate between fields.
4. Press <kbd>Enter</kbd> to submit, or <kbd>Escape</kbd> to cancel.

## Adding datapoints

1. Navigate to a goal and press <kbd>Enter</kbd> to open its details.
2. Press <kbd>a</kbd> to enter datapoint input mode.
3. Use <kbd>Tab</kbd> / <kbd>Shift</kbd>+<kbd>Tab</kbd> to navigate between fields.
4. Press <kbd>Enter</kbd> to submit, or <kbd>Escape</kbd> to cancel.

## Filter / search

- Press <kbd>/</kbd> to enter filter mode.
- Type to fuzzy-search goals by slug or title. Characters must appear in order but
  don't need to be consecutive — e.g. "wk" matches "**w**or**k**out", "**w**al**k**".
- Press <kbd>Escape</kbd> to clear the filter and show all goals.

## Auto-refresh

- Press <kbd>t</kbd> to toggle auto-refresh (refreshes every 5 minutes).
- Press <kbd>r</kbd> to manually refresh goals.
- The TUI also refreshes automatically when you use
  [`buzz add`](/commands/managing/#buzz-add) in another terminal.

## Disabling colors

The TUI honors the global [`--no-color`](/commands/overview/#--no-color) flag, so
`buzz --no-color` launches the grid in plain text — handy for limited terminals,
screen readers, or accessibility tools.
