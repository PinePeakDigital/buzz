#!/bin/bash

# Script to fetch all CodeRabbit PR feedback using GitHub CLI
# This includes general comments, inline review comments, and review summaries
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Fetching CodeRabbit PR feedback...${NC}"
echo ""

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    echo -e "${RED}❌ Error: GitHub CLI (gh) is not installed${NC}"
    echo "Please install it from: https://cli.github.com/"
    exit 1
fi

# Get current PR number
echo -e "${YELLOW}📍 Detecting current PR...${NC}"
PR_NUMBER=$(gh pr view --json number --jq '.number' 2>/dev/null)

if [ -z "$PR_NUMBER" ]; then
    echo -e "${RED}❌ Error: Not in a PR context${NC}"
    echo "This script must be run from a branch with an associated pull request."
    echo "Please check out a PR branch first."
    exit 1
fi

echo -e "${GREEN}✓ Found PR #${PR_NUMBER}${NC}"
echo ""

# Fetch PR details for context
PR_TITLE=$(gh pr view "$PR_NUMBER" --json title --jq '.title')
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PR #${PR_NUMBER}: ${PR_TITLE}${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Section 1: General PR Timeline Comments
echo -e "${YELLOW}📝 GENERAL PR COMMENTS${NC}"
echo -e "${YELLOW}─────────────────────────────────────────────────────────────${NC}"

COMMENTS=$(gh pr view "$PR_NUMBER" --json comments --jq '.comments[] | select(.author.login == "coderabbitai") | {body: .body, createdAt: .createdAt}')

if [ -z "$COMMENTS" ]; then
    echo "No general comments from CodeRabbit found."
else
    echo "$COMMENTS" | jq -r '
        "Created: " + .createdAt + "\n" +
        "─────────────────────────────────────────────────────────────\n" +
        .body + "\n" +
        "═════════════════════════════════════════════════════════════\n"
    '
fi

echo ""

# Section 2: Inline Review Comments (diff-specific comments)
echo -e "${YELLOW}💬 INLINE REVIEW COMMENTS${NC}"
echo -e "${YELLOW}─────────────────────────────────────────────────────────────${NC}"

# Get repository owner and name
REPO=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')

# Fetch review comments using GitHub API
REVIEW_COMMENTS=$(gh api "/repos/$REPO/pulls/$PR_NUMBER/comments" --jq '.[] | select(.user.login == "coderabbitai") | {path: .path, line: .line, body: .body, createdAt: .created_at, position: .position}')

if [ -z "$REVIEW_COMMENTS" ]; then
    echo "No inline review comments from CodeRabbit found."
else
    echo "$REVIEW_COMMENTS" | jq -r '
        "File: " + .path + " (line " + (.line // .position | tostring) + ")" + "\n" +
        "Created: " + .createdAt + "\n" +
        "─────────────────────────────────────────────────────────────\n" +
        .body + "\n" +
        "═════════════════════════════════════════════════════════════\n"
    '
fi

echo ""

# Section 3: Review Summaries
echo -e "${YELLOW}📋 REVIEW SUMMARIES${NC}"
echo -e "${YELLOW}─────────────────────────────────────────────────────────────${NC}"

REVIEWS=$(gh api "/repos/$REPO/pulls/$PR_NUMBER/reviews" --jq '.[] | select(.user.login == "coderabbitai") | {state: .state, body: .body, submittedAt: .submitted_at}')

if [ -z "$REVIEWS" ]; then
    echo "No review summaries from CodeRabbit found."
else
    echo "$REVIEWS" | jq -r '
        "State: " + .state + "\n" +
        "Submitted: " + .submittedAt + "\n" +
        "─────────────────────────────────────────────────────────────\n" +
        (.body // "No summary provided") + "\n" +
        "═════════════════════════════════════════════════════════════\n"
    '
fi

echo ""
echo -e "${GREEN}✅ CodeRabbit feedback retrieval complete${NC}"
