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

#### macOS Users

If you download the binary directly from GitHub releases, macOS may show an "unidentified developer" warning. To resolve this, remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine /path/to/buzz
```

Replace `/path/to/buzz` with the actual path to the downloaded binary.

**Note:** This workaround is only needed for direct downloads. If you install via `bin` (recommended) or build from source, you won't encounter this issue.

### From Source

If you have Go installed:

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

**buzz add** - Add a datapoint to a goal without opening the TUI:

```bash
buzz add <goalslug> <value> [comment]

# Examples:
buzz add opsec 1                    # Adds value 1 with default comment "Added via buzz"
buzz add workout 2.5 'morning run'  # Adds value 2.5 with custom comment
buzz add study 00:05 'quick review' # Adds 5 minutes (converted to 0.083333 hours)
buzz add focus 1:30                 # Adds 1.5 hours (1 hour 30 minutes)
```

The `<value>` parameter supports both decimal numbers and time formats:
- **Decimal numbers**: `1`, `2.5`, `-1.5`
- **Time format (HH:MM)**: `00:05` (5 minutes), `1:30` (1.5 hours), `2:45` (2.75 hours)
- **Time format (HH:MM:SS)**: `1:30:45` (1.5125 hours)

Time formats are automatically converted to decimal hours before submitting to Beeminder.

The comment parameter is optional and defaults to "Added via buzz" if not provided.

**Note:** When you run `buzz add` while the TUI is running in another terminal, the TUI will automatically refresh within 1 second to show the new datapoint.

**buzz refresh** - Refresh autodata for a goal:

```bash
buzz refresh <goalslug>

# Example:
buzz refresh fitbit    # Forces Beeminder to fetch latest data from Fitbit
```

This command is analogous to pressing the refresh button on a goal's page in the Beeminder web interface. It forces Beeminder to refetch autodata for goals with automatic data sources (like Fitbit, GitHub, etc.) and refreshes the graph image. This is useful when you want to immediately update a goal's data instead of waiting for Beeminder's automatic sync.

**Note:** This is an asynchronous operation. The command returns immediately after the goal is queued for refresh. It may take a few moments for Beeminder to actually fetch the new data.

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
- **Red** - Overdue or due today
- **Orange** - Due within 1-2 days  
- **Blue** - Due within 3-6 days
- **Green** - Due within 7+ days
- **Gray** - Far future deadlines

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

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and contribution guidelines.
