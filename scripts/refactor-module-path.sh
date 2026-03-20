#!/usr/bin/env bash
set -euo pipefail

OLD=github.com/guild-framework/guild-core
NEW=github.com/guild-framework/guild-core

echo "Refactoring import paths: $OLD -> $NEW"

git ls-files '*.go' | while read -r f; do
  if grep -q "$OLD/" "$f"; then
    sed -i '' -e "s|$OLD/|$NEW/|g" "$f"
  fi
done

git ls-files '*.md' '*.yml' '*.yaml' '*.proto' 2>/dev/null | while read -r f; do
  if grep -q "$OLD" "$f"; then
    sed -i '' -e "s|$OLD|$NEW|g" "$f"
  fi
done

echo "Running go mod tidy and formatting"
go mod tidy
gofmt -w .

echo "Done."

