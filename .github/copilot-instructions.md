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

When working on a PR branch, you can access all CodeRabbit feedback using the GitHub MCP tools available to Copilot. The GitHub MCP provides authenticated access to GitHub's API without requiring token configuration.

### What Feedback is Available

CodeRabbit provides feedback in three locations:

1. **General PR Comments** - Timeline comments posted by CodeRabbit on the PR conversation
2. **Inline Review Comments** - Code-specific comments attached to particular lines in the diff
3. **Review Summaries** - Overall review summaries submitted by CodeRabbit

### GitHub MCP Tools to Use

Use the following MCP tools in sequence to retrieve all CodeRabbit feedback:

#### Step 1: Find the Current PR

```
github-mcp-server-list_pull_requests
  owner: <repo-owner>
  repo: <repo-name>
  state: open
```

This returns a list of open PRs. Identify the current PR by matching the branch name.

#### Step 2: Get General PR Comments

```
github-mcp-server-get_issue_comments
  owner: <repo-owner>
  repo: <repo-name>
  issue_number: <pr-number>
```

Filter the results for comments where `author.login` is `"coderabbitai[bot]"` or `"coderabbitai"`.

#### Step 3: Get Inline Review Comments

```
github-mcp-server-get_pull_request_review_comments
  owner: <repo-owner>
  repo: <repo-name>
  pullNumber: <pr-number>
```

Filter the results for comments where `user.login` is `"coderabbitai[bot]"` or `"coderabbitai"`.

#### Step 4: Get Review Summaries

```
github-mcp-server-get_pull_request_reviews
  owner: <repo-owner>
  repo: <repo-name>
  pullNumber: <pr-number>
```

Filter the results for reviews where `user.login` is `"coderabbitai[bot]"` or `"coderabbitai"`.

### Example Usage

For the repository `narthur/buzz` with PR #97:

1. List PRs to find current PR number
2. Get issue comments: `issue_number: 97`
3. Get review comments: `pullNumber: 97`
4. Get reviews: `pullNumber: 97`

### Benefits of GitHub MCP Approach

- ✅ **Built-in authentication** - No token configuration needed
- ✅ **Always available** - Works in Copilot environment without additional setup
- ✅ **Type-safe** - Structured data from API
- ✅ **Complete coverage** - Accesses all three types of CodeRabbit feedback
- ✅ **Pagination handled** - MCP tools handle pagination automatically
