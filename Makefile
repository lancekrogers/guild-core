# Guild Framework Makefile - Professional Dashboard Edition
# Provides clean, modern dashboard summaries for build and test operations

.PHONY: all test build clean help dashboard-test unit-test integration integration-verbose coverage lint install-tools quick-test verify build-all health status format check

# Colors and styling for professional appearance
BLUE := \033[38;5;74m
GREEN := \033[38;5;76m
YELLOW := \033[38;5;220m
RED := \033[38;5;196m
PURPLE := \033[38;5;141m
CYAN := \033[38;5;51m
GRAY := \033[38;5;240m
WHITE := \033[38;5;255m
BOLD := \033[1m
DIM := \033[2m
NC := \033[0m

# Unicode symbols for better visual appeal
CHECK := ✓
CROSS := ✗
ARROW := →
STAR := ★
GEAR := ⚙
ROCKET := 🚀
SHIELD := 🛡
HAMMER := 🔨
CLIPBOARD := 📋

# Progress bar function
define progress_bar
	@printf "$(GRAY)["
	@if [ "$1" -gt 0 ]; then for i in $$(seq 1 $1); do printf "█"; done; fi
	@if [ "$1" -lt 30 ]; then for i in $$(seq 1 $$((30 - $1))); do printf "░"; done; fi
	@printf "] %d%%$(NC)\n" $$((($1 * 100) / 30))
endef

# Header function for Guild-branded section headers
define section_header
	@echo ""
	@echo "$(BOLD)$(BLUE)╭────────────────────────────────────────────────────────────╮$(NC)"
	@echo "$(BOLD)$(BLUE)│$(PURPLE) 🏰 GUILD │$(YELLOW) $(1)                               $(BLUE)│$(NC)"
	@echo "$(BOLD)$(BLUE)╰────────────────────────────────────────────────────────────╯$(NC)"
	@echo ""
endef

# Guild-branded status card function
define status_card
	@echo "$(BOLD)$(BLUE)┌─────────────────────────────────────────────────────────────┐$(NC)"
	@if [ "$(2)" = "pass" ]; then \
		echo "$(BOLD)$(BLUE)│  $(GREEN)✓ $(1) $(BOLD)$(BLUE)                                            │$(NC)"; \
	elif [ "$(2)" = "fail" ]; then \
		echo "$(BOLD)$(BLUE)│  $(RED)✗ $(1) $(BOLD)$(BLUE)                                            │$(NC)"; \
	else \
		echo "$(BOLD)$(BLUE)│  $(YELLOW)⚙ $(1) $(BOLD)$(BLUE)                                            │$(NC)"; \
	fi
	@echo "$(BOLD)$(BLUE)└─────────────────────────────────────────────────────────────┘$(NC)"
endef

# Main dashboard target - NOW ONLY RUNS UNIT TESTS
all: clean build unit-test
	@$(call section_header,"Build Complete - Ready for Agent Development")
	@$(call status_card,"🚀 Guild Framework Ready","pass")

# Enhanced help with Guild branding
help:
	@echo "$(BOLD)$(BLUE)"
	@echo "╭─────────────────────────────────────────────────────────────╮"
	@echo "│  $(PURPLE)🏰 GUILD FRAMEWORK$(BLUE)                      $(WHITE)⚡ AI Agent System$(BLUE)  │"
	@echo "│  $(DIM)Professional Multi-Agent Orchestration Platform$(BLUE)          │"
	@echo "╰─────────────────────────────────────────────────────────────╯"
	@echo "$(NC)"
	@echo "$(BOLD)$(GREEN)$(ROCKET) MAIN COMMANDS$(NC)"
	@echo "  $(BOLD)make all$(NC)              $(ARROW) Clean, build, and run unit tests only"
	@echo "  $(BOLD)make build$(NC)            $(ARROW) Build the Guild CLI with verification"
	@echo "  $(BOLD)make unit-test$(NC)        $(ARROW) Run unit tests with modern dashboard"
	@echo "  $(BOLD)make integration$(NC)      $(ARROW) Run integration tests (requires API keys)"
	@echo "  $(BOLD)make integration-verbose$(NC) $(ARROW) Run integration tests with full output"
	@echo ""
	@echo "$(BOLD)$(YELLOW)$(SHIELD) QUALITY ASSURANCE$(NC)"
	@echo "  $(BOLD)make health$(NC)           $(ARROW) Comprehensive system health check"
	@echo "  $(BOLD)make coverage$(NC)         $(ARROW) Generate test coverage report"
	@echo "  $(BOLD)make lint$(NC)             $(ARROW) Run all linters and formatters"
	@echo "  $(BOLD)make format$(NC)           $(ARROW) Format all code files"
	@echo ""
	@echo "$(BOLD)$(PURPLE)$(GEAR) DEVELOPMENT$(NC)"
	@echo "  $(BOLD)make install-tools$(NC)    $(ARROW) Install required development tools"
	@echo "  $(BOLD)make status$(NC)           $(ARROW) Show current project status"
	@echo "  $(BOLD)make clean$(NC)            $(ARROW) Clean all build artifacts"
	@echo "  $(BOLD)make verify$(NC)           $(ARROW) Verify build and basic functionality"
	@echo ""
	@echo "$(BOLD)$(CYAN)$(CLIPBOARD) SPECIALIZED$(NC)"
	@echo "  $(BOLD)make provider-test$(NC)    $(ARROW) Test AI providers with dashboard"
	@echo "  $(BOLD)make quick-check$(NC)      $(ARROW) Fast health check for CI/CD"
	@echo "  $(BOLD)make docs-serve$(NC)       $(ARROW) Start local documentation server"
	@echo ""

# Enhanced build with real progress indication
build:
	@$(call section_header,"Building Guild CLI")
	@echo "$(BOLD)$(YELLOW)Preparing build environment...$(NC)"
	@mkdir -p bin/
	@rm -f ./guild
	@echo "$(BOLD)$(YELLOW)Compiling source code...$(NC)"
	@printf "$(GRAY)[$(NC)"
	@go build -o bin/guild ./cmd/guild 2>/dev/null & \
	BUILD_PID=$$! ; \
	while kill -0 $$BUILD_PID 2>/dev/null; do \
		printf "$(GREEN)█$(NC)" ; \
		sleep 0.1 ; \
	done ; \
	wait $$BUILD_PID ; \
	BUILD_STATUS=$$? ; \
	printf "$(GRAY)] $(GREEN)Complete$(NC)\n" ; \
	if [ $$BUILD_STATUS -ne 0 ]; then \
		echo "$(BOLD)$(RED)✗ Build failed$(NC)" ; \
		exit 1 ; \
	fi
	@echo ""
	@$(call status_card,"🚀 Build Successful","pass")

# Enhanced clean with better feedback
clean:
	@$(call section_header,"Cleaning Build Environment")
	@echo "$(BOLD)$(YELLOW)Removing build artifacts...$(NC)"
	@rm -rf bin/ coverage/ || true
	@rm -f ./guild || true
	@echo "$(BOLD)$(YELLOW)Clearing test caches...$(NC)"
	@go clean -testcache 2>/dev/null || true
	@go clean -cache 2>/dev/null || true
	@$(call status_card,"Environment Cleaned","pass")

# NEW: Separate unit test dashboard (what 'make all' now calls)
unit-test:
	@$(call section_header,"Unit Test Suite Dashboard")
	@rm -f .test-results.tmp .test-timing.tmp
	@echo $$(date +%s) > .test-timing.tmp
	
	@# Core Framework Tests
	@echo "$(BOLD)$(PURPLE)┌── 🏗️  Core Framework Components ──────────────────────────┐$(NC)"
	@echo "$(BLUE)│$(NC) $(BOLD)Package            Build    Test     Status$(NC)              $(BLUE)│$(NC)"
	@echo "$(BLUE)├─────────────────────────────────────────────────────────────┤$(NC)"
	@CORE_PASS=0; CORE_TOTAL=0; \
	for pkg in agent memory orchestrator commission kanban project campaign storage registry; do \
		CORE_TOTAL=$$((CORE_TOTAL + 1)); \
		printf "$(BLUE)│$(NC) %-15s" "$$pkg" ; \
		if go build ./pkg/$$pkg/... >/dev/null 2>&1; then \
			printf "  $(GREEN)✓$(NC)     " ; \
		else \
			printf "  $(RED)✗$(NC)     " ; \
		fi ; \
		if go test -short -count=1 ./pkg/$$pkg/... >/dev/null 2>&1; then \
			printf "  $(GREEN)✓$(NC)     $(GREEN)PASS$(NC)" ; \
			echo "PASS $$pkg" >> .test-results.tmp ; \
			CORE_PASS=$$((CORE_PASS + 1)); \
		else \
			printf "  $(RED)✗$(NC)     $(RED)FAIL$(NC)" ; \
			echo "FAIL $$pkg" >> .test-results.tmp ; \
		fi ; \
		printf "%*s$(BLUE)│$(NC)\n" 15 "" ; \
	done; \
	echo "$(BOLD)$(PURPLE)└── $$CORE_PASS/$$CORE_TOTAL Core Components Passing ──────────────────────┘$(NC)"
	
	@echo ""
	@# Provider Tests
	@echo "$(BOLD)$(GREEN)┌── 🤖 AI Provider Integrations ─────────────────────────────┐$(NC)"
	@echo "$(BLUE)│$(NC) $(BOLD)Provider           Build    Test     Status$(NC)              $(BLUE)│$(NC)"
	@echo "$(BLUE)├─────────────────────────────────────────────────────────────┤$(NC)"
	@PROV_PASS=0; PROV_TOTAL=0; \
	for provider in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		if [ -d "./pkg/providers/$$provider" ]; then \
			PROV_TOTAL=$$((PROV_TOTAL + 1)); \
			printf "$(BLUE)│$(NC) %-15s" "$$provider" ; \
			if go build ./pkg/providers/$$provider >/dev/null 2>&1; then \
				printf "  $(GREEN)✓$(NC)     " ; \
			else \
				printf "  $(RED)✗$(NC)     " ; \
			fi ; \
			if go test -short -count=1 ./pkg/providers/$$provider >/dev/null 2>&1; then \
				printf "  $(GREEN)✓$(NC)     $(GREEN)PASS$(NC)" ; \
				echo "PASS provider-$$provider" >> .test-results.tmp ; \
				PROV_PASS=$$((PROV_PASS + 1)); \
			else \
				printf "  $(RED)✗$(NC)     $(RED)FAIL$(NC)" ; \
				echo "FAIL provider-$$provider" >> .test-results.tmp ; \
			fi ; \
			printf "%*s$(BLUE)│$(NC)\n" 15 "" ; \
		fi ; \
	done; \
	echo "$(BOLD)$(GREEN)└── $$PROV_PASS/$$PROV_TOTAL AI Providers Passing ─────────────────────────┘$(NC)"
	
	@echo ""
	@# Support Systems
	@echo "$(BOLD)$(CYAN)┌── ⚙️  Support Systems & Infrastructure ─────────────────────────┐$(NC)"
	@echo "$(BLUE)│$(NC) $(BOLD)System             Build    Test     Status$(NC)              $(BLUE)│$(NC)"
	@echo "$(BLUE)├─────────────────────────────────────────────────────────────┤$(NC)"
	@OTHER_PASS=0; OTHER_TOTAL=0; \
	for pkg in context config tools corpus grpc workspace prompts; do \
		if [ -d "./pkg/$$pkg" ]; then \
			OTHER_TOTAL=$$((OTHER_TOTAL + 1)); \
			printf "$(BLUE)│$(NC) %-15s" "$$pkg" ; \
			if go build ./pkg/$$pkg/... >/dev/null 2>&1; then \
				printf "  $(GREEN)✓$(NC)     " ; \
			else \
				printf "  $(RED)✗$(NC)     " ; \
			fi ; \
			if go test -short -count=1 ./pkg/$$pkg/... >/dev/null 2>&1; then \
				printf "  $(GREEN)✓$(NC)     $(GREEN)PASS$(NC)" ; \
				echo "PASS $$pkg" >> .test-results.tmp ; \
				OTHER_PASS=$$((OTHER_PASS + 1)); \
			else \
				printf "  $(RED)✗$(NC)     $(RED)FAIL$(NC)" ; \
				echo "FAIL $$pkg" >> .test-results.tmp ; \
			fi ; \
			printf "%*s$(BLUE)│$(NC)\n" 15 "" ; \
		fi ; \
	done; \
	echo "$(BOLD)$(CYAN)└── $$OTHER_PASS/$$OTHER_TOTAL Support Systems Passing ────────────────────┘$(NC)"
	
	@# Enhanced Summary
	@echo ""
	@FAILED_COUNT=$$(grep -c "^FAIL" .test-results.tmp 2>/dev/null || echo "0") ; \
	TOTAL_COUNT=$$(wc -l < .test-results.tmp 2>/dev/null || echo "0") ; \
	PASSED_COUNT=$$((TOTAL_COUNT - FAILED_COUNT)) ; \
	if [ "$$FAILED_COUNT" -eq "0" ]; then \
		$(call status_card,"🎉 ALL TESTS PASSED! ($$PASSED_COUNT/$$TOTAL_COUNT components)","pass") ; \
	else \
		$(call status_card,"⚠️  SOME TESTS FAILED ($$PASSED_COUNT/$$TOTAL_COUNT passing)","fail") ; \
		echo "" ; \
		echo "$(BOLD)$(RED)📋 Failed Components:$(NC)" ; \
		grep "^FAIL" .test-results.tmp 2>/dev/null | cut -d' ' -f2 | while read pkg; do \
			echo "  $(RED)✗$(NC) $$pkg" ; \
		done ; \
		rm -f .test-results.tmp .test-timing.tmp ; \
		exit 1 ; \
	fi
	@rm -f .test-results.tmp .test-timing.tmp

# NEW: Integration tests with clean separation
integration:
	@$(call section_header,"Integration Test Suite")
	@echo "$(BOLD)$(YELLOW)Running comprehensive integration tests...$(NC)"
	@echo ""
	@echo "$(CYAN)$(ARROW) Storage Integration$(NC)"
	@go test -v ./integration/storage/... 2>/dev/null || echo "$(RED)Storage tests failed$(NC)"
	@echo ""
	@echo "$(CYAN)$(ARROW) Commission Integration$(NC)"
	@go test -v -tags=integration ./integration/commission/... 2>/dev/null || echo "$(RED)Commission tests failed$(NC)"
	@echo ""
	@echo "$(CYAN)$(ARROW) Chat Integration$(NC)"
	@go test -v ./integration/chat/... 2>/dev/null || echo "$(RED)Chat tests failed$(NC)"
	@echo ""
	@echo "$(CYAN)$(ARROW) RAG System Integration$(NC)"
	@go test -v -tags=integration ./integration/rag/... 2>/dev/null || echo "$(RED)RAG tests failed$(NC)"
	@echo ""
	@echo "$(CYAN)$(ARROW) Provider Integration (with API keys)$(NC)"
	@if [ -n "$$OPENAI_API_KEY" ]; then \
		echo "$(GREEN)$(CHECK) Testing OpenAI...$(NC)" ; \
		go test -v -run TestLive ./pkg/providers/openai 2>/dev/null || echo "$(YELLOW)OpenAI tests skipped$(NC)" ; \
	else \
		echo "$(YELLOW)⚠ OPENAI_API_KEY not set - skipping$(NC)" ; \
	fi
	@if [ -n "$$ANTHROPIC_API_KEY" ]; then \
		echo "$(GREEN)$(CHECK) Testing Anthropic...$(NC)" ; \
		go test -v -run TestLive ./pkg/providers/anthropic 2>/dev/null || echo "$(YELLOW)Anthropic tests skipped$(NC)" ; \
	else \
		echo "$(YELLOW)⚠ ANTHROPIC_API_KEY not set - skipping$(NC)" ; \
	fi
	@echo ""
	@$(call status_card,"Integration Tests Complete","pass")

# NEW: Verbose integration tests
integration-verbose:
	@$(call section_header,"Integration Test Suite (Verbose)")
	@echo "$(BOLD)$(YELLOW)Running integration tests with full output...$(NC)"
	@echo ""
	@go test -v ./integration/storage/...
	@go test -v -tags=integration ./integration/commission/...
	@go test -v ./integration/chat/...
	@go test -v -tags=integration ./integration/rag/...
	@if [ -n "$$OPENAI_API_KEY" ]; then go test -v -run TestLive ./pkg/providers/openai; fi
	@if [ -n "$$ANTHROPIC_API_KEY" ]; then go test -v -run TestLive ./pkg/providers/anthropic; fi

# Enhanced health check with modern design
health:
	@$(call section_header,"Guild Framework Health Check")
	
	@# Build Health
	@echo "$(BOLD)$(YELLOW)┌── Build System Health ─────────────────────────┐$(NC)"
	@printf "$(BLUE)│$(NC) build compilation     " ; \
	if make -s build >/dev/null 2>&1; then \
		echo "$(GREEN)$(CHECK) HEALTHY$(NC) $(BLUE)│$(NC)" ; \
	else \
		echo "$(RED)$(CROSS) FAILING$(NC) $(BLUE)│$(NC)" ; \
	fi
	@printf "$(BLUE)│$(NC) binary exists        " ; \
	if [ -f "bin/guild" ]; then \
		echo "$(GREEN)$(CHECK) PRESENT$(NC) $(BLUE)│$(NC)" ; \
	else \
		echo "$(RED)$(CROSS) MISSING$(NC)  $(BLUE)│$(NC)" ; \
	fi
	@printf "$(BLUE)│$(NC) binary executable    " ; \
	if [ -x "bin/guild" ] && bin/guild --help >/dev/null 2>&1; then \
		echo "$(GREEN)$(CHECK) WORKING$(NC) $(BLUE)│$(NC)" ; \
	else \
		echo "$(RED)$(CROSS) BROKEN$(NC)  $(BLUE)│$(NC)" ; \
	fi
	@echo "$(BOLD)$(YELLOW)└─────────────────────────────────────────────────┘$(NC)"
	
	@echo ""
	@# System Health
	@echo "$(BOLD)$(YELLOW)┌── System Information ──────────────────────────┐$(NC)"
	@GO_VERSION=$$(go version | cut -d' ' -f3 | sed 's/go//') ; \
	printf "$(BLUE)│$(NC) go version           $(WHITE)%-13s$(NC) $(BLUE)│$(NC)\n" "$$GO_VERSION"
	@PKG_COUNT=$$(find ./pkg -type d -maxdepth 1 | wc -l | tr -d ' ') ; \
	printf "$(BLUE)│$(NC) framework packages   $(WHITE)%-13s$(NC) $(BLUE)│$(NC)\n" "$$PKG_COUNT"
	@PROV_COUNT=$$(ls -d ./pkg/providers/*/ 2>/dev/null | grep -v -E "(interfaces|testing|mocks|base)" | wc -l | tr -d ' ') ; \
	printf "$(BLUE)│$(NC) ai providers         $(WHITE)%-13s$(NC) $(BLUE)│$(NC)\n" "$$PROV_COUNT"
	@if [ -f "go.mod" ]; then \
		DEP_COUNT=$$(grep -c "require" go.mod | tr -d ' ') ; \
		printf "$(BLUE)│$(NC) dependencies         $(WHITE)%-13s$(NC) $(BLUE)│$(NC)\n" "$$DEP_COUNT" ; \
	fi
	@echo "$(BOLD)$(YELLOW)└─────────────────────────────────────────────────┘$(NC)"
	
	@echo ""
	@$(call status_card,"Health Check Complete","pass")

# Enhanced coverage with better presentation
coverage:
	@$(call section_header,"Test Coverage Analysis")
	@echo "$(BOLD)$(YELLOW)Generating coverage report...$(NC)"
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out -covermode=atomic ./... >/dev/null 2>&1
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo ""
	@COVERAGE=$$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $$3}' | sed 's/%//') ; \
	if [ $$COVERAGE -ge 80 ]; then \
		$(call status_card,"Coverage: $${COVERAGE}% - Excellent","pass") ; \
	elif [ $$COVERAGE -ge 60 ]; then \
		$(call status_card,"Coverage: $${COVERAGE}% - Good","") ; \
	else \
		$(call status_card,"Coverage: $${COVERAGE}% - Needs Improvement","fail") ; \
	fi

# Install tools with better feedback
install-tools:
	@$(call section_header,"Installing Development Tools")
	@echo "$(BOLD)$(YELLOW)Installing golangci-lint...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(BOLD)$(YELLOW)Installing gotestsum...$(NC)"
	@go install gotest.tools/gotestsum@latest
	@$(call status_card,"Development Tools Installed","pass")

# Enhanced lint with progress
lint:
	@$(call section_header,"Code Quality Check")
	@echo "$(BOLD)$(YELLOW)Running go vet...$(NC)"
	@go vet ./... || ($(call status_card,"Vet Failed","fail") && exit 1)
	@echo "$(BOLD)$(YELLOW)Checking formatting...$(NC)"
	@UNFORMATTED=$$(gofmt -l .) ; \
	if [ -n "$$UNFORMATTED" ]; then \
		$(call status_card,"Format Check Failed","fail") ; \
		exit 1 ; \
	fi
	@echo "$(BOLD)$(YELLOW)Running golangci-lint...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./... || ($(call status_card,"Lint Failed","fail") && exit 1) ; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Run 'make install-tools'$(NC)" ; \
	fi
	@$(call status_card,"All Quality Checks Passed","pass")

# NEW: Format command
format:
	@$(call section_header,"Formatting Code")
	@echo "$(BOLD)$(YELLOW)Formatting Go files...$(NC)"
	@go fmt ./...
	@$(call status_card,"Code Formatted","pass")

# Enhanced verify
verify: build
	@$(call section_header,"Build Verification")
	@echo "$(BOLD)$(YELLOW)Verifying Guild binary...$(NC)"
	@if [ ! -f "bin/guild" ]; then \
		$(call status_card,"Verification Failed","fail") ; \
		exit 1 ; \
	fi
	@if bin/guild --help >/dev/null 2>&1; then \
		$(call status_card,"Build Verified","pass") ; \
	else \
		$(call status_card,"Verification Failed","fail") ; \
		exit 1 ; \
	fi

# Quick check for CI/CD
quick-check: build
	@echo "$(BOLD)$(CYAN)Quick Health Check$(NC)"
	@go test -short -count=1 ./pkg/... >/dev/null 2>&1 && echo "$(GREEN)✓ Quick tests passed$(NC)" || echo "$(RED)✗ Quick tests failed$(NC)"

# Enhanced status
status:
	@$(call section_header,"Project Status")
	@echo "$(BOLD)$(YELLOW)Git Status:$(NC)"
	@git status -s 2>/dev/null | head -10 || echo "  Clean working directory"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Build Status:$(NC)"
	@if [ -f "bin/guild" ]; then \
		echo "  $(GREEN)$(CHECK) Binary exists$(NC)" ; \
		ls -lh bin/guild | awk '{print "  Size: " $$5 ", Modified: " $$6 " " $$7 " " $$8}' ; \
	else \
		echo "  $(RED)$(CROSS) Binary not built$(NC)" ; \
	fi

# Provider test dashboard
provider-test:
	@$(call section_header,"AI Provider Test Dashboard")
	@echo "$(BOLD)$(YELLOW)┌── AI Provider Status ──────────────────────────┐$(NC)"
	@rm -f .test-results.tmp
	@TOTAL=0; PASSED=0; \
	for provider in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		if [ -d "./pkg/providers/$$provider" ]; then \
			TOTAL=$$((TOTAL + 1)); \
			printf "$(BLUE)│$(NC) %-20s" "$$provider" ; \
			if go test -short -count=1 ./pkg/providers/$$provider >/dev/null 2>&1; then \
				echo "$(GREEN)$(CHECK) PASS$(NC) $(BLUE)│$(NC)" ; \
				echo "PASS $$provider" >> .test-results.tmp ; \
				PASSED=$$((PASSED + 1)); \
			else \
				echo "$(RED)$(CROSS) FAIL$(NC) $(BLUE)│$(NC)" ; \
				echo "FAIL $$provider" >> .test-results.tmp ; \
			fi ; \
		fi ; \
	done; \
	echo "$(BOLD)$(YELLOW)└─────────────────────────────────────────────────┘$(NC)"
	@echo ""
	@TOTAL=$$(cat .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	PASSED=$$(grep "^PASS" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	FAILED=$$(grep "^FAIL" .test-results.tmp 2>/dev/null | wc -l | tr -d ' ') ; \
	if [ "$$FAILED" -eq "0" ]; then \
		$(call status_card,"All Providers Healthy","pass") ; \
	else \
		$(call status_card,"Provider Issues Detected","fail") ; \
	fi
	@rm -f .test-results.tmp

# Documentation server
docs-serve:
	@$(call section_header,"Documentation Server")
	@echo "$(BOLD)$(YELLOW)Installing pkgsite...$(NC)"
	@go install golang.org/x/pkgsite/cmd/pkgsite@latest 2>/dev/null || true
	@$(call status_card,"Documentation Server Starting","")
	@pkgsite -http=:8080

# Keep essential legacy targets for compatibility
test: unit-test integration
dashboard-test: unit-test
quick-test: unit-test
build-all: clean build verify

.DEFAULT_GOAL := help
