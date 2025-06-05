# Guild Framework Makefile
# Provides clean dashboard summaries for build and test operations

.PHONY: all test build clean help dashboard-test provider-test coverage lint install-tools quick-test integration-test verify build-all test-race test-timeout test-full check-stray-binaries test-with-progress providers-dashboard health

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
	@echo "  make build-all        - Build everything and verify"
	@echo "  make verify           - Verify build and basic functionality"
	@echo "  make test             - Run all tests with verbose output"
	@echo "  make test-full        - Run comprehensive test suite with race detection"
	@echo "  make dashboard-test   - Run tests with clean dashboard summary"
	@echo "  make test-with-progress - Run tests with detailed progress indicators"
	@echo "  make quick-test       - Run only unit tests (no integration)"
	@echo "  make health           - Show project health status"
	@echo ""
	@echo "$(GREEN)Provider Testing:$(NC)"
	@echo "  make provider-test    - Test all AI providers"
	@echo "  make provider-test PROVIDER=openai - Test specific provider"
	@echo "  make providers-dashboard - Show provider test dashboard"
	@echo ""
	@echo "$(GREEN)Quality Commands:$(NC)"
	@echo "  make test-race        - Run tests with race detection"
	@echo "  make test-timeout     - Run tests with timeout protection"
	@echo "  make coverage         - Run tests with coverage report"
	@echo "  make lint             - Run linters and format checks"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make fmt              - Format all Go code"
	@echo "  make check-stray-binaries - Check for binaries outside bin/"
	@echo ""
	@echo "$(GREEN)Development:$(NC)"
	@echo "  make install-tools    - Install required development tools"
	@echo "  make integration-test - Run integration tests (requires API keys)"
	@echo "  make test-sqlite      - Run SQLite storage tests"
	@echo "  make test-integration-all - Run all integration tests including SQLite"
	@echo "  make build-examples   - Build example files with special tags"
	@echo "  make status           - Show project status"
	@echo "  make test-failures    - Show detailed test failures"
	@echo "  make test-file FILE=path/to/test.go - Test specific file"

# Build the project
build:
	@echo "$(BLUE)Building Guild CLI...$(NC)"
	@mkdir -p bin/
	@rm -f ./guild  # Clean any binary in root directory
	@go build -o bin/guild ./cmd/guild || (echo "$(RED)Build failed$(NC)" && exit 1)
	@echo "$(GREEN)✓ Build successful$(NC)"

# Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -rf coverage/
	@rm -f ./guild  # Remove any binary in root directory
	@go clean -testcache
	@go clean -cache
	@echo "$(GREEN)✓ Clean complete$(NC)"

# Install development tools
install-tools:
	@echo "$(BLUE)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install gotest.tools/gotestsum@latest
	@echo "$(GREEN)✓ Tools installed$(NC)"

