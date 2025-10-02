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

## Usage

- Use **arrow keys** or **j/k** (vim-style) to navigate
- Press **Enter** or **Space** to select/deselect items
- Press **q** or **Ctrl+C** to quit

## Development

To install dependencies:

```bash
go mod tidy
```
