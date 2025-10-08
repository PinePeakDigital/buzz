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

The project has comprehensive test coverage for business logic and utility functions. See [docs/TESTING.md](docs/TESTING.md) for complete testing documentation.

### Quick Start

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

### Test Files

- `beeminder_test.go` - Tests for Beeminder API functions
- `handlers_test.go` - Tests for input validation and handlers
- `utils_test.go` - Tests for utility functions
- `config_test.go` - Tests for configuration management
- `model_test.go` - Tests for application state and models

**Coverage:** 18.5% of statements (178 test cases)

For detailed information about testing strategy, coverage, and best practices, see [docs/TESTING.md](docs/TESTING.md).

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
