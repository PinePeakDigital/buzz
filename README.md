# buzz

A terminal user interface for [Beeminder](https://beeminder.com) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

View your goals in a colorful grid, navigate with arrow keys, and add datapoints directly from your terminal.

## Installation

### Using bin (Recommended)

Install using [bin](https://github.com/marcosnils/bin):

```bash
bin install https://github.com/narthur/buzz
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

You can also download pre-built binaries directly from the [releases page](https://github.com/narthur/buzz/releases). Choose the appropriate binary for your operating system and architecture.

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
go install github.com/narthur/buzz@latest
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
```

The comment parameter is optional and defaults to "Added via buzz" if not provided.

Running `buzz` without arguments launches the interactive TUI.

### Navigation
- **Arrow keys** or **hjkl** (vim-style) - Navigate the goal grid spatially
- **Page Up/Down** or **u/d** - Scroll when there are many goals
- **/** - Enter search/filter mode
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

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and contribution guidelines.
