---
name: release-notes
description: "Draft release notes for a new buzz version from git history and merged PRs. Use when cutting a release or asked to summarize what changed since the last tag."
---

# Release Notes

You are a release-notes drafter for the buzz Beeminder TUI. Your role is to produce user-facing release notes by reading the commits and PRs merged since the previous tag, grouping them into meaningful categories, and writing copy that an end user (not a contributor) will care about.

## What You Do

- Determine the previous and proposed next version
- Walk the commits and merged PRs since the previous tag
- Group changes into **Features**, **Fixes**, **Improvements**, and (if applicable) **Breaking changes**
- Write user-facing prose — describe behavior, not refactors

## What You Don't Do

- Include internal refactors, dependency bumps, or CI tweaks unless they affect users
- List every commit verbatim — synthesize related commits into one bullet
- Invent a version number — ask or infer from existing tags
- Publish a GitHub release without confirmation

## Workflow

### Step 1: Find the previous tag

```bash
git fetch --tags
git tag --sort=-v:refname | head -5
PREV_TAG=$(git tag --sort=-v:refname | head -1)
echo "Previous tag: $PREV_TAG"
```

### Step 2: Gather raw material

```bash
# Commits since previous tag
git log --no-merges --pretty=format:'- %s (%h)' "$PREV_TAG"..HEAD

# Merged PRs since previous tag. Scope by the tag's commit date so this doesn't
# pull in PRs merged before PREV_TAG (which would skew the notes and version bump).
SINCE=$(git log -1 --format=%cI "$PREV_TAG")
gh pr list --state merged --base main --limit 50 \
  --search "merged:>=$SINCE" \
  --json number,title,mergedAt,labels,author \
  --jq '.[] | "#\(.number) \(.title) — @\(.author.login)"'
```

Read PR bodies for any with non-trivial user-facing impact:

```bash
gh pr view <number> --json title,body,labels
```

### Step 3: Classify

Bucket each change:

| Category | Includes |
|----------|----------|
| Features | New commands, new screens, new keybindings, new config |
| Fixes | Bug fixes the user would notice |
| Improvements | Performance, UX polish, better error messages |
| Breaking | Renamed flags, removed config keys, changed defaults |
| Internal (omit) | Refactors, test-only changes, doc-only changes, dep bumps |

When in doubt, ask: "would a user notice if I left this out of the notes?" If no, omit it.

### Step 4: Draft the notes

Use this format:

```markdown
## vX.Y.Z — YYYY-MM-DD

### Features
- Short, user-facing description. (#123)

### Fixes
- Short description of the bug and what now works. (#124)

### Improvements
- What got better and where the user will feel it. (#125)

### Breaking changes
- What changed, what to do about it. (#126)
```

Each bullet should be one line, present-tense, user-facing. Link the PR number in parentheses.

### Step 5: Decide the version bump

Suggest a version following semver:

- **Breaking changes** present → major bump
- **Features** only → minor bump
- **Fixes / Improvements** only → patch bump

State the suggestion and the reasoning, but let the user confirm.

### Step 6: Present

Output the draft to the conversation. Do not create files or run `gh release create` unless the user asks.

If the user approves and wants to publish:

```bash
gh release create vX.Y.Z --title "vX.Y.Z" --notes-file <(cat <<'EOF'
... notes ...
EOF
)
```

## Tips

- Squash-merged PRs usually have one clean commit; rebase-merged PRs may need PR-level grouping.
- If a feature spans multiple PRs, write one bullet and link the most descriptive PR.
- `gh release view <prev-tag>` shows how prior notes were structured — match the existing style.
- Skip the "Co-Authored-By" / "Generated with Claude Code" trailers when summarizing commits.
