#!/bin/sh
# Pre-push hook for potaco: runs tests before pushing
set -e

echo "Running tests..."
go test ./... -v

echo "All tests passed. Pushing..."
