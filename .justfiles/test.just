# Test recipes
BUILDTOOL := "go run ./internal/buildutil"

# Show available test commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Run all tests (clean + build + unit + integration + happy) with visual dashboard
[no-cd]
all:
    @{{BUILDTOOL}} all

# Run unit tests with visual dashboard
[no-cd]
unit:
    @{{BUILDTOOL}} test

# Run unit tests with verbose output
[no-cd]
unit-verbose:
    @{{BUILDTOOL}} -v test

# Run integration tests with visual dashboard
[no-cd]
integration:
    @{{BUILDTOOL}} integration

# Run integration tests with verbose output
[no-cd]
integration-verbose:
    @{{BUILDTOOL}} -v integration

# Run E2E tests with visual dashboard
[no-cd]
e2e:
    @{{BUILDTOOL}} e2e

# Run E2E tests with verbose output
[no-cd]
e2e-verbose:
    @{{BUILDTOOL}} -v e2e

# Run happy path validation tests with visual dashboard
[no-cd]
happy:
    @{{BUILDTOOL}} happy

# Run happy path validation tests with verbose output
[no-cd]
happy-verbose:
    @{{BUILDTOOL}} -v happy

# Validate demo scripts
[no-cd]
validate-demo:
    @{{BUILDTOOL}} validate-demo

# Run tests with coverage (using go test directly)
[no-cd]
coverage:
    go test -cover ./...

# Generate HTML coverage report
[no-cd]
coverage-html:
    go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
