# Development Setup

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
