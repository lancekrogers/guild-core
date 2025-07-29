# ⚔️ Guild Framework Build System ⚔️
# Clean dispatcher Makefile - all visuals handled by buildutil

BUILDTOOL := go run ./internal/buildutil
.DEFAULT_GOAL := help
.PHONY: build test test-verbose test-pkg integration integration-verbose integration-debug e2e e2e-verbose validate-demo clean all all-verbose quick ci-build ci-test ci-integration ci-e2e ci-clean install uninstall help install-completion install-bash-completion install-zsh-completion install-fish-completion benchmark benchmark-suggestions benchmark-ui benchmark-ui-thresholds test-ui-integration test-ui-complete happy happy-verbose ci-happy

# Primary targets (with visual output)
# DEVELOPER TARGET: Full build with go vet validation and visual feedback
# Use this when developing to ensure code quality
build:
	@$(BUILDTOOL) build

test:
	@$(BUILDTOOL) test

integration:
	@$(BUILDTOOL) integration

e2e:
	@$(BUILDTOOL) e2e

happy:
	@$(BUILDTOOL) happy

# Verbose variants for debugging test failures
# Usage: make test-verbose, make integration-verbose, etc.
# For all verbose targets, append -verbose to the target name
test-verbose:
	@$(BUILDTOOL) -v test

integration-verbose:
	@$(BUILDTOOL) -v integration

e2e-verbose:
	@$(BUILDTOOL) -v e2e

happy-verbose:
	@$(BUILDTOOL) -v happy

# Debug specific integration test suites with full output
# Usage: make integration-debug SUITE=user-journey
integration-debug:
	@if [ -z "$(SUITE)" ]; then \
		echo "Usage: make integration-debug SUITE=<suite-name>"; \
		echo "Available suites:"; \
		find integration/happy-path -type d -name "*" -depth 1 | sed 's|integration/happy-path/||' | sort; \
	else \
		echo "🔍 Running integration tests for suite: $(SUITE) with full output"; \
		go test -v -timeout 60s ./integration/happy-path/$(SUITE)/... 2>&1 | tee test-$(SUITE).log; \
		echo ""; \
		echo "📄 Output saved to test-$(SUITE).log"; \
	fi

# TUI tests with proper terminal cleanup
test-teatest:
	@echo "🧪 Running TUI tests with terminal protection..."
	@trap 'printf "\033c" && echo "✅ Terminal restored"' EXIT && \
		go test -v -timeout 30s ./internal/chat/commands/... ./internal/chat/... ./internal/ui/init/... -run "Tea|TUI" 2>&1 | \
		grep -E "(PASS|FAIL|ok|^---)" || true
	@printf "\033c"
	@echo "✅ TUI tests completed - terminal restored"

# Fix terminal after test corruption
fix-terminal:
	@if [ -f scripts/fix-terminal.sh ]; then \
		bash scripts/fix-terminal.sh; \
	else \
		printf "\033c\033[?1049l\033[?25h\033[0m"; \
		stty sane 2>/dev/null || true; \
		reset; \
		echo "✅ Terminal restored"; \
	fi

# Show project dashboard
dashboard:
	@$(BUILDTOOL) dashboard

validate-demo:
	@$(BUILDTOOL) validate-demo

# Performance benchmarks
benchmark:
	@echo "🚀 Running comprehensive performance benchmarks..."
	@go run benchmarks/run_benchmarks.go

benchmark-suggestions:
	@echo "🚀 Running suggestion system benchmarks..."
	@go test -bench=BenchmarkSuggestion -benchmem -benchtime=10s ./benchmarks

# UI Performance benchmarks
benchmark-ui:
	@echo "🎨 Running UI performance benchmarks..."
	@echo "📊 Theme Management Performance:"
	@go test -bench=BenchmarkThemeManager -benchmem ./internal/ui/theme 2>/dev/null | grep -E "(Benchmark|B/op|allocs/op)"
	@echo ""
	@echo "🖼️  Component Rendering Performance:"
	@go test -bench=BenchmarkComponentLibrary -benchmem ./internal/ui/components 2>/dev/null | grep -E "(Benchmark|B/op|allocs/op)"
	@echo ""
	@echo "⚡ Memory Usage Analysis:"
	@go test -bench=BenchmarkMemoryUsage -benchmem ./internal/ui/... 2>/dev/null | grep -E "(Benchmark|B/op|allocs/op)"

# UI Performance threshold validation
benchmark-ui-thresholds:
	@echo "🎯 Validating UI performance thresholds..."
	@echo "Theme System Thresholds:"
	@go test -v -run="TestPerformanceThresholds" ./internal/ui/theme 2>/dev/null | grep -E "(RUN|completed in|PASS)"
	@echo ""
	@echo "Component Rendering Thresholds:"
	@go test -v -run="TestComponentPerformanceThresholds" ./internal/ui/components 2>/dev/null | grep -E "(RUN|completed in|PASS)"
	@echo ""
	@echo "Shortcut System Thresholds:"
	@go test -v -run="TestShortcutPerformanceThresholds" ./internal/ui/shortcuts 2>/dev/null | grep -E "(RUN|completed in|PASS)"
	@echo ""
	@echo "✅ All performance targets validated!"