# Run all tests with dashboard summary
dashboard-test:
	@echo "$(BLUE)┌─────────────────────────────────────────┐$(NC)"
	@echo "$(BLUE)│           🏰 Guild Test Suite           │$(NC)"
	@echo "$(BLUE)└─────────────────────────────────────────┘$(NC)"
	@echo ""
	@# Create temp file for results
	@rm -f .test-results.tmp .test-timing.tmp
	@START_TIME=$$(date +%s) ; \
	echo "$$START_TIME" > .test-timing.tmp
	@# Test Core Packages
	@echo "$(YELLOW)┌─ Core Components ─────────────────────┐$(NC)"
	@CORE_PASS=0; CORE_TOTAL=0; \
	for pkg in agent memory orchestrator objective kanban project campaign storage; do \
		CORE_TOTAL=$$((CORE_TOTAL + 1)); \
		printf "$(BLUE)│$(NC) %-18s" "$$pkg" ; \
		if go test -short -count=1 ./pkg/$$pkg/... > /dev/null 2>&1; then \
			echo "$(GREEN)✓ PASS $(NC)$(BLUE)│$(NC)" ; \
			echo "PASS $$pkg" >> .test-results.tmp ; \
			CORE_PASS=$$((CORE_PASS + 1)); \
		else \
			echo "$(RED)✗ FAIL $(NC)$(BLUE)│$(NC)" ; \
			echo "FAIL $$pkg" >> .test-results.tmp ; \
		fi ; \
	done; \
	echo "$(YELLOW)└─ $$CORE_PASS/$$CORE_TOTAL passed ──────────────────────┘$(NC)"
	@echo ""
	@# Test Providers
	@echo "$(YELLOW)┌─ AI Providers ────────────────────────┐$(NC)"
	@PROV_PASS=0; PROV_TOTAL=0; \
	for provider in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		if [ -d "./pkg/providers/$$provider" ]; then \
			PROV_TOTAL=$$((PROV_TOTAL + 1)); \
			printf "$(BLUE)│$(NC) %-18s" "$$provider" ; \
			if go test -short -count=1 ./pkg/providers/$$provider > /dev/null 2>&1; then \
				echo "$(GREEN)✓ PASS $(NC)$(BLUE)│$(NC)" ; \
				echo "PASS provider-$$provider" >> .test-results.tmp ; \
				PROV_PASS=$$((PROV_PASS + 1)); \
			else \
				echo "$(RED)✗ FAIL $(NC)$(BLUE)│$(NC)" ; \
				echo "FAIL provider-$$provider" >> .test-results.tmp ; \
			fi ; \
		fi ; \
	done; \
	echo "$(YELLOW)└─ $$PROV_PASS/$$PROV_TOTAL passed ──────────────────────┘$(NC)"
	@echo ""
	@# Test Other Components
	@echo "$(YELLOW)┌─ Support Systems ─────────────────────┐$(NC)"
	@OTHER_PASS=0; OTHER_TOTAL=0; \
	for pkg in registry context ui tools corpus config grpc mcp prompts; do \
		if [ -d "./pkg/$$pkg" ]; then \
			OTHER_TOTAL=$$((OTHER_TOTAL + 1)); \
			printf "$(BLUE)│$(NC) %-18s" "$$pkg" ; \
			if go test -short -count=1 ./pkg/$$pkg/... > /dev/null 2>&1; then \
				echo "$(GREEN)✓ PASS $(NC)$(BLUE)│$(NC)" ; \
				echo "PASS $$pkg" >> .test-results.tmp ; \
				OTHER_PASS=$$((OTHER_PASS + 1)); \
			else \
				echo "$(RED)✗ FAIL $(NC)$(BLUE)│$(NC)" ; \
				echo "FAIL $$pkg" >> .test-results.tmp ; \
			fi ; \
		fi ; \
	done; \
	echo "$(YELLOW)└─ $$OTHER_PASS/$$OTHER_TOTAL passed ──────────────────────┘$(NC)"
	@echo ""
	@# Calculate timing
	@START_TIME=$$(cat .test-timing.tmp) ; \
	END_TIME=$$(date +%s) ; \
	DURATION=$$((END_TIME - START_TIME))
	@# Summary with box
	@echo "$(BLUE)┌─────────────────────────────────────────┐$(NC)"
	@TOTAL=$$(cat .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	PASSED=$$(grep "^PASS" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	FAILED=$$(grep "^FAIL" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	DURATION=$$(cat .test-timing.tmp 2>/dev/null | xargs -I {} expr $$(date +%s) - {} 2>/dev/null || echo "0") ; \
	if [ "$$FAILED" -eq "0" ]; then \
		printf "$(BLUE)│$(GREEN) ✓ ALL TESTS PASSED!$(NC)" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 21)) "" ; \
		printf "$(BLUE)│$(NC)   Completed: $$PASSED/$$TOTAL packages" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 23 - $$(echo "$$PASSED/$$TOTAL" | wc -c))) "" ; \
		printf "$(BLUE)│$(NC)   Duration:  $${DURATION}s" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 12 - $$(echo "$$DURATION" | wc -c))) "" ; \
	else \
		printf "$(BLUE)│$(RED) ✗ TESTS FAILED$(NC)" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 16)) "" ; \
		printf "$(BLUE)│$(NC)   Failed: $$FAILED, Passed: $$PASSED" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 15 - $$(echo "$$FAILED" | wc -c) - $$(echo "$$PASSED" | wc -c))) "" ; \
		printf "$(BLUE)│$(NC)   Duration: $${DURATION}s" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 11 - $$(echo "$$DURATION" | wc -c))) "" ; \
	fi
	@echo "$(BLUE)└─────────────────────────────────────────┘$(NC)"
	@# Show failed packages if any
	@FAILED=$$(grep "^FAIL" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	if [ "$$FAILED" -gt "0" ]; then \
		echo "" ; \
		echo "$(RED)Failed packages:$(NC)" ; \
		grep "^FAIL" .test-results.tmp | cut -d' ' -f2 | while read pkg; do \
			echo "$(RED)  ✗$(NC) $$pkg" ; \
		done ; \
		rm -f .test-results.tmp .test-timing.tmp ; \
		exit 1 ; \
	fi
	@rm -f .test-results.tmp .test-timing.tmp

# Run tests with verbose output
test:
	@echo "$(BLUE)Running all tests (verbose)...$(NC)"
	go test -v -race -count=1 ./...

# Quick test - only unit tests, no integration
quick-test:
	@echo "$(BLUE)Running quick tests (unit only)...$(NC)"
	go test -short -race -count=1 ./...

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
	@go vet ./...
	@go fmt ./...
	@# Check for unformatted files
	@UNFORMATTED=$$(gofmt -l .) ; \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "$(RED)ERROR: The following files are not properly formatted:$(NC)" ; \
		echo "$$UNFORMATTED" ; \
		exit 1 ; \
	fi ; \
	echo "$(GREEN)✓ All files properly formatted$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./... ; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Run 'make install-tools' first.$(NC)" ; \
	fi
	@make -s check-stray-binaries

# Integration tests (requires API keys for provider tests)
integration-test:
	@echo "$(BLUE)Running integration tests...$(NC)"
	@echo ""
	@echo "$(YELLOW)Running storage integration tests...$(NC)"
	@go test -v ./integration/storage/...
	@echo ""
	@echo "$(YELLOW)Running commission integration tests...$(NC)"
	@go test -v ./integration/commission/...
	@echo ""
	@echo "$(YELLOW)Running chat integration tests...$(NC)"
	@go test -v ./integration/chat/...
	@echo ""
	@echo "$(YELLOW)Running corpus integration tests...$(NC)"
	@go test -v -tags=integration ./integration/corpus/...
	@echo ""
	@echo "$(YELLOW)Running RAG integration tests...$(NC)"
	@go test -v -tags=integration ./integration/rag/...
	@echo ""
	@echo "$(YELLOW)Running project integration tests...$(NC)"
	@go test -v -tags=integration ./pkg/project/...
	@echo ""
	@echo "$(YELLOW)Running provider integration tests (requires API keys)...$(NC)"
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

# Generate protobuf code
proto:
	@echo "$(BLUE)Generating protobuf code...$(NC)"
	@mkdir -p pkg/grpc/pb
	@protoc -I proto \
		--go_out=pkg/grpc/pb --go_opt=paths=source_relative \
		--go-grpc_out=pkg/grpc/pb --go-grpc_opt=paths=source_relative \
		proto/guild/v1/*.proto
	@echo "$(GREEN)✓ Protobuf generation complete$(NC)"

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
	@echo "$(BLUE)┌─────────────────────────────────────────┐$(NC)"
	@echo "$(BLUE)│        🤖 AI Provider Test Suite        │$(NC)"
	@echo "$(BLUE)└─────────────────────────────────────────┘$(NC)"
	@echo ""
	@echo "$(YELLOW)┌─ AI Providers ────────────────────────┐$(NC)"
	@rm -f .test-results.tmp
	@TOTAL=0; PASSED=0; \
	for provider in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		if [ -d "./pkg/providers/$$provider" ]; then \
			TOTAL=$$((TOTAL + 1)); \
			printf "$(BLUE)│$(NC) %-18s" "$$provider" ; \
			if go test -short -count=1 ./pkg/providers/$$provider > /dev/null 2>&1; then \
				echo "$(GREEN)✓ PASS $(NC)$(BLUE)│$(NC)" ; \
				echo "PASS $$provider" >> .test-results.tmp ; \
				PASSED=$$((PASSED + 1)); \
			else \
				echo "$(RED)✗ FAIL $(NC)$(BLUE)│$(NC)" ; \
				echo "FAIL $$provider" >> .test-results.tmp ; \
			fi ; \
		fi ; \
	done; \
	echo "$(YELLOW)└─ $$PASSED/$$TOTAL providers passed ──────────────┘$(NC)"
	@echo ""
	@echo "$(BLUE)┌─────────────────────────────────────────┐$(NC)"
	@TOTAL=$$(cat .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	PASSED=$$(grep "^PASS" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	FAILED=$$(grep "^FAIL" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	if [ "$$FAILED" -eq "0" ]; then \
		printf "$(BLUE)│$(GREEN) ✓ ALL PROVIDERS PASSED!$(NC)" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 23)) "" ; \
		printf "$(BLUE)│$(NC)   Tested: $$PASSED/$$TOTAL providers" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 18 - $$(echo "$$PASSED/$$TOTAL" | wc -c))) "" ; \
	else \
		printf "$(BLUE)│$(RED) ✗ PROVIDER TESTS FAILED$(NC)" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 25)) "" ; \
		printf "$(BLUE)│$(NC)   Failed: $$FAILED, Passed: $$PASSED" ; \
		printf "%*s$(BLUE)│$(NC)\n" $$((40 - 15 - $$(echo "$$FAILED" | wc -c) - $$(echo "$$PASSED" | wc -c))) "" ; \
	fi
	@echo "$(BLUE)└─────────────────────────────────────────┘$(NC)"
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
	@echo "$(BLUE)┌─────────────────────────────────────────┐$(NC)"
	@echo "$(BLUE)│       🏥 Guild Framework Health         │$(NC)"
	@echo "$(BLUE)└─────────────────────────────────────────┘$(NC)"
	@echo ""
	@# Build Status Check
	@echo "$(YELLOW)┌─ Build Health ────────────────────────┐$(NC)"
	@printf "$(BLUE)│$(NC) build status        " ; \
	if make -s build > /dev/null 2>&1; then \
		echo "$(GREEN)✓ PASS $(NC)$(BLUE)│$(NC)" ; \
	else \
		echo "$(RED)✗ FAIL $(NC)$(BLUE)│$(NC)" ; \
	fi
	@printf "$(BLUE)│$(NC) binary exists       " ; \
	if [ -f "bin/guild" ]; then \
		echo "$(GREEN)✓ PASS $(NC)$(BLUE)│$(NC)" ; \
	else \
		echo "$(RED)✗ FAIL $(NC)$(BLUE)│$(NC)" ; \
	fi
	@printf "$(BLUE)│$(NC) binary executable   " ; \
	if [ -x "bin/guild" ]; then \
		echo "$(GREEN)✓ PASS $(NC)$(BLUE)│$(NC)" ; \
	else \
		echo "$(RED)✗ FAIL $(NC)$(BLUE)│$(NC)" ; \
	fi
	@echo "$(YELLOW)└───────────────────────────────────────┘$(NC)"
	@echo ""
	@# Quick Provider Check
	@echo "$(YELLOW)┌─ Provider Health ─────────────────────┐$(NC)"
	@PROVIDER_TOTAL=0; PROVIDER_PASS=0; \
	for provider in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		if [ -d "./pkg/providers/$$provider" ]; then \
			PROVIDER_TOTAL=$$((PROVIDER_TOTAL + 1)); \
			if go test -short -count=1 ./pkg/providers/$$provider > /dev/null 2>&1; then \
				PROVIDER_PASS=$$((PROVIDER_PASS + 1)); \
			fi ; \
		fi ; \
	done; \
	printf "$(BLUE)│$(NC) providers tested     " ; \
	if [ "$$PROVIDER_PASS" -eq "$$PROVIDER_TOTAL" ]; then \
		echo "$(GREEN)✓ $$PROVIDER_PASS/$$PROVIDER_TOTAL $(NC)$(BLUE)│$(NC)" ; \
	else \
		echo "$(RED)✗ $$PROVIDER_PASS/$$PROVIDER_TOTAL $(NC)$(BLUE)│$(NC)" ; \
	fi
	@echo "$(YELLOW)└───────────────────────────────────────┘$(NC)"
	@echo ""
	@# System Info
	@echo "$(YELLOW)┌─ System Info ─────────────────────────┐$(NC)"
	@GO_VERSION=$$(go version | cut -d' ' -f3 | sed 's/go//') ; \
	printf "$(BLUE)│$(NC) go version          %-11s$(BLUE)│$(NC)\n" "$$GO_VERSION"
	@PKG_COUNT=$$(find ./pkg -type d -name "*" | grep -v ".git" | wc -l | tr -d ' ') ; \
	printf "$(BLUE)│$(NC) packages            %-11s$(BLUE)│$(NC)\n" "$$PKG_COUNT"
	@PROV_COUNT=$$(ls -d ./pkg/providers/*/ 2>/dev/null | grep -v -E "(interfaces|testing|mocks|base)" | wc -l | tr -d ' ') ; \
	printf "$(BLUE)│$(NC) providers           %-11s$(BLUE)│$(NC)\n" "$$PROV_COUNT"
	@if [ -f "go.mod" ]; then \
		DEP_COUNT=$$(grep -c "^[[:space:]]*[^[:space:]]*[[:space:]]*v" go.mod | tr -d ' ') ; \
		printf "$(BLUE)│$(NC) dependencies        %-11s$(BLUE)│$(NC)\n" "$$DEP_COUNT" ; \
	fi
	@echo "$(YELLOW)└───────────────────────────────────────┘$(NC)"

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

