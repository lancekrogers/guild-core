# Guild Framework Makefile
# Provides clean dashboard summaries for build and test operations

.PHONY: all test build clean help dashboard-test provider-test coverage lint install-tools quick-test integration-test

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Default target
all: clean build dashboard-test

# Help command
help:
	@echo "$(BLUE)Guild Framework - Make Commands$(NC)"
	@echo ""
	@echo "$(GREEN)Main Commands:$(NC)"
	@echo "  make all              - Clean, build, and run all tests with dashboard"
	@echo "  make build            - Build the Guild CLI"
	@echo "  make test             - Run all tests with verbose output"
	@echo "  make dashboard-test   - Run tests with clean dashboard summary"
	@echo "  make quick-test       - Run only unit tests (no integration)"
	@echo "  make health           - Show project health status"
	@echo ""
	@echo "$(GREEN)Provider Testing:$(NC)"
	@echo "  make provider-test    - Test all AI providers"
	@echo "  make provider-test PROVIDER=openai - Test specific provider"
	@echo "  make providers-dashboard - Show provider test dashboard"
	@echo ""
	@echo "$(GREEN)Quality Commands:$(NC)"
	@echo "  make coverage         - Run tests with coverage report"
	@echo "  make lint             - Run linters"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make fmt              - Format all Go code"
	@echo ""
	@echo "$(GREEN)Development:$(NC)"
	@echo "  make install-tools    - Install required development tools"
	@echo "  make integration-test - Run integration tests (requires API keys)"
	@echo "  make status           - Show project status"
	@echo "  make test-failures    - Show detailed test failures"
	@echo "  make test-file FILE=path/to/test.go - Test specific file"

# Build the project
build:
	@echo "$(BLUE)Building Guild CLI...$(NC)"
	@go build -o bin/guild ./cmd/guild || (echo "$(RED)Build failed$(NC)" && exit 1)
	@echo "$(GREEN)✓ Build successful$(NC)"

# Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -rf coverage/
	@go clean -testcache
	@echo "$(GREEN)✓ Clean complete$(NC)"

# Install development tools
install-tools:
	@echo "$(BLUE)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install gotest.tools/gotestsum@latest
	@echo "$(GREEN)✓ Tools installed$(NC)"

