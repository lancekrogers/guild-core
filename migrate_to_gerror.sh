#!/bin/bash

# Script to help migrate fmt.Errorf to gerror
# Usage: ./migrate_to_gerror.sh <package_path>

PACKAGE_PATH="${1:-pkg}"

echo "=== Guild Framework Error Migration Tool ==="
echo "Analyzing $PACKAGE_PATH for fmt.Errorf usage..."
echo

# Count total occurrences
TOTAL=$(grep -r "fmt.Errorf" "$PACKAGE_PATH" --include="*.go" | grep -v "_test.go" | wc -l)
echo "Total fmt.Errorf occurrences (non-test): $TOTAL"
echo

# Show packages with most occurrences
echo "Top 10 packages with fmt.Errorf:"
grep -r "fmt.Errorf" "$PACKAGE_PATH" --include="*.go" | grep -v "_test.go" | cut -d: -f1 | sort | uniq -c | sort -nr | head -10
echo

# Show common patterns
echo "Common error patterns:"
grep -r "fmt.Errorf" "$PACKAGE_PATH" --include="*.go" | grep -v "_test.go" | sed 's/.*fmt.Errorf/fmt.Errorf/' | sort | uniq -c | sort -nr | head -10
echo

# Check which files already import gerror
echo "Files already using gerror:"
grep -r "guild-ventures/guild-core/pkg/gerror" "$PACKAGE_PATH" --include="*.go" | grep -v "_test.go" | wc -l

# Suggested migration patterns
echo
echo "=== Migration Guide ==="
echo "1. Add import: \"github.com/guild-ventures/guild-core/pkg/gerror\""
echo
echo "2. Simple errors (no wrapping):"
echo "   OLD: fmt.Errorf(\"message\")"
echo "   NEW: gerror.New(gerror.ErrCodeXXX, \"message\", nil)."
echo "        WithComponent(\"ComponentName\")."
echo "        WithOperation(\"OperationName\")"
echo
echo "3. Wrapped errors:"
echo "   OLD: fmt.Errorf(\"message: %w\", err)"
echo "   NEW: gerror.Wrap(err, gerror.ErrCodeXXX, \"message\")."
echo "        WithComponent(\"ComponentName\")."
echo "        WithOperation(\"OperationName\")"
echo
echo "4. Errors with values:"
echo "   OLD: fmt.Errorf(\"message %s: %w\", value, err)"
echo "   NEW: gerror.Wrap(err, gerror.ErrCodeXXX, \"message\")."
echo "        WithComponent(\"ComponentName\")."
echo "        WithOperation(\"OperationName\")."
echo "        WithDetails(\"key\", value)"
echo
echo "Common error codes:"
echo "  - ErrCodeStorage: Database/file operations"
echo "  - ErrCodeValidation: Input validation"
echo "  - ErrCodeNotFound: Resource not found"
echo "  - ErrCodeInternal: Internal errors"
echo "  - ErrCodeTransaction: Transaction failures"
echo "  - ErrCodeAgent: Agent-related errors"
echo "  - ErrCodeOrchestration: Orchestration errors"