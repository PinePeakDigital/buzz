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

### Navigation
- **Arrow keys** or **hjkl** (vim-style) - Navigate the goal grid spatially
- **Page Up/Down** or **u/d** - Scroll when there are many goals
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

### Auto-refresh
- Press **t** to toggle auto-refresh (refreshes every 5 minutes)
- Press **r** to manually refresh goals

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and contribution guidelines.
