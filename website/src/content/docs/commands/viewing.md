---
title: Viewing goals
description: Commands for inspecting and reviewing your Beeminder goals.
---

These commands read your goals and print them to the terminal. None of them
modify anything on Beeminder. All of them respect the [`--no-color`](/commands/overview/#--no-color)
flag and the [urgency color scheme](/commands/overview/#urgency-colors).

## `buzz next`

Output a terse summary of the next due goal:

```bash
buzz next
# Example output: p3 +1 in 0 days 2h
```

The output format is `goalslug limsum timeframe`:

- **`goalslug`** — the goal's slug/identifier
- **`limsum`** — a summary of what you need to do (e.g. "+1 in 0 days", "+2 within 1 day")
- **`timeframe`** — time until the goal is due (e.g. "2h" for 2 hours, "3d" for 3 days)

You can also run `buzz next` in watch mode to continuously monitor your next goal:

```bash
buzz next --watch    # Refreshes every 5 minutes
buzz next -w         # Shorthand for --watch
```

In watch mode the display updates automatically and shows a timestamp. Press
<kbd>Ctrl</kbd>+<kbd>C</kbd> to exit.

## `buzz list`

List all goals with a summary overview:

```bash
buzz list
# Example output:
# Total goals: 3
#
# Slug      Title           Units     Rate     Stakes
# --------  --------------  --------  -------  ------
# coding    Code Daily      lines     5/d      $10.00
# exercise  Daily Exercise  workouts  1/d      $5.00
# reading   Read Books      pages     -        $0.00
```

Displays a table with the following information for each goal:

- **Slug** — the goal's unique identifier
- **Title** — the goal's display name (or "-" if not set)
- **Units** — the unit being measured (e.g. "hours", "pages", "workouts", or "-" if not set)
- **Rate** — the commitment rate (e.g. "5/d" for 5 per day, "1/w" for 1 per week, or "-" if not set)
- **Stakes** — the current pledge amount

Goals are sorted alphabetically by slug, and the total number of goals is shown
at the top. This is useful for a quick overview of all your goals without
focusing on deadlines.

## `buzz today`

Output all goals due today:

```bash
buzz today
# Example output:
# exercise  +1 in 0 days  5h       5:30 PM
# reading   +1 in 0 days  8h       8:45 PM
# water     +3 in 0 days  10h      11:00 PM
```

Goals are shown in a table with columns aligned for easy scanning, sorted by due
date, then by stakes, then by name. Each row includes the goal slug, the amount
needed (delta value), the relative deadline (time remaining), and the absolute
deadline (date and time).

## `buzz tomorrow`

Output all goals due tomorrow:

```bash
buzz tomorrow
# Example output:
# coding    +2 within 1 day  1d   tomorrow 2:30 PM
# writing   +1 within 1 day  1d   tomorrow 5:45 PM
```

Shows all goals due tomorrow in the same format as [`buzz today`](#buzz-today).

## `buzz due`

Output all goals due within a duration you specify:

```bash
buzz due <duration>

# Examples:
buzz due 10m    # Goals due within the next 10 minutes
buzz due 1h     # Goals due within the next hour
buzz due 5d     # Goals due within the next 5 days
buzz due 1w     # Goals due within the next week
buzz due 2w     # Goals due within the next 2 weeks
```

Supported duration units:

- **`m`** or **`M`** — minutes (e.g. `10m`, `30m`, `0.5m`)
- **`h`** or **`H`** — hours (e.g. `1h`, `24h`, `0.5h`)
- **`d`** or **`D`** — days (e.g. `1d`, `5d`, `7d`)
- **`w`** or **`W`** — weeks (e.g. `1w`, `2w`)

Displays goals in the same table format as [`buzz today`](#buzz-today), including
overdue goals (those past their deadline). Useful for planning ahead and seeing
what's coming up in a custom time window.

## `buzz less`

Output all do-less type goals:

```bash
buzz less
# Example output:
# junkfood       -1 by 3pm   6h
# procrastinate  -2 by 5pm   8h
```

Lists all goals where you're trying to do *less* of something (weight loss, habit
breaking, etc.). Useful for reviewing negative goals separately from positive ones.

## `buzz view`

View detailed information about a specific goal:

```bash
buzz view <goalslug>

# Example:
buzz view exercise
# Output:
# Goal: exercise
# Title:       Daily Exercise
# Fine print:  At least 30 minutes
# Limsum:      +1 in 0 days
# Pledge:      $5.00
# Autodata:    none
# URL:         https://www.beeminder.com/username/exercise
```

Additional options:

- **`--web`** — open the goal in your default web browser
- **`--json`** — output goal data as JSON
- **`--datapoints`** — include datapoints in the JSON output (use with `--json`)

```bash
buzz view exercise --web               # Opens goal in browser
buzz view exercise --json              # Output as JSON
buzz view exercise --json --datapoints # JSON with datapoints included
```

## `buzz data`

List a goal's datapoints in chronological order (oldest first):

```bash
buzz data <goalslug>

# Example:
buzz data exercise
# Output:
# 2024-01-01   3
# 2024-01-02   12.5   morning run
```

Each line shows the datapoint's date, value, and comment (comments are omitted
when empty). Dates come from Beeminder's daystamp, so they match the day the
datapoint counts toward regardless of your timezone.

Sorting defaults to oldest-first so the newest datapoints land at the bottom of
an unpaged dump. Use `--desc` for newest-first (or `--asc` to be explicit); the
two flags are mutually exclusive:

```bash
buzz data exercise --desc   # newest datapoint first
buzz data exercise --asc    # oldest first (same as the default)
```

## `buzz schedule`

Display the distribution of goal deadlines throughout a 24-hour day:

```bash
buzz schedule
```

Shows a visual representation of how all goal deadlines are distributed across a
24-hour day, regardless of when they're actually due. The output has two parts:

1. **Hourly density overview** — a compact bar chart showing the number of goals
   per hour across the day
2. **Detailed timeline** — a vertical timeline listing all goals grouped by their
   deadline time

```text
HOURLY DENSITY
                  ▁▁          ██    ▁▁       ▃▃       ▁▁          ▄▄
00 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20 21 22 23
┴──┴──┴──┴──┴──┴──┼──┴──┴──┴──┼──┴──┼──┴──┴──┼──┴──┴──┼──┴──┴──┴──┼──┴
                  1           5     1        2        1           3

TIMELINE
────────────────────────────────────────────────
06:00 ├─ wake_up
10:30 ├─ exercise, vitamins, breakfast, meditation, journal
12:00 ├─ lunch
15:00 ├─ afternoon_walk, water_check
18:00 ├─ dinner_prep
22:00 ├─ bedtime_routine, reading, evening_review
```

The command extracts the time-of-day from all goal deadlines (ignoring the date)
and groups goals by their exact deadline time. Useful for:

- Identifying the busiest hours of your day
- Spotting scheduling imbalances and bottlenecks
- Understanding your goal distribution patterns
- Planning better time allocation

The visualization uses ASCII characters that work well even with colors disabled
(`--no-color`).

## `buzz review`

Launch an interactive review of all your goals:

```bash
buzz review
```

Displays one goal at a time, allowing you to review all your goals in detail.
Goals are sorted alphabetically by slug.

**Features:**

- View detailed information about each goal (slug, rate, current value, buffer,
  pledge, due date)
- See your progress with a goal counter (e.g. "Goal 1 of 10")
- Goals are color-coded by urgency (same as the main TUI)
- Navigate with the keyboard:
  - **Next goal:** <kbd>→</kbd>, <kbd>l</kbd>, <kbd>n</kbd>, or <kbd>j</kbd>
  - **Previous goal:** <kbd>←</kbd>, <kbd>h</kbd>, <kbd>p</kbd>, or <kbd>k</kbd>
  - **Open in browser:** <kbd>o</kbd> or <kbd>Enter</kbd>
  - **Quit:** <kbd>q</kbd> or <kbd>Esc</kbd>
