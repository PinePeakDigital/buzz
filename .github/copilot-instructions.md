# GitHub Copilot Instructions

## Commit Message Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages. When suggesting commit messages, always follow this format:

```
type(scope): description
```

### Valid Commit Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code (formatting, whitespace, etc.)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools
- `perf`: A code change that improves performance
- `ci`: Changes to CI configuration files and scripts
- `build`: Changes that affect the build system or external dependencies
- `revert`: Reverts a previous commit

### Scope (Optional)

The scope is optional and should be the name of the affected module, component, or area of the codebase (e.g., `api`, `db`, `auth`, `ui`).

### Examples

Good commit messages:
```
feat: add user authentication system
fix(api): handle nil pointer in user service
docs: update README with installation steps
refactor(db): simplify connection pooling
test: add tests for beeminder API integration
chore: update dependencies
perf(grid): optimize goal rendering
ci: add linting to workflow
```

### Notes

- The description should be in lowercase and start with a verb in imperative mood
- Keep the description concise but descriptive
- A commit-msg hook enforces this format, so all commits must comply
- See `DEVELOPMENT.md` for more details on our git workflow

## Accessing CodeRabbit PR Feedback

When working on a PR branch, you can access all CodeRabbit feedback using the provided shell script:

```bash
./scripts/get-coderabbit-feedback.sh
```

### What the Script Retrieves

The script fetches comprehensive CodeRabbit feedback from three sources:

1. **General PR Comments** - Timeline comments posted by CodeRabbit on the PR conversation
2. **Inline Review Comments** - Code-specific comments attached to particular lines in the diff
3. **Review Summaries** - Overall review summaries submitted by CodeRabbit

### Usage

Make sure you're on a PR branch before running the script:

```bash
git checkout <pr-branch>
./scripts/get-coderabbit-feedback.sh
```

The script will automatically detect the current PR and display all CodeRabbit feedback in a structured, readable format.

### Requirements

- GitHub CLI (`gh`) must be installed and authenticated
- Must be run from within a PR branch context