# UI Integration tests
test-ui-integration:
	@echo "🔗 Running UI integration tests..."
	@go test -v -timeout 60s ./integration/ui/... -run "TestUI.*Integration"
	@echo "✅ UI integration tests completed!"

# Complete UI test suite
test-ui-complete:
	@echo "🎨 Running complete UI test suite..."
	@echo "📋 Unit Tests:"
	@go test -v ./internal/ui/... | grep -E "(RUN|PASS|FAIL)"
	@echo ""
	@echo "🔗 Integration Tests:"
	@$(MAKE) test-ui-integration
	@echo ""
	@echo "⚡ Performance Tests:"
	@$(MAKE) benchmark-ui-thresholds
	@echo ""
	@echo "✅ Complete UI test suite finished!"


test-cleanup-verify:
	@echo "🔍 Verifying terminal cleanup helpers..."
	@go test -v ./internal/testing/... -run TestTerminalCleanup
	@echo "✅ Cleanup verification complete"

clean:
	@$(BUILDTOOL) clean

all:
	@$(BUILDTOOL) all

all-verbose:
	@$(BUILDTOOL) -v all

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

ci-e2e:
	@$(BUILDTOOL) --no-color e2e

ci-happy:
	@$(BUILDTOOL) --no-color happy

ci-clean:
	@$(BUILDTOOL) --no-color clean

# Install/uninstall targets
# USER TARGET: Fast installation without development checks
# This is the primary installation method for end users who want to get productive quickly
install:
	@$(BUILDTOOL) build-only
	@$(BUILDTOOL) install
	@$(MAKE) install-completion

# DEVELOPER TARGET: Remove Guild from system
uninstall:
	@$(BUILDTOOL) uninstall

# Shell completion installation
install-completion:
	@echo "Installing shell completion..."
	@if [ ! -f ./bin/guild ]; then \
		echo "Error: guild binary not found. Run 'make install' first."; \
		exit 1; \
	fi
	@# Detect shell by checking SHELL environment variable
	@if echo "$$SHELL" | grep -q bash; then \
		$(MAKE) install-bash-completion; \
	elif echo "$$SHELL" | grep -q zsh; then \
		$(MAKE) install-zsh-completion; \
	elif echo "$$SHELL" | grep -q fish; then \
		$(MAKE) install-fish-completion; \
	else \
		echo "Could not detect shell type from $$SHELL. Please run one of:"; \
		echo "  make install-bash-completion"; \
		echo "  make install-zsh-completion"; \
		echo "  make install-fish-completion"; \
	fi

install-bash-completion:
	@echo "Installing bash completion..."
	@if [ ! -f ./bin/guild ]; then \
		echo "Error: guild binary not found. Run 'make install' first."; \
		exit 1; \
	fi
	@if [ -d /etc/bash_completion.d ]; then \
		./bin/guild completion bash | sudo tee /etc/bash_completion.d/guild > /dev/null; \
		echo "Bash completion installed to /etc/bash_completion.d/guild"; \
	elif [ -d /usr/local/etc/bash_completion.d ]; then \
		./bin/guild completion bash | sudo tee /usr/local/etc/bash_completion.d/guild > /dev/null; \
		echo "Bash completion installed to /usr/local/etc/bash_completion.d/guild"; \
	elif command -v brew >/dev/null 2>&1 && [ -d "$$(brew --prefix)/etc/bash_completion.d" ]; then \
		./bin/guild completion bash > "$$(brew --prefix)/etc/bash_completion.d/guild"; \
		echo "Bash completion installed to $$(brew --prefix)/etc/bash_completion.d/guild"; \
	else \
		echo "Error: bash completion directory not found"; \
		echo "You can manually install by running:"; \
		echo "  guild completion bash > /path/to/completion/dir/guild"; \
		exit 1; \
	fi

install-zsh-completion:
	@echo "Installing zsh completion..."
	@if [ ! -f ./bin/guild ]; then \
		echo "Error: guild binary not found. Run 'make install' first."; \
		exit 1; \
	fi
	@# Try standard zsh completion locations in order
	@if [ -d /opt/homebrew/share/zsh/site-functions ] && [ -w /opt/homebrew/share/zsh/site-functions ]; then \
		./bin/guild completion zsh > /opt/homebrew/share/zsh/site-functions/_guild; \
		echo "Zsh completion installed to /opt/homebrew/share/zsh/site-functions/_guild"; \
	elif [ -d /usr/local/share/zsh/site-functions ] && [ -w /usr/local/share/zsh/site-functions ]; then \
		./bin/guild completion zsh > /usr/local/share/zsh/site-functions/_guild; \
		echo "Zsh completion installed to /usr/local/share/zsh/site-functions/_guild"; \
	else \
		mkdir -p ~/.zsh/completions; \
		./bin/guild completion zsh > ~/.zsh/completions/_guild; \
		echo "Zsh completion installed to ~/.zsh/completions/_guild"; \
		echo "Add this to your ~/.zshrc if not already present:"; \
		echo "  fpath=(~/.zsh/completions \$$fpath)"; \
		echo "  autoload -U compinit && compinit"; \
	fi
	@echo "Start a new shell or run 'source ~/.zshrc' to use completion."

