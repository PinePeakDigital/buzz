#!/bin/bash

# Setup Git hooks for the buzz project
set -e

echo "🔧 Setting up Git hooks..."

# Ensure hooks directory exists
mkdir -p .git/hooks

# Copy commit message hook
cp scripts/commit-msg .git/hooks/commit-msg
chmod +x .git/hooks/commit-msg

echo "✅ Git hooks installed successfully!"
echo "📋 Conventional commit format will now be enforced."
echo ""
echo "💡 Valid commit formats:"
echo "   feat: add new feature"
echo "   fix: resolve bug in authentication"
echo "   docs: update README"
echo "   refactor: simplify user service"
