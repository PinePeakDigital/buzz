---
title: Commands overview
description: The buzz command-line interface and global flags.
---

Running `buzz` with no arguments launches the [interactive TUI](/guides/tui/).
Every common action is also available as a one-shot subcommand that prints plain
output and exits — ideal for scripts, cron jobs, and status bars.

## Command index

### [Viewing goals](/commands/viewing/)

| Command | Description |
| --- | --- |
| [`buzz next`](/commands/viewing/#buzz-next) | Terse summary of the next due goal |
| [`buzz list`](/commands/viewing/#buzz-list) | List all goals with a summary overview |
| [`buzz today`](/commands/viewing/#buzz-today) | All goals due today |
| [`buzz tomorrow`](/commands/viewing/#buzz-tomorrow) | All goals due tomorrow |
| [`buzz due`](/commands/viewing/#buzz-due) | Goals due within a duration you specify |
| [`buzz less`](/commands/viewing/#buzz-less) | All do-less type goals |
| [`buzz view`](/commands/viewing/#buzz-view) | Detailed information about a goal |
| [`buzz data`](/commands/viewing/#buzz-data) | List a goal's datapoints |
| [`buzz schedule`](/commands/viewing/#buzz-schedule) | Deadline distribution across a 24-hour day |
| [`buzz review`](/commands/viewing/#buzz-review) | Interactive review of all goals |

### [Managing goals](/commands/managing/)

| Command | Description |
| --- | --- |
| [`buzz add`](/commands/managing/#buzz-add) | Add a datapoint to a goal |
| [`buzz refresh`](/commands/managing/#buzz-refresh) | Refresh autodata for a goal |
| [`buzz charge`](/commands/managing/#buzz-charge) | Create a charge on your account |
| [`buzz deadline`](/commands/managing/#buzz-deadline) | Change a goal's deadline |
| [`buzz ratchet`](/commands/managing/#buzz-ratchet) | Remove safety buffer from a goal |
| [`buzz auth login`](/commands/managing/#buzz-auth-login) | Authenticate with Beeminder |

## Global flags

### `--no-color`

If you prefer plain text output without colors, use the global `--no-color` flag
with any command:

```bash
buzz --no-color next
buzz --no-color today
buzz --no-color view mygoal
buzz --no-color            # Even works with the TUI
```

The flag can be placed anywhere on the command line and works with all buzz
commands. This is useful for:

- Terminal environments with limited color support
- Scripts or automation where colored output is not desired
- Screen readers or accessibility tools
- Logging output to files

## Urgency colors

Commands that list goals color-code each one by deadline urgency, using the same
scheme as the TUI grid:

| Color | Meaning |
| --- | --- |
| **Red** | Overdue or due today (`safebuf < 1`) |
| **Orange** | Due within 1 day (`safebuf < 2`) |
| **Blue** | Due within 2 days (`safebuf < 3`) |
| **Green** | Due within 3–6 days (`safebuf < 7`) |
| **Gray** | Due in 7+ days |
