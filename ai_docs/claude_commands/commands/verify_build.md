## Verify Build

@context

This command verifies that the Guild project builds and tests correctly. It ensures that all requirements are met before committing code.

### Build Verification

Run these commands to verify the build:

```bash
# Verify code builds without errors
go build ./...

# Check for compiler warnings
go build -gcflags="-e" ./...

# Run tests with race detection
go test -race ./...

# Check test coverage
go test -cover ./...

# Verify all dependencies are properly documented in go.mod
go mod tidy
go mod verify
```

### Code Quality Checks

Run these quality checks to ensure code standards:

```bash
# Run gofmt to check formatting
gofmt -d -s .

# Run golint
golint ./...

# Run go vet
go vet ./...

# Check for common mistakes
staticcheck ./...
```

### Module Path Verification

Ensure all imports use the correct module path:

```bash
# Check for any remaining references to old module path
grep -r "github.com/lancekrogers/guild" --include="*.go" .

# Verify imports use lowercase path (following Go conventions)
grep -r "github.com/Blockhead-Consulting/Guild" --include="*.go" .
grep -r "github.com/Blockhead-Consulting/guild" --include="*.go" .

# Correct path should be:
grep -r "github.com/blockhead-consulting/guild" --include="*.go" .
```

### Directory Structure Verification

Verify the project follows the expected structure:

```bash
# Check that required directories exist
[ -d cmd ] && echo "cmd directory exists" || echo "cmd directory missing"
[ -d pkg ] && echo "pkg directory exists" || echo "pkg directory missing"
[ -d internal ] && echo "internal directory exists" || echo "internal directory missing"

# Verify objective system structure
[ -d internal/prompts/objective/markdown ] && echo "prompt directory structure correct" || echo "prompt directory structure incorrect"
[ -d pkg/generator/objective ] && echo "generator directory structure correct" || echo "generator directory structure incorrect"
[ -d pkg/ui/objective ] && echo "UI directory structure correct" || echo "UI directory structure incorrect"
```

### Clean Build

For a completely clean build:

```bash
# Remove build artifacts
go clean -cache -testcache

# Rebuild from scratch
go build ./...
```

When implementing new code, always run these verification steps to ensure the project maintains high quality and correct structure.
