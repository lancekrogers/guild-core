# Guild Framework Build System
# https://just.systems/man/

# Import modules
mod docker 'just/docker.just'

# Default recipe - show available commands
default:
    @just --list

# ============================================================================
# Main Build Commands
# ============================================================================

# Build Guild binary
build:
    @echo "🏰 Building Guild..."
    @go run ./internal/buildutil build

# Run unit tests
test:
    @echo "🧪 Running unit tests..."
    @go run ./internal/buildutil test

# Run integration tests
integration:
    @echo "🔧 Running integration tests..."
    @go run ./internal/buildutil integration

# Run all tests
all:
    @echo "🚀 Running all tests..."
    @go run ./internal/buildutil all

# Quick build (no visuals, just compile)
quick:
    @go build -o bin/guild ./cmd/guild

# Clean build artifacts
clean:
    @echo "🧹 Cleaning build artifacts..."
    @go run ./internal/buildutil clean
    @rm -rf bin/ test-output/ *.log

# ============================================================================
# Installation
# ============================================================================

# Install Guild locally
install: build
    @echo "📦 Installing Guild..."
    @go run ./internal/buildutil install

# Uninstall Guild
uninstall:
    @echo "🗑️ Uninstalling Guild..."
    @go run ./internal/buildutil uninstall

# ============================================================================
# Development Tools
# ============================================================================

# Run linter
lint:
    @echo "🔍 Running linter..."
    @golangci-lint run

# Format code
fmt:
    @echo "✨ Formatting code..."
    @go fmt ./...
    @gofumpt -w .

# Run go mod tidy
tidy:
    @echo "📦 Tidying dependencies..."
    @go mod tidy

# Generate code (proto, mocks, etc.)
generate:
    @echo "⚙️ Generating code..."
    @./scripts/generate-proto.sh
    @go generate ./...

# ============================================================================
# Performance
# ============================================================================

# Run benchmarks
bench:
    @echo "⚡ Running benchmarks..."
    @go test -bench=. -benchmem ./pkg/...

# Run specific benchmark
bench-pkg pkg:
    @echo "⚡ Running benchmarks for {{pkg}}..."
    @go test -bench=. -benchmem ./{{pkg}}/...

# ============================================================================
# Documentation
# ============================================================================

# Serve documentation locally
docs:
    @echo "📚 Serving documentation..."
    @echo "Visit http://localhost:6060/pkg/github.com/lancekrogers/guild/"
    @godoc -http=:6060

# Generate API documentation
api-docs:
    @echo "📝 Generating API documentation..."
    @swag init -g cmd/guild/main.go -o docs/api

# ============================================================================
# Utilities
# ============================================================================

# Show project status
status:
    @echo "📊 Project Status"
    @echo "=================="
    @echo "Branch: $(git branch --show-current)"
    @echo "Commit: $(git rev-parse --short HEAD)"
    @echo "Modified files: $(git status --porcelain | wc -l)"
    @echo ""
    @echo "Go version: $(go version)"
    @echo "Module: $(go list -m)"

# Run pre-commit hooks
pre-commit:
    @echo "🔒 Running pre-commit hooks..."
    @pre-commit run --all-files

# Fix terminal after corruption
fix-terminal:
    @printf "\033c\033[?1049l\033[?25h\033[0m"
    @stty sane 2>/dev/null || true
    @reset
    @echo "✅ Terminal restored"

# ============================================================================
# CI/CD
# ============================================================================

# Run CI pipeline locally
ci:
    @echo "🔄 Running CI pipeline..."
    @just clean
    @just build
    @just test
    @just integration
    @echo "✅ CI pipeline passed"

# Run CI tests (no color output)
ci-test:
    @go run ./internal/buildutil --no-color test

# Run CI integration tests (no color output)
ci-integration:
    @go run ./internal/buildutil --no-color integration

# ============================================================================
# Shortcuts
# ============================================================================

# Common shortcuts
alias t := test
alias i := integration
alias b := build
alias c := clean

# Docker shortcuts (using full module path)
d:
    @just docker shell
    
dt:
    @just docker test
    
di:
    @just docker integration

# ============================================================================
# Help
# ============================================================================

# Show detailed help
help:
    @echo "Guild Framework Build System"
    @echo "============================"
    @echo ""
    @echo "Main Commands:"
    @echo "  just build         - Build Guild binary"
    @echo "  just test          - Run unit tests"
    @echo "  just integration   - Run integration tests"
    @echo "  just all           - Run all tests"
    @echo "  just install       - Install Guild locally"
    @echo ""
    @echo "Docker Commands:"
    @echo "  just docker shell  - Interactive Docker shell"
    @echo "  just docker test   - Run tests in Docker"
    @echo "  just docker init   - Test guild init workflow"
    @echo ""
    @echo "Shortcuts:"
    @echo "  just t            - Run tests"
    @echo "  just b            - Build"
    @echo "  just d            - Docker shell"
    @echo ""
    @echo "For full list: just --list"
    @echo "For Docker commands: just docker --list"