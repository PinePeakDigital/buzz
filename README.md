# buzz

A terminal user interface for [Beeminder](https://beeminder.com) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

View your goals in a colorful grid, navigate with arrow keys, and add datapoints directly from your terminal.

## Installation

### Using bin (Recommended)

Install using [bin](https://github.com/marcosnils/bin):

```bash
bin install https://github.com/pinepeakdigital/buzz
```

This will download the latest release and make it available in your PATH.

To update:

```bash
bin update buzz
```

### Homebrew

Install using [Homebrew](https://brew.sh):

```bash
brew tap narthur/tap
brew install narthur/tap/buzz
```

To upgrade:

```bash
brew upgrade narthur/tap/buzz
```

### Direct Download

You can also download pre-built binaries directly from the [releases page](https://github.com/pinepeakdigital/buzz/releases). Choose the appropriate binary for your operating system and architecture.

To update, download the latest release from the releases page and replace your existing binary.

#### macOS Users

If you download the binary directly from GitHub releases, macOS may show an "unidentified developer" warning. To resolve this, remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine /path/to/buzz
```

Replace `/path/to/buzz` with the actual path to the downloaded binary.

**Note:** This workaround is only needed for direct downloads. If you install via `bin` (recommended) or build from source, you won't encounter this issue.

### From Source

If you have Go installed, you can install and update using the same command:

```bash
go install github.com/pinepeakdigital/buzz@latest
```

## Authentication

On first run, you'll be prompted to authenticate with Beeminder:

1. Visit https://www.beeminder.com/api/v1/auth_token.json to get your credentials
2. Copy the JSON output (format: `{"username":"your_username","auth_token":"your_token"}`)
3. Paste it into the application when prompted
4. Press Enter to save

Your credentials will be stored securely in `~/.buzzrc` with read/write permissions for your user only.

## Configuration

### Logging (Optional)

Buzz supports optional logging of HTTP requests and responses to help with debugging and monitoring API calls. Logging is **disabled by default** to respect user preferences and avoid cluttering your filesystem.

To enable logging, edit your `~/.buzzrc` file and add a `log_file` field:

```json
{
  "username": "your_username",
  "auth_token": "your_token",
  "log_file": "/path/to/buzz.log"
}
```

When logging is enabled, buzz will append timestamped entries for each HTTP request and response to the specified log file:

```
[2025-12-09 12:34:56] REQUEST: GET https://www.beeminder.com/api/v1/users/alice/goals.json?auth_token=...
[2025-12-09 12:34:57] RESPONSE: 200 https://www.beeminder.com/api/v1/users/alice/goals.json?auth_token=...
```

To disable logging, simply remove the `log_file` field from your `~/.buzzrc` or set it to an empty string.

## Usage

### Command Line Interface

**buzz next** - Output a terse summary of the next due goal:
```bash
buzz next
# Example output: p3 +1 in 0 days 2h
```

The output format is: `goalslug limsum timeframe`
- `goalslug`: The goal's slug/identifier
- `limsum`: Summary of what you need to do (e.g., "+1 in 0 days", "+2 within 1 day")
- `timeframe`: Time until the goal is due (e.g., "2h" for 2 hours, "3d" for 3 days)

You can also run `buzz next` in watch mode to continuously monitor your next goal:

```bash
buzz next --watch    # Refreshes every 5 minutes
buzz next -w         # Shorthand for --watch
```

In watch mode, the display updates automatically and shows a timestamp. Press Ctrl+C to exit.

**buzz list** - List all goals with a summary overview:

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
- **Slug** - The goal's unique identifier
- **Title** - The goal's display name (or "-" if not set)
- **Units** - The unit being measured (e.g., "hours", "pages", "workouts", or "-" if not set)
- **Rate** - The commitment rate (e.g., "5/d" for 5 per day, "1/w" for 1 per week, or "-" if not set)
- **Stakes** - The current pledge amount

Goals are sorted alphabetically by slug. The command also shows the total number of goals at the top. This command is useful for getting a quick overview of all your goals without focusing on deadlines.

**buzz today** - Output all goals due today:

```bash
buzz today
# Example output:
# exercise  +1 in 0 days  5h       5:30 PM
# reading   +1 in 0 days  8h       8:45 PM
# water     +3 in 0 days  10h      11:00 PM
```

Displays goals in a table format with columns aligned for easy scanning. Goals are sorted by due date, then by stakes, then by name. The output includes:
- Goal slug (name)
- Amount needed (delta value)
- Relative deadline (time remaining)
- Absolute deadline (date and time)

Goals are color-coded by urgency (same as the TUI grid):
- **Red** - Overdue or due today (safebuf < 1)
- **Orange** - Due within 1 day (safebuf < 2)
- **Blue** - Due within 2 days (safebuf < 3)
- **Green** - Due within 3-6 days (safebuf < 7)
- **Gray** - Due in 7+ days

**buzz tomorrow** - Output all goals due tomorrow:

```bash
buzz tomorrow
# Example output:
# coding    +2 within 1 day  1d   tomorrow 2:30 PM
# writing   +1 within 1 day  1d   tomorrow 5:45 PM
```

Shows all goals that are due tomorrow in the same format as `buzz today`, with color coding based on urgency.

**buzz due** - Output all goals due within a specified duration:

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
- `m` or `M` - minutes (e.g., `10m`, `30m`, `0.5m`)
- `h` or `H` - hours (e.g., `1h`, `24h`, `0.5h`)
- `d` or `D` - days (e.g., `1d`, `5d`, `7d`)
- `w` or `W` - weeks (e.g., `1w`, `2w`)

Displays goals in the same table format as `buzz today` and `buzz tomorrow`, with color coding based on urgency. Includes overdue goals (those past their deadline) in the results. This command is useful for planning ahead and seeing what goals are coming up in a custom time window.

**buzz less** - Output all do-less type goals:

```bash
buzz less
# Example output:
# junkfood   -1 by 3pm      6h
# procrastinate  -2 by 5pm  8h
```

Lists all goals where you're trying to do less of something (weight loss, habit breaking, etc.), with color coding based on urgency. Useful for reviewing negative goals separately from positive ones.

**buzz add** - Add a datapoint to a goal without opening the TUI:

```bash
buzz add <goalslug> <value> [comment] [--requestid=<id>]

# Examples:
buzz add opsec 1                    # Adds value 1 with default comment "Added via buzz"
buzz add workout 2.5 'morning run'  # Adds value 2.5 with custom comment
buzz add study 00:05 'quick review' # Adds 5 minutes (converted to 0.083333 hours)
buzz add focus 1:30                 # Adds 1.5 hours (1 hour 30 minutes)
buzz add reading 3 'finished chapter 5' --requestid=abc123  # Adds with a request ID for idempotency
```

The `<value>` parameter supports both decimal numbers and time formats:
- **Decimal numbers**: `1`, `2.5`, `-1.5`
- **Time format (HH:MM)**: `00:05` (5 minutes), `1:30` (1.5 hours), `2:45` (2.75 hours)
- **Time format (HH:MM:SS)**: `1:30:45` (1.5125 hours)

Time formats are automatically converted to decimal hours before submitting to Beeminder.

The comment parameter is optional and defaults to "Added via buzz" if not provided.

The `--requestid` flag is optional and provides idempotency:
- **Prevents duplicates**: Safely retry submissions without creating duplicate datapoints
- **Updates existing**: If a datapoint with the same requestid exists but differs, it gets updated
- **Scoped per goal**: The same requestid can be used across different goals

**Note:** When you run `buzz add` while the TUI is running in another terminal, the TUI will automatically refresh within 1 second to show the new datapoint.

**buzz refresh** - Refresh autodata for a goal:

```bash
buzz refresh <goalslug>

# Example:
buzz refresh fitbit    # Forces Beeminder to fetch latest data from Fitbit
```

This command is analogous to pressing the refresh button on a goal's page in the Beeminder web interface. It forces Beeminder to refetch autodata for goals with automatic data sources (like Fitbit, GitHub, etc.) and refreshes the graph image. This is useful when you want to immediately update a goal's data instead of waiting for Beeminder's automatic sync.

**Note:** This is an asynchronous operation. The command returns immediately after the goal is queued for refresh. It may take a few moments for Beeminder to actually fetch the new data.

**buzz view** - View detailed information about a specific goal:

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
- `--web` - Open the goal in your default web browser
- `--json` - Output goal data as JSON
- `--datapoints` - Include datapoints in JSON output (use with `--json`)

```bash
buzz view exercise --web              # Opens goal in browser
buzz view exercise --json             # Output as JSON
buzz view exercise --json --datapoints # JSON with datapoints included
```

**buzz charge** - Create a charge for the authenticated user:

```bash
buzz charge <amount> <note> [--dryrun]

# Examples:
buzz charge 10 "Intentional charge for motivation"
buzz charge 5.50 "Weekly commitment fee" --dryrun  # Test without actually charging
```

Creates a charge on your Beeminder account. This is useful for self-imposed penalties or commitment fees. The minimum charge amount is $1.00.

- `<amount>`: The amount to charge (must be >= 1.00)
- `<note>`: A description of what the charge is for (required)
- `--dryrun`: Test the charge without actually creating it (optional)

**Note:** This creates a real charge on your payment method unless you use the `--dryrun` flag.

**buzz schedule** - Display goal deadline distribution throughout a 24-hour day:

```bash
buzz schedule
```

Shows a visual representation of how all goal deadlines are distributed throughout a 24-hour day, regardless of when they're actually due. This helps you identify scheduling patterns and bottlenecks in your goal deadlines.

The output consists of two parts:

1. **Hourly Density Overview** - A compact bar chart showing the number of goals per hour across the 24-hour day
2. **Detailed Timeline** - A vertical timeline listing all goals grouped by their deadline time

Example output:
```
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

This command extracts the time-of-day from all goal deadlines (ignoring the date) and groups goals by their exact deadline time. This is useful for:
- Identifying the busiest hours of your day
- Spotting scheduling imbalances and bottlenecks
- Understanding your goal distribution patterns
- Planning better time allocation

The visualization uses ASCII characters that work well even with colors disabled (`--no-color` flag).

**buzz review** - Interactive review of all goals:

```bash
buzz review
```

This launches an interactive interface that displays one goal at a time, allowing you to review all your goals in detail. The goals are sorted alphabetically by slug.

**Features:**
- View detailed information about each goal (slug, rate, current value, buffer, pledge, due date)
- Navigate between goals with keyboard shortcuts:
  - **Next goal:** `→`, `l`, `n`, or `j`
  - **Previous goal:** `←`, `h`, `p`, or `k`
  - **Open in browser:** `o` or `Enter`
  - **Quit:** `q` or `Esc`
- See your progress through the list with a goal counter (e.g., "Goal 1 of 10")
- Goals are color-coded based on urgency (same as the main TUI)

Running `buzz` without arguments launches the interactive TUI.

### Navigation
- **Arrow keys** or **hjkl** (vim-style) - Navigate the goal grid spatially
- **Page Up/Down** or **u/d** - Scroll when there are many goals
- **/** - Enter search/filter mode
- **n** - Create a new goal
- **Escape** - Exit search mode or close modals
- **Enter** - View goal details and add datapoints
- **q** or **Ctrl+C** - Quit

### Goal Grid
Goals are displayed in a colorful grid based on their deadline urgency:
- **Red** - Overdue or due today (safebuf < 1)
- **Orange** - Due within 1 day (safebuf < 2)
- **Blue** - Due within 2 days (safebuf < 3)
- **Green** - Due within 3-6 days (safebuf < 7)
- **Gray** - Due in 7+ days

### Creating Goals
1. Press **n** to open the goal creation modal
2. Fill in the required fields:
   - **Slug** - Unique identifier for the goal (alphanumeric, dashes, underscores)
   - **Title** - Display name for the goal
   - **Goal Type** - Type of goal (e.g., hustler, biker, fatloser, gainer)
   - **Goal Units** - Units for the goal (e.g., workouts, pages, pounds)
   - **Exactly 2 of 3** parameters: goaldate, goalval, rate (use "null" to skip one)
3. Use **Tab/Shift+Tab** to navigate between fields
4. **Enter** to submit, **Escape** to cancel

### Adding Datapoints
1. Navigate to a goal and press **Enter** to open details
2. Press **a** to enter datapoint input mode
3. Use **Tab/Shift+Tab** to navigate between fields
4. **Enter** to submit, **Escape** to cancel

### Filter/Search
- Press **/** to enter filter mode
- Type to fuzzy search goals by slug or title
- Characters must appear in order but don't need to be consecutive
- Example: "wk" matches "workout", "wor**k**", or "**w**al**k**"
- Press **Escape** to clear the filter and show all goals

### Auto-refresh
- Press **t** to toggle auto-refresh (refreshes every 5 minutes)
- Press **r** to manually refresh goals
- The TUI also automatically refreshes when you use `buzz add` in another terminal

### Disabling Colors

If you prefer plain text output without colors, you can use the global `--no-color` flag with any command:

```bash
buzz --no-color next
buzz --no-color today
buzz --no-color view mygoal
buzz --no-color            # Even works with the TUI
```

The `--no-color` flag can be placed anywhere in the command line and works with all buzz commands. This is useful for:
- Terminal environments with limited color support
- Scripts or automation where colored output is not desired
- Screen readers or accessibility tools
- Logging output to files

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and contribution guidelines.
