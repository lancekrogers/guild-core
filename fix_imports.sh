#!/bin/bash

# Find all Go files
find . -name "*.go" -type f | while read -r file; do
  # Replace "github.com/blockhead-consulting/Guild" with "github.com/blockhead-consulting/guild"
  sed -i '' 's|github.com/blockhead-consulting/Guild|github.com/blockhead-consulting/guild|g' "$file"
done

echo "Import paths have been fixed!"