# Documentation commands
docs-serve: ## Run local documentation server with pkgsite
	@echo "$(BLUE)Starting documentation server...$(NC)"
	@go install golang.org/x/pkgsite/cmd/pkgsite@latest 2>/dev/null || true
	@echo "$(GREEN)Documentation server starting at http://localhost:8080/github.com/guild-ventures/guild-core$(NC)"
	@pkgsite -http=:8080

docs-generate: ## Generate API documentation
	@echo "$(BLUE)Generating API documentation...$(NC)"
	@mkdir -p docs/api
	@go doc -all ./... > docs/api/generated-api-reference.txt
	@echo "$(GREEN)✓ API documentation generated at docs/api/generated-api-reference.txt$(NC)"

# Verify build and basic functionality
verify: build
	@echo "$(BLUE)Verifying Guild binary...$(NC)"
	@if [ ! -f "bin/guild" ]; then \
		echo "$(RED)ERROR: Binary not found at bin/guild$(NC)" ; \
		exit 1 ; \
	fi
	@echo "$(GREEN)✓ Binary exists$(NC)"
	@# Test that the binary runs without crashing
	@if bin/guild --help > /dev/null 2>&1; then \
		echo "$(GREEN)✓ Binary runs and shows help$(NC)" ; \
	else \
		echo "$(RED)ERROR: Binary failed to run or show help$(NC)" ; \
		exit 1 ; \
	fi
	@echo "$(GREEN)✓ Build verification complete$(NC)"

