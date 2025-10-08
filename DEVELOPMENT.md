# Development Setup

## Prerequisites

- Go 1.21 or later

## Building from Source

To build the application locally:

```bash
go build
```

This will create a `buzz` executable in the current directory.

## Running in Development

You can run the application in development mode:

```bash
go run main.go
```

## Dependencies

To install and update dependencies:

```bash
go mod tidy
```

## Testing

### Running Tests

Run all tests:

```bash
go test
```

Run tests with verbose output:

```bash
go test -v
```

Run tests with coverage:

```bash
go test -cover
```

### Test Organization

Tests are organized alongside their source files:

- `beeminder_test.go` - Tests for Beeminder API functions
- `handlers_test.go` - Tests for input validation and handlers
- `utils_test.go` - Tests for utility functions
- `config_test.go` - Tests for configuration management
- `model_test.go` - Tests for application state and models

### Writing Tests

When adding new functionality:

1. Write tests for pure functions first (functions with no side effects)
2. Use table-driven tests for comprehensive coverage
3. Test edge cases and error conditions
4. Follow Go testing conventions and naming patterns

Example test structure:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"basic case", "input", "expected"},
        {"edge case", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### CI/CD

Tests are automatically run in CI/CD pipelines:
- On every pull request
- Before building cross-platform binaries

## Project Structure

- `main.go` - Main application entry point and Bubble Tea orchestration
- `model.go` - Application state models and initialization
- `handlers.go` - Keyboard input handlers
- `grid.go` - Grid rendering and modal UI
- `styles.go` - Lipgloss styling definitions
- `beeminder.go` - Beeminder API integration
- `auth.go` - Authentication handling
- `config.go` - Configuration management
- `messages.go` - Bubble Tea commands and messages
- `utils.go` - Helper functions

## Git Hooks Setup

This project uses conventional commits for consistent commit messaging. To set up the git hooks that enforce this format:

```bash
./scripts/setup-hooks.sh
```

This will install a commit-msg hook that validates commit messages against the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Valid Commit Formats

- `feat: description` - A new feature
- `fix: description` - A bug fix  
- `docs: description` - Documentation changes
- `style: description` - Code style changes (formatting, etc.)
- `refactor: description` - Code changes that neither fix bugs nor add features
- `test: description` - Adding or updating tests
- `chore: description` - Build process or auxiliary tool changes
- `perf: description` - Performance improvements
- `ci: description` - CI configuration changes
- `build: description` - Build system changes
- `revert: description` - Reverts a previous commit

### Examples

```
feat: add user authentication system
fix(api): handle nil pointer in user service  
docs: update README with installation steps
refactor(db): simplify connection pooling
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Ensure commit messages follow conventional commits format
5. Submit a pull request
