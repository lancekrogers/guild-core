# Preventing Test Binaries in Root Directory

## Problem

When running `go test` with certain flags, it generates `.test` binary files in the current directory:
- `agent_test.test`
- `chat_test.test`
- `memory_test.test`

These files clutter the repository root and should not be there.

## Root Cause

Test binaries are created when:
1. Running `go test -c` (compile test binary)
2. Running `go test -run=XXX` with certain flags
3. Running tests with `-cpuprofile`, `-memprofile`, or `-trace` flags
4. Using certain test caching behaviors

## Solutions

### 1. Use the Makefile (Recommended)

Always use the Makefile targets which handle test execution properly:

```bash
make test          # Run all tests
make unit-test     # Run unit tests with dashboard
make integration   # Run integration tests
make coverage      # Generate coverage report
```

### 2. Configure Go Test Cache

Set the test cache directory to avoid local binary generation:

```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export GOCACHE=$HOME/.cache/go-build
export GOTESTCACHE=$HOME/.cache/go-test
```

### 3. Use Test Scripts

Create a test wrapper script that ensures clean execution:

```bash
#!/bin/bash
# scripts/test.sh

# Create temporary directory for test artifacts
TESTDIR=$(mktemp -d)
trap "rm -rf $TESTDIR" EXIT

# Run tests with explicit cache directory
GOCACHE=$TESTDIR go test "$@"
```

### 4. Git Hooks

Add a pre-commit hook to prevent committing test binaries:

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Check for test binaries
if find . -name "*.test" -not -path "./.git/*" | grep -q .; then
    echo "Error: Test binaries found. Please remove them before committing."
    find . -name "*.test" -not -path "./.git/*"
    exit 1
fi
```

### 5. VS Code Configuration

If using VS Code, configure it to use proper test execution:

```json
// .vscode/settings.json
{
    "go.testFlags": [
        "-v"
    ],
    "go.testEnvVars": {
        "GOCACHE": "${workspaceFolder}/.cache/go-build"
    },
    "files.exclude": {
        "**/*.test": true,
        ".cache": true
    }
}
```

### 6. Makefile Enhancement

The Makefile already handles this correctly by:
- Building to `/tmp/guild-build-test-$$$$` for compilation checks
- Not using `-c` flag for test execution
- Cleaning up temporary files after tests

### 7. Developer Guidelines

1. **Always use `make test`** instead of `go test` directly
2. If you must use `go test`, avoid these flags:
   - `-c` (compile only)
   - `-o` (output binary)
   - Profile flags without proper output paths

3. If test binaries are created accidentally:
   ```bash
   # Clean them up immediately
   find . -name "*.test" -delete
   ```

## Automated Cleanup

Add this to your Makefile clean target:

```makefile
clean:
    @rm -f bin/*
    @rm -f *.test
    @rm -rf .test-*
    @find . -name "*.test" -not -path "./.git/*" -delete
    @echo "Cleaned build artifacts and test binaries"
```

## Summary

To prevent test binaries in the root directory:
1. Use `make test` commands exclusively
2. Configure your IDE to use the Makefile
3. Set up git hooks to catch accidents
4. Clean up immediately if binaries are created

This maintains a clean repository structure and follows enterprise development standards.