# Run all tests with dashboard summary
dashboard-test:
	@echo "$(BLUE)Running Tests - Dashboard View$(NC)"
	@echo "================================"
	@echo ""
	@# Create temp file for results
	@rm -f .test-results.tmp
	@# Run tests for each major package and capture results
	@echo "$(YELLOW)Testing Core Packages...$(NC)"
	@for pkg in agent memory orchestrator objective kanban; do \
		printf "  %-20s" "$$pkg:" ; \
		if go test -short -count=1 ./pkg/$$pkg/... > /dev/null 2>&1; then \
			echo "$(GREEN)✓ PASS$(NC)" ; \
			echo "PASS $$pkg" >> .test-results.tmp ; \
		else \
			echo "$(RED)✗ FAIL$(NC)" ; \
			echo "FAIL $$pkg" >> .test-results.tmp ; \
		fi ; \
	done
	@echo ""
	@echo "$(YELLOW)Testing Providers...$(NC)"
	@for provider in mock anthropic deepseek deepinfra ollama ora openai; do \
		printf "  %-20s" "$$provider:" ; \
		if [ -d "./pkg/providers/$$provider" ]; then \
			if go test -short -count=1 ./pkg/providers/$$provider > /dev/null 2>&1; then \
				echo "$(GREEN)✓ PASS$(NC)" ; \
				echo "PASS provider-$$provider" >> .test-results.tmp ; \
			else \
				echo "$(RED)✗ FAIL$(NC)" ; \
				echo "FAIL provider-$$provider" >> .test-results.tmp ; \
			fi ; \
		else \
			echo "$(YELLOW)- SKIP$(NC)" ; \
		fi ; \
	done
	@echo ""
	@echo "$(YELLOW)Testing Other Components...$(NC)"
	@for pkg in registry context ui tools corpus; do \
		printf "  %-20s" "$$pkg:" ; \
		if [ -d "./pkg/$$pkg" ]; then \
			if go test -short -count=1 ./pkg/$$pkg/... > /dev/null 2>&1; then \
				echo "$(GREEN)✓ PASS$(NC)" ; \
				echo "PASS $$pkg" >> .test-results.tmp ; \
			else \
				echo "$(RED)✗ FAIL$(NC)" ; \
				echo "FAIL $$pkg" >> .test-results.tmp ; \
			fi ; \
		else \
			echo "$(YELLOW)- SKIP$(NC)" ; \
		fi ; \
	done
	@echo ""
	@echo "================================"
	@# Summary
	@TOTAL=$$(cat .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	PASSED=$$(grep "^PASS" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	FAILED=$$(grep "^FAIL" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	if [ "$$FAILED" -eq "0" ]; then \
		echo "$(GREEN)✓ All tests passed! ($$PASSED/$$TOTAL)$(NC)" ; \
	else \
		echo "$(RED)✗ Tests failed: $$FAILED failed, $$PASSED passed ($$TOTAL total)$(NC)" ; \
		echo "" ; \
		echo "$(YELLOW)Failed packages:$(NC)" ; \
		grep "^FAIL" .test-results.tmp | cut -d' ' -f2 | while read pkg; do \
			echo "  - $$pkg" ; \
		done ; \
		rm -f .test-results.tmp ; \
		exit 1 ; \
	fi
	@rm -f .test-results.tmp

# Run tests with verbose output
test:
	@echo "$(BLUE)Running all tests (verbose)...$(NC)"
	go test -v ./...

# Quick test - only unit tests, no integration
quick-test:
	@echo "$(BLUE)Running quick tests (unit only)...$(NC)"
	go test -short ./...

# Test specific provider
provider-test:
ifdef PROVIDER
	@echo "$(BLUE)Testing provider: $(PROVIDER)$(NC)"
	@go test -v ./pkg/providers/$(PROVIDER)/...
else
	@echo "$(BLUE)Testing all providers...$(NC)"
	@for provider in mock anthropic deepseek deepinfra ollama ora openai; do \
		echo "$(YELLOW)Testing $$provider...$(NC)" ; \
		go test -short ./pkg/providers/$$provider/... || true ; \
		echo "" ; \
	done
endif

# Run tests with coverage
coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage/coverage.html$(NC)"
	@echo ""
	@echo "$(YELLOW)Coverage Summary:$(NC)"
	@go tool cover -func=coverage/coverage.out | grep total | awk '{print "  Total Coverage: " $$3}'

# Run linters
lint:
	@echo "$(BLUE)Running linters...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./... ; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Run 'make install-tools' first.$(NC)" ; \
		go vet ./... ; \
	fi

# Integration tests (requires API keys)
integration-test:
	@echo "$(BLUE)Running integration tests...$(NC)"
	@echo "$(YELLOW)Note: This requires API keys to be set$(NC)"
	@echo ""
	@# Check for API keys
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "$(YELLOW)⚠ OPENAI_API_KEY not set - skipping OpenAI integration tests$(NC)" ; \
	else \
		echo "$(GREEN)✓ Testing OpenAI integration...$(NC)" ; \
		go test -v -run TestLive ./pkg/providers/openai || true ; \
	fi
	@if [ -z "$$ANTHROPIC_API_KEY" ]; then \
		echo "$(YELLOW)⚠ ANTHROPIC_API_KEY not set - skipping Anthropic integration tests$(NC)" ; \
	else \
		echo "$(GREEN)✓ Testing Anthropic integration...$(NC)" ; \
		go test -v -run TestLive ./pkg/providers/anthropic || true ; \
	fi

# Benchmark tests
bench:
	@echo "$(BLUE)Running benchmarks...$(NC)"
	go test -bench=. -benchmem ./...

# Check for outdated dependencies
check-deps:
	@echo "$(BLUE)Checking dependencies...$(NC)"
	@go list -u -m all | grep -v "^github.com/guild-ventures/guild-core" | grep "\[" || echo "$(GREEN)✓ All dependencies up to date$(NC)"

# Format code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

# Development server with hot reload (using Task)
dev:
	@echo "$(BLUE)Starting development server...$(NC)"
	@if command -v task >/dev/null 2>&1; then \
		task ui:dev:run ; \
	else \
		echo "$(RED)Task not installed. Please install Task from https://taskfile.dev$(NC)" ; \
		exit 1 ; \
	fi

# Show test failures in detail
test-failures:
	@echo "$(BLUE)Running tests and showing failures...$(NC)"
	@go test ./... -json | grep -E '"Action":"fail"' | jq -r '.Package + ": " + .Test + " - " + .Output' || echo "$(GREEN)No test failures!$(NC)"

# Test only providers with dashboard
providers-dashboard:
	@echo "$(BLUE)Provider Tests - Dashboard View$(NC)"
	@echo "================================"
	@rm -f .test-results.tmp
	@for provider in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		printf "  %-20s" "$$provider:" ; \
		if [ -d "./pkg/providers/$$provider" ]; then \
			if go test -short -count=1 ./pkg/providers/$$provider > /dev/null 2>&1; then \
				echo "$(GREEN)✓ PASS$(NC)" ; \
				echo "PASS $$provider" >> .test-results.tmp ; \
			else \
				echo "$(RED)✗ FAIL$(NC)" ; \
				echo "FAIL $$provider" >> .test-results.tmp ; \
			fi ; \
		else \
			echo "$(YELLOW)- N/A$(NC)" ; \
		fi ; \
	done
	@echo "================================"
	@TOTAL=$$(cat .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	PASSED=$$(grep "^PASS" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	FAILED=$$(grep "^FAIL" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	if [ "$$FAILED" -eq "0" ]; then \
		echo "$(GREEN)✓ All provider tests passed! ($$PASSED/$$TOTAL)$(NC)" ; \
	else \
		echo "$(RED)✗ Provider tests: $$FAILED failed, $$PASSED passed$(NC)" ; \
	fi
	@rm -f .test-results.tmp

# Quick status check
status:
	@echo "$(BLUE)Project Status$(NC)"
	@echo "=============="
	@echo ""
	@echo "$(YELLOW)Git Status:$(NC)"
	@git status -s || echo "  Clean working directory"
	@echo ""
	@echo "$(YELLOW)Build Status:$(NC)"
	@if [ -f "bin/guild" ]; then \
		echo "  $(GREEN)✓ Binary exists$(NC)" ; \
		ls -lh bin/guild | awk '{print "  Size: " $$5 ", Modified: " $$6 " " $$7 " " $$8}' ; \
	else \
		echo "  $(RED)✗ Binary not built$(NC)" ; \
	fi
	@echo ""
	@echo "$(YELLOW)Test Cache:$(NC)"
	@CACHE_SIZE=$$(du -sh $$(go env GOCACHE) 2>/dev/null | cut -f1) ; \
	echo "  Cache size: $$CACHE_SIZE"

# Project health check
health:
	@echo "$(BLUE)Guild Framework Health Check$(NC)"
	@echo "============================"
	@echo ""
	@echo "$(YELLOW)Provider Status:$(NC)"
	@make -s providers-dashboard
	@echo ""
	@echo "$(YELLOW)Build Status:$(NC)"
	@if make -s build > /dev/null 2>&1; then \
		echo "  $(GREEN)✓ Build succeeds$(NC)" ; \
	else \
		echo "  $(RED)✗ Build fails$(NC)" ; \
	fi
	@echo ""
	@echo "$(YELLOW)Quick Stats:$(NC)"
	@echo -n "  Go version: " && go version | cut -d' ' -f3
	@echo -n "  Total packages: " && find ./pkg -type d -name "*.go" | wc -l | tr -d ' '
	@echo -n "  Total providers: " && ls -d ./pkg/providers/*/ 2>/dev/null | grep -v -E "(interfaces|testing|mocks|base)" | wc -l | tr -d ' '
	@echo ""

# Run specific test file
test-file:
	@if [ -z "$(FILE)" ]; then \
		echo "$(RED)Usage: make test-file FILE=path/to/test_file.go$(NC)" ; \
		exit 1 ; \
	fi
	@echo "$(BLUE)Testing file: $(FILE)$(NC)"
	@go test -v $(FILE)

# CI/CD friendly test output
ci-test:
	@echo "Running CI tests..."
	@rm -f .ci-test-results.json
	@echo '{"tests": [' > .ci-test-results.json
	@FIRST=1 ; \
	for pkg in $$(go list ./... | grep -v /vendor/); do \
		if [ "$$FIRST" -ne 1 ]; then echo "," >> .ci-test-results.json; fi ; \
		FIRST=0 ; \
		PKG_NAME=$$(echo $$pkg | sed 's|github.com/guild-ventures/guild-core/||') ; \
		if go test -short -count=1 $$pkg > /dev/null 2>&1; then \
			printf '{"package": "%s", "status": "pass"}' "$$PKG_NAME" >> .ci-test-results.json ; \
		else \
			printf '{"package": "%s", "status": "fail"}' "$$PKG_NAME" >> .ci-test-results.json ; \
		fi ; \
	done
	@echo ']' >> .ci-test-results.json
	@echo '}' >> .ci-test-results.json
	@# Output summary
	@TOTAL=$$(cat .ci-test-results.json | grep -o '"package"' | wc -l | tr -d ' ') ; \
	PASSED=$$(cat .ci-test-results.json | grep -o '"status": "pass"' | wc -l | tr -d ' ') ; \
	FAILED=$$(cat .ci-test-results.json | grep -o '"status": "fail"' | wc -l | tr -d ' ') ; \
	echo "Test Results: $$PASSED passed, $$FAILED failed out of $$TOTAL total" ; \
	if [ "$$FAILED" -gt 0 ]; then exit 1; fi

# Default make behavior
.DEFAULT_GOAL := help