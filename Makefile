# ⚔️ Guild Framework Build System ⚔️
# Clean dispatcher Makefile - all visuals handled by buildutil

BUILDTOOL := go run ./internal/buildutil
.DEFAULT_GOAL := help
.PHONY: build test integration clean all quick ci-build ci-test ci-integration ci-clean help

# Primary targets (with visual output)
build:
	@$(BUILDTOOL) build

test:
	@$(BUILDTOOL) test

integration:
	@$(BUILDTOOL) integration

clean:
	@$(BUILDTOOL) clean

all:
	@$(BUILDTOOL) all

# Quick build (no visuals, just compile)
quick:
	@go build -o bin/guild ./cmd/guild

# CI variants (no colors)
ci-build:
	@$(BUILDTOOL) --no-color build

ci-test:
	@$(BUILDTOOL) --no-color test

ci-integration:
	@$(BUILDTOOL) --no-color integration

ci-clean:
	@$(BUILDTOOL) --no-color clean

# Help target
help:
	@echo "Guild Framework Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build        Build Guild with progress display"
	@echo "  test         Run unit tests with visual feedback"
	@echo "  integration  Run integration tests"
	@echo "  clean        Remove all build artifacts"
	@echo "  all          Clean, build, test, and integration"
	@echo "  quick        Fast build without visuals"
	@echo "  ci-*         CI variants (no colors)"
	@echo ""
	@echo "Examples:"
	@echo "  make build      # Build with progress bars"
	@echo "  make test       # Run tests with feedback"
	@echo "  make ci-build   # Build for CI (plain text)"