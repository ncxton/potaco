#!/bin/sh
# Pre-commit hook for potaco: runs gofmt, go vet, and go mod tidy check
set -e

echo "Running gofmt check..."
if [ -n "$(gofmt -l .)" ]; then
    echo "ERROR: The following files need gofmt:"
    gofmt -l .
    echo "Run 'gofmt -w .' to fix."
    exit 1
fi

echo "Running go vet..."
go vet ./...

echo "Checking go mod tidy..."
go mod tidy
if ! git diff --exit-code go.mod go.sum; then
    echo "ERROR: go.mod or go.sum is not tidy. Run 'go mod tidy' and commit."
    exit 1
fi

echo "All pre-commit checks passed."
