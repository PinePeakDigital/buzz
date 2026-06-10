# buzz

A terminal user interface for [Beeminder](https://beeminder.com) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

View your goals in a colorful grid, navigate with arrow keys, and add datapoints directly from your terminal.

![buzz demo](scripts/demo/demo.gif)

> The demo above is generated automatically from fictional data — see [`scripts/demo/`](scripts/demo/).

📖 **Full documentation:** [buzz.nathanarthur.com](https://buzz.nathanarthur.com)

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

## Usage

Run `buzz` with no arguments to launch the interactive TUI, or use a subcommand such as `buzz today` or `buzz add` for one-shot, scriptable output.

The full reference — authentication, configuration, every command, and TUI navigation — lives in the documentation:

👉 **[buzz.nathanarthur.com](https://buzz.nathanarthur.com)**

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and contribution guidelines.