# Build everything and verify
build-all: clean build verify
	@echo "$(GREEN)✓ Complete build and verification finished$(NC)"

# Run tests with race detection
test-race:
	@echo "$(BLUE)Running tests with race detection...$(NC)"
	@go test -race ./...

# Run tests with timeout to catch hanging tests
test-timeout:
	@echo "$(BLUE)Running tests with timeout...$(NC)"
	@go test -timeout=2m ./...

# Run comprehensive test suite
test-full: test-race test-timeout lint
	@echo "$(GREEN)✓ Full test suite completed$(NC)"

# Check for stray binaries
check-stray-binaries:
	@echo "$(BLUE)Checking for stray binaries...$(NC)"
	@STRAY_BINARIES=$$(find . -type f -name "guild" -o -name "*.exe" | grep -v bin/ | head -5) ; \
	if [ -n "$$STRAY_BINARIES" ]; then \
		echo "$(RED)ERROR: Found stray binaries outside bin/ directory:$(NC)" ; \
		echo "$$STRAY_BINARIES" ; \
		exit 1 ; \
	fi ; \
	echo "$(GREEN)✓ No stray binaries found$(NC)"

# Run tests with progress indicators for long-running tests
test-with-progress:
	@echo "$(BLUE)┌─────────────────────────────────────────┐$(NC)"
	@echo "$(BLUE)│      🏰 Guild Test Suite (Detailed)    │$(NC)"
	@echo "$(BLUE)└─────────────────────────────────────────┘$(NC)"
	@echo ""
	@ALL_PACKAGES=$$(go list ./... | grep -v /vendor/ | wc -l | tr -d ' ') ; \
	echo "$(YELLOW)Running tests for $$ALL_PACKAGES packages...$(NC)" ; \
	echo "" ; \
	CURRENT=0 ; \
	go list ./... | grep -v /vendor/ | while read pkg; do \
		CURRENT=$$((CURRENT + 1)) ; \
		PKG_NAME=$$(echo $$pkg | sed 's|github.com/guild-ventures/guild-core/||' | sed 's|github.com/guild-ventures/guild-core||') ; \
		if [ -z "$$PKG_NAME" ]; then PKG_NAME="root"; fi ; \
		printf "$(BLUE)[$$CURRENT/$$ALL_PACKAGES]$(NC) %-30s " "$$PKG_NAME" ; \
		if go test -short -count=1 $$pkg > /dev/null 2>&1; then \
			echo "$(GREEN)✓$(NC)" ; \
		else \
			echo "$(RED)✗$(NC)" ; \
		fi ; \
	done

# Test SQLite storage specifically
test-sqlite:
	@echo "$(BLUE)Running SQLite storage tests...$(NC)"
	@echo ""
	@echo "$(YELLOW)Testing storage package...$(NC)"
	@go test -v ./pkg/storage/...
	@echo ""
	@echo "$(YELLOW)Testing storage integration...$(NC)"
	@go test -v ./integration/storage/...
	@echo ""
	@echo "$(GREEN)✓ SQLite tests complete$(NC)"

# Test all integration tests including storage
test-integration-all: test-sqlite integration-test
	@echo "$(GREEN)✓ All integration tests complete$(NC)"

# Build examples separately (they have build tags)
build-examples:
	@echo "$(BLUE)Building examples...$(NC)"
	@go build -tags example -o bin/commission_example ./examples/commission_refinement_example.go
	@echo "$(GREEN)✓ Examples built successfully$(NC)"

# Default make behavior
.DEFAULT_GOAL := help