install-fish-completion:
	@echo "Installing fish completion..."
	@if [ ! -f ./bin/guild ]; then \
		echo "Error: guild binary not found. Run 'make install' first."; \
		exit 1; \
	fi
	@mkdir -p ~/.config/fish/completions
	@./bin/guild completion fish > ~/.config/fish/completions/guild.fish
	@echo "Fish completion installed to ~/.config/fish/completions/guild.fish"

# Test specific packages helper
test-pkg:
	@echo "To test specific packages, use go test directly:"
	@echo ""
	@echo "Examples:"
	@echo "  go test ./pkg/agent/...              # Test all agent packages"
	@echo "  go test ./tools/jump/...             # Test jump tool"
	@echo "  go test -v ./pkg/memory/...          # Verbose output"
	@echo "  go test -race ./pkg/providers/...    # With race detection"
	@echo "  go test -short ./pkg/...             # Skip long tests"
	@echo "  go test -run TestName ./pkg/...      # Run specific test by name"
	@echo ""
	@echo "Multiple packages:"
	@echo "  go test ./pkg/agent/... ./pkg/memory/..."
	@echo ""
	@echo "With timeout:"
	@echo "  go test -timeout 30s ./tools/..."
	@echo ""
	@echo "⚠️  NEVER use 'go test -c' as it creates .test binaries in the root!"

# Help target
help:
	@echo "Guild Framework Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "=== VERBOSE MODE ==="
	@echo "  For detailed output, append -verbose to most targets:"
	@echo "  Examples: make test-verbose, make all-verbose, make integration-verbose"
	@echo ""
	@echo "=== USER TARGETS ==="
	@echo "  install                  Fast install Guild (no go vet, for users)"
	@echo "  uninstall                Remove Guild from Go bin directory"
	@echo ""
	@echo "=== DEVELOPER TARGETS ==="
	@echo "  build                    Full build with go vet validation"
	@echo "  test                     Run unit tests with visual feedback"
	@echo "  test-verbose             Run unit tests with verbose output (shows all test logs)"
	@echo "  test-pkg                 Show examples of testing specific packages"
	@echo "  test-teatest             Run TUI tests with proper terminal cleanup"
	@echo "  integration              Run integration tests"
	@echo "  integration-verbose      Run integration tests with verbose output"
	@echo "  integration-debug        Debug specific test suite (e.g., make integration-debug SUITE=user-journey)"
	@echo "  e2e                      Run end-to-end tests"
	@echo "  e2e-verbose              Run end-to-end tests with verbose output"
	@echo "  happy                    Run comprehensive performance and SLA validation tests"
	@echo "  happy-verbose            Run happy path tests with verbose output"
	@echo "  validate-demo            Validate demo scripts and functionality"
	@echo "  benchmark                Run comprehensive performance benchmarks"
	@echo "  benchmark-suggestions    Run suggestion system benchmarks only"
	@echo "  benchmark-ui             Run UI performance benchmarks with memory profiling"
	@echo "  benchmark-ui-thresholds  Validate UI performance meets hard thresholds"
	@echo "  test-ui-integration      Run UI system integration tests"
	@echo "  test-ui-complete         Run complete UI test suite (unit + integration + performance)"
	@echo "  clean                    Remove all build artifacts"
	@echo "  all                      Clean, build, test, and integration"
	@echo "  all-verbose              Run all tasks with verbose output"
	@echo "  quick                    Fast build without visuals"
	@echo "  ci-*                     CI variants (no colors)"
	@echo "  ci-happy                 Run happy path tests for CI (no colors)"
	@echo "  fix-terminal             Fix terminal after test corruption"
	@echo "  dashboard                Show project status dashboard"
	@echo ""
	@echo "=== COMPLETION TARGETS ==="
	@echo "  install-completion       Auto-detect shell and install completion"
	@echo "  install-bash-completion  Install bash completion"
	@echo "  install-zsh-completion   Install zsh completion"
	@echo "  install-fish-completion  Install fish completion"
	@echo ""
	@echo "=== QUICK START FOR USERS ==="
	@echo "  make install      # Fast install (30 seconds)"
	@echo "  guild init        # Create workspace with Elena agent"
	@echo "  guild serve       # Start daemon (separate terminal)"
	@echo "  guild chat        # Start chatting immediately"
	@echo ""
	@echo "=== DEVELOPMENT WORKFLOW ==="
	@echo "  make build        # Full validation build"
	@echo "  make test         # Run all tests properly"
	@echo "  make ci-build     # Build for CI (plain text)"