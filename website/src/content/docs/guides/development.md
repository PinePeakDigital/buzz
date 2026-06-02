---
title: Development
description: Set up a local development environment and contribute to buzz.
---

buzz is written in Go. This page covers building from source, running tests, and
the project's contribution conventions.

## Prerequisites

- Go 1.24 or later

## Building from source

Build the application locally:

```bash
go build
```

This creates a `buzz` executable in the current directory.

Run it in development mode without building first:

```bash
go run main.go
```

Install and update dependencies:

```bash
go mod tidy
```

## Testing

The project has test coverage for business logic and utility functions.

```bash
go test          # Run all tests
go test -v       # Verbose output
go test -cover   # With coverage
```

Key test files include `beeminder_test.go` (API functions), `handlers_test.go`
(input validation and handlers), `utils_test.go` (utilities), `config_test.go`
(configuration), and `model_test.go` (application state and models). See
`docs/TESTING.md` in the repository for the full testing strategy.

## Project structure

| File | Responsibility |
| --- | --- |
| `main.go` | Entry point and Bubble Tea orchestration |
| `model.go` | Application state models and initialization |
| `handlers.go` | Keyboard input handlers |
| `grid.go` | Grid rendering and modal UI |
| `styles.go` | Lipgloss styling definitions |
| `beeminder.go` | Beeminder API integration |
| `auth.go` | Authentication handling |
| `config.go` | Configuration management |
| `messages.go` | Bubble Tea commands and messages |
| `utils.go` | Helper functions |

## Git hooks

The project uses [Conventional Commits](https://www.conventionalcommits.org/) for
consistent commit messages. Install the commit-msg hook that enforces this format:

```bash
./scripts/setup-hooks.sh
```

### Valid commit types

`feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `ci`,
`build`, and `revert`.

```text
feat: add user authentication system
fix(api): handle nil pointer in user service
docs: update README with installation steps
refactor(db): simplify connection pooling
```

## Release process

Releases are automated based on conventional commits. When changes are merged to
`main`, the Release workflow:

1. Analyzes commit messages for conventional commit types
2. Calculates the next semantic version (patch, minor, or major)
3. Builds cross-platform binaries
4. Creates a GitHub release with the new version tag

### Manual releases

You can also trigger a release with a specific version:

1. Go to the
   [Actions tab](https://github.com/PinePeakDigital/buzz/actions/workflows/release.yml).
2. Click **Run workflow**.
3. Optionally enter a version override (e.g. `v1.2.3`). If left empty, the version
   is auto-calculated from conventional commits.
4. Click **Run workflow** to start the release.

This is useful for hotfix releases, manually controlling version numbers, or
releasing when conventional commits aren't sufficient.

## Contributing

1. Fork the repository.
2. Create a feature branch.
3. Make your changes.
4. Ensure commit messages follow the Conventional Commits format.
5. Submit a pull request.

## Documentation site

This documentation site lives in the [`website/`](https://github.com/PinePeakDigital/buzz/tree/main/website)
directory of the repository and is built with
[Starlight](https://starlight.astro.build/). To work on the docs locally:

```bash
cd website
pnpm install
pnpm dev      # Start the dev server at http://localhost:4321
pnpm build    # Build the production site into website/dist
```

The site auto-deploys to <https://buzz.nathanarthur.com> on every push to `main`.
