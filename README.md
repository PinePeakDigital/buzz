# buzz

Beeminder TUI - A terminal user interface built with Bubble Tea.

## Prerequisites

- Go 1.21 or later

## Building

To build the application:

```bash
go build
```

This will create a `buzz` executable in the current directory.

## Running

You can run the application in two ways:

### Run directly with Go:
```bash
go run main.go
```

### Run the compiled binary:
```bash
./buzz
```

## Authentication

On first run, you'll be prompted to authenticate with Beeminder:

1. Visit https://www.beeminder.com/api/v1/auth_token.json to get your credentials
2. Copy the JSON output (format: `{"username":"your_username","auth_token":"your_token"}`)
3. Paste it into the application when prompted
4. Press Enter to save

Your credentials will be stored securely in `~/.buzzrc` with read/write permissions for your user only.

On subsequent runs, the application will automatically load your saved credentials.

## Usage

- Use **arrow keys** or **j/k** (vim-style) to navigate
- Press **Enter** or **Space** to select/deselect items
- Press **q** or **Ctrl+C** to quit

## Development

To install dependencies:

```bash
go mod tidy
```
