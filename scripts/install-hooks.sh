#!/bin/sh
# Install git hooks for potaco
# Usage: sh scripts/install-hooks.sh
set -e

HOOKS_DIR=".git/hooks"

echo "Installing pre-commit hook..."
cp scripts/pre-commit.sh "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-commit"

echo "Installing pre-push hook..."
cp scripts/pre-push.sh "$HOOKS_DIR/pre-push"
chmod +x "$HOOKS_DIR/pre-push"

echo "Git hooks installed successfully."
echo "  pre-commit: gofmt, go vet, go mod tidy check"
echo "  pre-push: go test ./..."
