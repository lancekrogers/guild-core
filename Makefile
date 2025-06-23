# ⚔️ Guild Framework Build System ⚔️
# Clean dispatcher Makefile - all visuals handled by buildutil

BUILDTOOL := go run ./internal/buildutil
.DEFAULT_GOAL := help
.PHONY: build test test-pkg integration e2e validate-demo clean all quick ci-build ci-test ci-integration ci-e2e ci-clean install uninstall help install-completion install-bash-completion install-zsh-completion install-fish-completion benchmark benchmark-suggestions

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


test-cleanup-verify:
	@echo "🔍 Verifying terminal cleanup helpers..."
	@go test -v ./internal/testing/... -run TestTerminalCleanup
	@echo "✅ Cleanup verification complete"

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

ci-e2e:
	@$(BUILDTOOL) --no-color e2e

ci-clean:
	@$(BUILDTOOL) --no-color clean

# Install/uninstall targets
# USER TARGET: Fast installation without development checks
# This is the primary installation method for end users who want to get productive quickly
install:
	@echo "🏗️  Building Guild binary for installation..."
	@go build -o bin/guild ./cmd/guild
	@$(BUILDTOOL) install
	@$(MAKE) install-completion

# DEVELOPER TARGET: Remove Guild from system
uninstall:
	@$(BUILDTOOL) uninstall

# Shell completion installation
install-completion: build
	@echo "Installing shell completion..."
	@if [ -n "$$BASH_VERSION" ]; then \
		$(MAKE) install-bash-completion; \
	elif [ -n "$$ZSH_VERSION" ]; then \
		$(MAKE) install-zsh-completion; \
	elif [ -n "$$FISH_VERSION" ]; then \
		$(MAKE) install-fish-completion; \
	else \
		echo "Could not detect shell type. Please run one of:"; \
		echo "  make install-bash-completion"; \
		echo "  make install-zsh-completion"; \
		echo "  make install-fish-completion"; \
	fi

install-bash-completion: build
	@echo "Installing bash completion..."
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

install-zsh-completion: build
	@echo "Installing zsh completion..."
	@./bin/guild completion zsh > "$${fpath[1]}/_guild" || { \
		echo "Error: Could not install to fpath."; \
		echo "You can manually install by running:"; \
		echo "  guild completion zsh > ~/.zsh/completions/_guild"; \
		echo "And ensure ~/.zsh/completions is in your fpath"; \
		exit 1; \
	}
	@echo "Zsh completion installed. Start a new shell to use it."

install-fish-completion: build
	@echo "Installing fish completion..."
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
	@echo "=== USER TARGETS ==="
	@echo "  install                  Fast install Guild (no go vet, for users)"
	@echo "  uninstall                Remove Guild from Go bin directory"
	@echo ""
	@echo "=== DEVELOPER TARGETS ==="
	@echo "  build                    Full build with go vet validation"
	@echo "  test                     Run unit tests with visual feedback"
	@echo "  test-pkg                 Show examples of testing specific packages"
	@echo "  test-teatest             Run TUI tests with proper terminal cleanup"
	@echo "  integration              Run integration tests"
	@echo "  e2e                      Run end-to-end tests"
	@echo "  validate-demo            Validate demo scripts and functionality"
	@echo "  benchmark                Run comprehensive performance benchmarks"
	@echo "  benchmark-suggestions    Run suggestion system benchmarks only"
	@echo "  clean                    Remove all build artifacts"
	@echo "  all                      Clean, build, test, and integration"
	@echo "  quick                    Fast build without visuals"
	@echo "  ci-*                     CI variants (no colors)"
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
	@echo "  guild chat        # Start chatting immediately"
	@echo ""
	@echo "=== DEVELOPMENT WORKFLOW ==="
	@echo "  make build        # Full validation build"
	@echo "  make test         # Run all tests properly"
	@echo "  make ci-build     # Build for CI (plain text)"