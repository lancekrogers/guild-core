# ─────────────────────────────────────────────────────────────────────────────
# Guild Framework Makefile — Enhanced Dashboard Edition
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: \
    all help clean build verify build-all \
    unit-test test dashboard-test \
    integration integration-verbose \
    coverage lint format install-tools \
    health status quick-test quick-check check \
    provider-test docs-serve dashboard \
    proto proto-check

#────────────────────────── COLOURS ───────────────────────────────────────
ifneq ($(NO_COLOR),1)
ifneq ($(CI),true)
BLUE   := \033[38;5;74m
GREEN  := \033[38;5;76m
YELLOW := \033[38;5;220m
RED    := \033[38;5;196m
PURPLE := \033[38;5;141m
CYAN   := \033[38;5;51m
GRAY   := \033[38;5;240m
WHITE  := \033[38;5;255m
BOLD   := \033[1m
DIM    := \033[2m
NC     := \033[0m
CLEAR  := \033[2K\r
else
BLUE := ; GREEN := ; YELLOW := ; RED := ; PURPLE := ; CYAN := ; GRAY := ; WHITE := ; BOLD := ; DIM := ; NC := ; CLEAR :=
endif
else
BLUE := ; GREEN := ; YELLOW := ; RED := ; PURPLE := ; CYAN := ; GRAY := ; WHITE := ; BOLD := ; DIM := ; NC := ; CLEAR :=
endif

#────────────────────────── ICONS ─────────────────────────────────────────
CHECK := ✓
CROSS := ✗
ARROW := →
ROCKET:= 🚀
CLIP  := 📋
SHIELD:= 🛡
GEAR  := ⚙
BUILD := 🔨
TEST  := 🧪
CLEAN := 🧹

#────────────────────────── HELPERS ───────────────────────────────────────
BAR := ──────────────────────────────────────────────────────────── # 60 dashes

# Enhanced progress bar that accurately tracks percentage
define progress_bar
	PERCENT=$(1); WIDTH=40; \
	FILLED=$$(($$PERCENT * $$WIDTH / 100)); \
	EMPTY=$$(($$WIDTH - $$FILLED)); \
	printf "$(CLEAR)$(GRAY)["; \
	if [ $$FILLED -gt 0 ]; then \
		for i in $$(seq 1 $$FILLED); do printf "$(GREEN)█"; done; \
	fi; \
	if [ $$EMPTY -gt 0 ]; then \
		for i in $$(seq 1 $$EMPTY); do printf "$(GRAY)░"; done; \
	fi; \
	printf "$(GRAY)] $(BOLD)%3d%%$(NC)" $$PERCENT
endef

# Progress bar with status message
define progress_status
	$(call progress_bar,$(1)); \
	printf " $(YELLOW)$(2)$(NC)\n"
endef

# Real-time progress bar that updates in place
define live_progress_bar
	PERCENT=$(1); WIDTH=40; MESSAGE="$(2)"; \
	FILLED=$$(($$PERCENT * $$WIDTH / 100)); \
	EMPTY=$$(($$WIDTH - $$FILLED)); \
	printf "\r$(CLEAR)$(GRAY)["; \
	if [ $$FILLED -gt 0 ]; then \
		for i in $$(seq 1 $$FILLED); do printf "$(GREEN)█"; done; \
	fi; \
	if [ $$EMPTY -gt 0 ]; then \
		for i in $$(seq 1 $$EMPTY); do printf "$(GRAY)░"; done; \
	fi; \
	printf "$(GRAY)] $(BOLD)%3d%%$(NC) $(YELLOW)$$MESSAGE$(NC)" $$PERCENT; \
	[ "$$PERCENT" -eq 100 ] && echo ""
endef

# Progress tracking for test suites
define update_test_progress
	CURRENT=$(1); TOTAL=$(2); PACKAGE=$(3); \
	PERCENT=$$(($$CURRENT * 100 / $$TOTAL)); \
	MESSAGE="Testing $$PACKAGE... ($$CURRENT/$$TOTAL)"; \
	$(call live_progress_bar,$$PERCENT,$$MESSAGE)
endef

# Run test with progress updates  
define run_test_with_progress
	CURRENT=$(1); TOTAL=$(2); PACKAGE=$(3); \
	PERCENT=$$(($$CURRENT * 100 / $$TOTAL)); \
	$(call live_progress_bar,$$PERCENT,Testing $$PACKAGE...); \
	if go test -short -count=1 ./pkg/$$PACKAGE/... >/dev/null 2>&1; then \
		echo "0"; \
	else \
		echo "1"; \
	fi
endef

define section_header
	@echo ""; \
	echo "$(BOLD)$(BLUE)┌$(BAR)$(NC)"; \
	printf "$(BOLD)$(BLUE)│$(NC) $(PURPLE)🏰 GUILD$(NC) $(BOLD)$(YELLOW)%s$(NC)\n" "$(strip $(1))"; \
	echo "$(BOLD)$(BLUE)└$(BAR)$(NC)"; \
	echo ""
endef

define status_card
	echo "$(BOLD)$(BLUE)┌$(BAR)$(NC)"; \
	if [ "$(2)" = "pass" ]; then \
		printf "$(BOLD)$(BLUE)│$(NC)  $(GREEN)✓ $(1)$(NC)\n"; \
	else \
		printf "$(BOLD)$(BLUE)│$(NC)  $(RED)✗ $(1)$(NC)\n"; \
	fi; \
	echo "$(BOLD)$(BLUE)└$(BAR)$(NC)"
endef

# Build status tracking
BUILD_STEPS := 10
CURRENT_STEP := 0

#────────────────────────── DEFAULT FLOW ──────────────────────────────────
all: dashboard

dashboard: clean build unit-test integration
	@$(call section_header,Complete Build & Test Summary)
	@echo "$(BOLD)$(BLUE)┌────────────────── Final Summary Dashboard ────────────────────┐$(NC)"
	@echo "$(BLUE)│$(NC)                                                                   $(BLUE)│$(NC)"
	@echo "$(BLUE)│$(NC)   $(BOLD)🏰 GUILD FRAMEWORK COMPLETE BUILD & TEST SUMMARY$(NC)              $(BLUE)│$(NC)"
	@echo "$(BLUE)│$(NC)                                                                   $(BLUE)│$(NC)"
	@echo "$(BLUE)├───────────────────────────────────────────────────────────────────┤$(NC)"
	@echo "$(BLUE)│$(NC)   All unit tests, builds, and integration tests completed.       $(BLUE)│$(NC)"
	@echo "$(BLUE)│$(NC)   Review the detailed results above for any failures.            $(BLUE)│$(NC)"
	@echo "$(BLUE)│$(NC)                                                                   $(BLUE)│$(NC)"
	@echo "$(BOLD)$(BLUE)└───────────────────────────────────────────────────────────────────┘$(NC)"
	@$(call status_card,$(ROCKET) Dashboard Run Complete,pass)

.DEFAULT_GOAL := help

#───────────────────────────── HELP ───────────────────────────────────────
help:
	@$(call section_header,Available Commands)
	@echo "  $(BOLD)Build Commands:$(NC)"
	@echo "    make dashboard      $(ARROW) Run full build dashboard with progress"
	@echo "    make build          $(ARROW) Build CLI with visual progress"
	@echo "    make verify         $(ARROW) Strict build (fails on error)"
	@echo "    make clean          $(ARROW) Clean build artifacts"
	@echo ""
	@echo "  $(BOLD)Test Commands:$(NC)"
	@echo "    make unit-test      $(ARROW) Run unit tests with dashboard"
	@echo "    make integration    $(ARROW) Run integration tests"
	@echo "    make coverage       $(ARROW) Generate coverage report"
	@echo ""
	@echo "  $(BOLD)Quality Commands:$(NC)"
	@echo "    make lint           $(ARROW) Run linters"
	@echo "    make health         $(ARROW) Health check dashboard"
	@echo ""
	@echo "  $(BOLD)Code Generation:$(NC)"
	@echo "    make proto          $(ARROW) Generate Go code from proto files"
	@echo "    make proto-check    $(ARROW) Verify proto file validity"

#──────────────────────────── CLEAN ───────────────────────────────────────
clean:
	@$(call section_header,$(CLEAN) Cleaning Build Environment)
	@$(call live_progress_bar,0,Starting cleanup...); \
	rm -rf bin 2>/dev/null || true; \
	$(call live_progress_bar,25,Removed binaries); \
	rm -rf coverage 2>/dev/null || true; \
	$(call live_progress_bar,50,Removed coverage data); \
	rm -f guild 2>/dev/null || true; \
	$(call live_progress_bar,75,Removed stray files); \
	go clean -testcache -cache 2>/dev/null || true; \
	$(call live_progress_bar,100,Cleanup complete); \
	echo ""; \
	$(call status_card,Environment Cleaned,pass)

#──────────────────────────── BUILD ───────────────────────────────────────
build:
	@$(call section_header,$(BUILD) Building Guild CLI)
	@mkdir -p bin
	@VET_STATUS=pending; BUILD_STATUS=pending; STRIP_STATUS=pending; \
	ERROR_COUNT=0; \
	$(call live_progress_bar,0,Initializing build process...); \
	sleep 0.2; \
	$(call live_progress_bar,10,Running dependency analysis...); \
	go mod download 2>/dev/null || true; \
	$(call live_progress_bar,20,Checking code quality with go vet...); \
	if go vet ./... >vet_errors.txt 2>&1; then \
		VET_STATUS=pass; \
		$(call live_progress_bar,40,✓ Code quality check passed); \
		rm -f vet_errors.txt; \
	else \
		VET_STATUS=fail; \
		ERROR_COUNT=$$((ERROR_COUNT+1)); \
		$(call live_progress_bar,40,✗ Code quality issues detected); \
	fi; \
	$(call live_progress_bar,50,Compiling Guild binary...); \
	if go build -o bin/guild ./cmd/guild >/dev/null 2>&1; then \
		BUILD_STATUS=pass; \
		$(call live_progress_bar,80,✓ Compilation successful); \
	else \
		BUILD_STATUS=fail; \
		ERROR_COUNT=$$((ERROR_COUNT+1)); \
		$(call live_progress_bar,80,✗ Compilation failed); \
		echo ""; \
		echo "$(RED)Build Error Details:$(NC)"; \
		go build -o bin/guild ./cmd/guild 2>&1 | head -10; \
	fi; \
	if [ "$$BUILD_STATUS" = "pass" ]; then \
		$(call live_progress_bar,90,Optimizing binary...); \
		if command -v strip >/dev/null 2>&1 && strip -s bin/guild 2>/dev/null; then \
			STRIP_STATUS=pass; \
		else \
			STRIP_STATUS=skip; \
		fi; \
	else \
		STRIP_STATUS=skip; \
	fi; \
	$(call live_progress_bar,100,Build process complete); \
	echo ""; \
	echo ""; \
	echo "$(BOLD)$(BLUE)┌─────────────── Build Summary ────────────────┐$(NC)"; \
	printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): " "Code Quality"; \
	if [ "$$VET_STATUS" = "pass" ]; then \
		echo "$(GREEN)✓ PASSED$(NC)                  $(BLUE)│$(NC)"; \
	else \
		echo "$(RED)✗ FAILED$(NC)                  $(BLUE)│$(NC)"; \
	fi; \
	printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): " "Compilation"; \
	if [ "$$BUILD_STATUS" = "pass" ]; then \
		echo "$(GREEN)✓ PASSED$(NC)                  $(BLUE)│$(NC)"; \
	else \
		echo "$(RED)✗ FAILED$(NC)                  $(BLUE)│$(NC)"; \
	fi; \
	printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): " "Optimization"; \
	if [ "$$STRIP_STATUS" = "pass" ]; then \
		echo "$(GREEN)✓ COMPLETED$(NC)               $(BLUE)│$(NC)"; \
	elif [ "$$STRIP_STATUS" = "skip" ]; then \
		echo "$(YELLOW)○ SKIPPED$(NC)                 $(BLUE)│$(NC)"; \
	else \
		echo "$(RED)✗ FAILED$(NC)                  $(BLUE)│$(NC)"; \
	fi; \
	echo "$(BLUE)├──────────────────────────────────────────────┤$(NC)"; \
	printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): " "Total Errors"; \
	if [ $$ERROR_COUNT -eq 0 ]; then \
		echo "$(GREEN)0$(NC)                         $(BLUE)│$(NC)"; \
	else \
		echo "$(RED)$$ERROR_COUNT$(NC)                         $(BLUE)│$(NC)"; \
	fi; \
	echo "$(BOLD)$(BLUE)└──────────────────────────────────────────────┘$(NC)"; \
	if [ $$ERROR_COUNT -eq 0 ]; then \
		echo "$(BOLD)$(BLUE)┌$(BAR)$(NC)"; \
		printf "$(BOLD)$(BLUE)│$(NC)  $(GREEN)✓ Build Completed Successfully$(NC)\n"; \
		echo "$(BOLD)$(BLUE)└$(BAR)$(NC)"; \
	else \
		echo "$(BOLD)$(BLUE)┌$(BAR)$(NC)"; \
		printf "$(BOLD)$(BLUE)│$(NC)  $(RED)✗ Build Completed with Errors$(NC)\n"; \
		echo "$(BOLD)$(BLUE)└$(BAR)$(NC)"; \
		if [ -f vet_errors.txt ] && [ "$$VET_STATUS" = "fail" ]; then \
			echo ""; \
			echo "$(BOLD)$(RED)Code Quality Issues:$(NC)"; \
			head -10 vet_errors.txt; \
			echo "$(DIM)... (showing first 10 lines)$(NC)"; \
			rm -f vet_errors.txt; \
		fi; \
	fi

#────────────────────── STRICT BUILD FOR CI ──────────────────────────────
verify:
	@$(MAKE) clean
	@go vet ./...
	@go build -o bin/guild ./cmd/guild
	@command -v strip >/dev/null 2>&1 && strip -s bin/guild
	@$(call status_card,Build Verified,pass)

#──────────────────────── UNIT-TEST DASHBOARD ────────────────────────────
unit-test:
	@$(call section_header,$(TEST) Unit Test Dashboard)
	@rm -f .unit_fail .build_fail .unit_pass .build_pass
	@echo ""; \
	echo "$(BOLD)$(PURPLE)┌────── Discovering and Testing All Packages ──────┐$(NC)"; \
	echo "$(PURPLE)│$(NC) Scanning for all Go packages...                  $(PURPLE)│$(NC)"; \
	echo "$(BOLD)$(PURPLE)└──────────────────────────────────────────────────┘$(NC)"; \
	echo ""; \
	TOTAL_PACKAGES=0; \
	BUILD_PASSED=0; \
	BUILD_FAILED=0; \
	TEST_PASSED=0; \
	TEST_FAILED=0; \
	CURRENT=0; \
	for pkg in $$(go list ./... 2>/dev/null | grep -v /vendor/); do \
		TOTAL_PACKAGES=$$((TOTAL_PACKAGES + 1)); \
	done; \
	echo "$(BOLD)Found $$TOTAL_PACKAGES packages to test$(NC)"; \
	echo ""; \
	for pkg in $$(go list ./... 2>/dev/null | grep -v /vendor/); do \
		CURRENT=$$((CURRENT + 1)); \
		PERCENT=$$((CURRENT * 100 / TOTAL_PACKAGES)); \
		PKG_SHORT=$$(echo $$pkg | sed 's|github.com/guild-ventures/guild-core/||'); \
		$(call live_progress_bar,$$PERCENT,Processing $$PKG_SHORT...); \
		if go build -o /tmp/guild-build-test-$$$$ $$pkg >/dev/null 2>&1; then \
			BUILD_PASSED=$$((BUILD_PASSED + 1)); \
			echo "$$PKG_SHORT" >> .build_pass; \
			if go test -short -count=1 $$pkg >/dev/null 2>&1; then \
				TEST_PASSED=$$((TEST_PASSED + 1)); \
				echo "$$PKG_SHORT" >> .unit_pass; \
			else \
				TEST_FAILED=$$((TEST_FAILED + 1)); \
				echo "$$PKG_SHORT" >> .unit_fail; \
			fi; \
		else \
			BUILD_FAILED=$$((BUILD_FAILED + 1)); \
			echo "$$PKG_SHORT" >> .build_fail; \
		fi; \
	done; \
	rm -f /tmp/guild-build-test-*; \
	$(call live_progress_bar,100,All packages tested); \
	echo ""; \
	echo ""; \
	echo "$(BOLD)$(BLUE)┌────────────────── Test Results Summary ──────────────────┐$(NC)"; \
	echo "$(BLUE)├───────────────────────────────────────────────────────────┤$(NC)"; \
	echo "$(BLUE)│$(NC) $(BOLD)Summary Statistics$(NC)                                       $(BLUE)│$(NC)"; \
	printf "$(BLUE)│$(NC)   %-25s : %-28d$(BLUE)│$(NC)\n" "Total Packages" $$TOTAL_PACKAGES; \
	printf "$(BLUE)│$(NC)   %-25s : $(GREEN)%-28d$(NC)$(BLUE)│$(NC)\n" "Build Passed" $$BUILD_PASSED; \
	printf "$(BLUE)│$(NC)   %-25s : " "Build Failed"; \
	if [ $$BUILD_FAILED -eq 0 ]; then \
		printf "$(GREEN)%-28d$(NC)$(BLUE)│$(NC)\n" $$BUILD_FAILED; \
	else \
		printf "$(RED)%-28d$(NC)$(BLUE)│$(NC)\n" $$BUILD_FAILED; \
	fi; \
	printf "$(BLUE)│$(NC)   %-25s : $(GREEN)%-28d$(NC)$(BLUE)│$(NC)\n" "Tests Passed" $$TEST_PASSED; \
	printf "$(BLUE)│$(NC)   %-25s : " "Tests Failed"; \
	if [ $$TEST_FAILED -eq 0 ]; then \
		printf "$(GREEN)%-28d$(NC)$(BLUE)│$(NC)\n" $$TEST_FAILED; \
	else \
		printf "$(RED)%-28d$(NC)$(BLUE)│$(NC)\n" $$TEST_FAILED; \
	fi; \
	echo "$(BLUE)├───────────────────────────────────────────────────────────┤$(NC)"; \
	if [ $$TOTAL_PACKAGES -gt 0 ]; then \
		BUILD_RATE=$$((BUILD_PASSED * 100 / TOTAL_PACKAGES)); \
		printf "$(BLUE)│$(NC)   %-25s : " "Build Success Rate"; \
		if [ $$BUILD_RATE -eq 100 ]; then \
			printf "$(GREEN)%d%%$(NC)" $$BUILD_RATE; \
		elif [ $$BUILD_RATE -ge 95 ]; then \
			printf "$(YELLOW)%d%%$(NC)" $$BUILD_RATE; \
		else \
			printf "$(RED)%d%%$(NC)" $$BUILD_RATE; \
		fi; \
		printf "%*s$(BLUE)│$(NC)\n" $$((26 - $${#BUILD_RATE})) ""; \
		if [ $$((TOTAL_PACKAGES - BUILD_FAILED)) -gt 0 ]; then \
			TEST_RATE=$$((TEST_PASSED * 100 / (TOTAL_PACKAGES - BUILD_FAILED))); \
			printf "$(BLUE)│$(NC)   %-25s : " "Test Success Rate"; \
			if [ $$TEST_RATE -eq 100 ]; then \
				printf "$(GREEN)%d%%$(NC)" $$TEST_RATE; \
			elif [ $$TEST_RATE -ge 90 ]; then \
				printf "$(YELLOW)%d%%$(NC)" $$TEST_RATE; \
			else \
				printf "$(RED)%d%%$(NC)" $$TEST_RATE; \
			fi; \
			printf "%*s$(BLUE)│$(NC)\n" $$((26 - $${#TEST_RATE})) ""; \
		fi; \
	fi; \
	echo "$(BOLD)$(BLUE)└───────────────────────────────────────────────────────────┘$(NC)"; \
	if [ $$TEST_FAILED -eq 0 ] && [ $$BUILD_FAILED -eq 0 ]; then \
		echo ""; \
		$(call status_card,✓ All Tests and Builds Passed,pass); \
	else \
		echo ""; \
		$(call status_card,✗ Some Tests or Builds Failed,fail); \
		if [ -f .build_fail ] && [ $$BUILD_FAILED -gt 0 ]; then \
			echo ""; \
			echo "$(BOLD)$(RED)Build Failures ($$BUILD_FAILED):$(NC)"; \
			cat .build_fail | sort | while read pkg; do echo "  - $$pkg"; done; \
		fi; \
		if [ -f .unit_fail ] && [ $$TEST_FAILED -gt 0 ]; then \
			echo ""; \
			echo "$(BOLD)$(RED)Test Failures ($$TEST_FAILED):$(NC)"; \
			cat .unit_fail | sort | while read pkg; do echo "  - $$pkg"; done; \
		fi; \
	fi
	@rm -f .unit_fail .build_fail .unit_pass .build_pass
	@rm -f guild_hall rag_test prompt_link_parser guild

#──────────────────────── INTEGRATION DASHBOARD ──────────────────────────
integration:
	@$(call section_header,$(TEST) Integration Test Dashboard)
	@rm -f .integration_fail .integration_pass
	@echo ""; \
	echo "$(BOLD)$(CYAN)┌─────── Discovering Integration Test Suites ───────┐$(NC)"; \
	TOTAL=0; PASSED=0; FAILED=0; CURRENT=0; \
	for D in $$(find ./integration -type d -mindepth 1 -maxdepth 1 2>/dev/null | sort); do \
		TOTAL=$$((TOTAL+1)); \
	done; \
	echo "$(CYAN)│$(NC) Found $$TOTAL integration test suites              $(CYAN)│$(NC)"; \
	echo "$(BOLD)$(CYAN)└───────────────────────────────────────────────────┘$(NC)"; \
	echo ""; \
	for D in $$(find ./integration -type d -mindepth 1 -maxdepth 1 2>/dev/null | sort); do \
		SUITE=$$(basename $$D); \
		CURRENT=$$((CURRENT+1)); \
		PERCENT=$$((CURRENT * 100 / TOTAL)); \
		$(call live_progress_bar,$$PERCENT,Testing $$SUITE...); \
		if go test -v -tags=integration $$D >/dev/null 2>&1; then \
			PASSED=$$((PASSED+1)); \
			echo "$$SUITE" >> .integration_pass; \
		else \
			FAILED=$$((FAILED+1)); \
			echo "$$SUITE" >> .integration_fail; \
		fi; \
	done; \
	$(call live_progress_bar,100,Integration tests complete); \
	echo ""; \
	echo ""; \
	echo "$(BOLD)$(BLUE)┌─────────────── Integration Test Results ──────────────┐$(NC)"; \
	echo "$(BLUE)│$(NC) $(BOLD)Suite                    Status$(NC)                      $(BLUE)│$(NC)"; \
	echo "$(BLUE)├────────────────────────────────────────────────────────┤$(NC)"; \
	for D in $$(find ./integration -type d -mindepth 1 -maxdepth 1 2>/dev/null | sort); do \
		SUITE=$$(basename $$D); \
		printf "$(BLUE)│$(NC)   %-20s " "$$SUITE"; \
		if grep -q "^$$SUITE$$" .integration_fail 2>/dev/null; then \
			printf "$(RED)✗ FAIL$(NC)                     "; \
		else \
			printf "$(GREEN)✓ PASS$(NC)                     "; \
		fi; \
		echo "$(BLUE)│$(NC)"; \
	done; \
	echo "$(BLUE)├────────────────────────────────────────────────────────┤$(NC)"; \
	printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): %-37d $(BLUE)│$(NC)\n" "Total Suites" $$TOTAL; \
	printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): $(GREEN)%-37d$(NC) $(BLUE)│$(NC)\n" "Passed" $$PASSED; \
	printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): " "Failed"; \
	if [ $$FAILED -eq 0 ]; then \
		printf "$(GREEN)%-37d$(NC) $(BLUE)│$(NC)\n" $$FAILED; \
	else \
		printf "$(RED)%-37d$(NC) $(BLUE)│$(NC)\n" $$FAILED; \
	fi; \
	echo "$(BLUE)├────────────────────────────────────────────────────────┤$(NC)"; \
	if [ $$TOTAL -gt 0 ]; then \
		SUCCESS_RATE=$$((PASSED * 100 / TOTAL)); \
		printf "$(BLUE)│$(NC) $(BOLD)%-15s$(NC): " "Success Rate"; \
		if [ $$SUCCESS_RATE -eq 100 ]; then \
			printf "$(GREEN)%d%%$(NC)" $$SUCCESS_RATE; \
		elif [ $$SUCCESS_RATE -ge 80 ]; then \
			printf "$(YELLOW)%d%%$(NC)" $$SUCCESS_RATE; \
		else \
			printf "$(RED)%d%%$(NC)" $$SUCCESS_RATE; \
		fi; \
		printf "%*s$(BLUE)│$(NC)\n" $$((34 - $${#SUCCESS_RATE})) ""; \
	fi; \
	echo "$(BOLD)$(BLUE)└────────────────────────────────────────────────────────┘$(NC)"; \
	if [ $$FAILED -eq 0 ]; then \
		echo ""; \
		$(call status_card,✓ All Integration Tests Passed,pass); \
	else \
		echo ""; \
		$(call status_card,✗ Some Integration Tests Failed,fail); \
		if [ -f .integration_fail ]; then \
			echo ""; \
			echo "$(BOLD)$(RED)Failed Integration Suites:$(NC)"; \
			cat .integration_fail | while read suite; do echo "  - $$suite"; done; \
		fi; \
	fi
	@rm -f .integration_fail .integration_pass
	@rm -f guild_hall rag_test prompt_link_parser guild

integration-verbose:
	@$(call section_header,Integration – Verbose)
	@go test -v ./integration/...

#──────────────────────────── HEALTH DASHBOARD ───────────────────────────
health:
	@$(call section_header,$(SHIELD) System Health Check)
	@CHECKS_TOTAL=5; CHECKS_PASSED=0; \
	$(call live_progress_bar,0,Starting health checks...); \
	sleep 0.3; \
	echo ""; \
	echo "$(BOLD)$(PURPLE)┌──────── System Health Status ────────┐$(NC)"; \
	$(call live_progress_bar,20,Checking build system...); \
	printf "\n$(PURPLE)│$(NC) %-20s: " "Build System"; \
	if $(MAKE) -s build >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Healthy$(NC)     $(PURPLE)│$(NC)"; \
		CHECKS_PASSED=$$((CHECKS_PASSED+1)); \
	else \
		echo "$(RED)✗ Unhealthy$(NC)   $(PURPLE)│$(NC)"; \
	fi; \
	$(call live_progress_bar,40,Checking binary presence...); \
	printf "\n$(PURPLE)│$(NC) %-20s: " "Binary Exists"; \
	if [ -f bin/guild ]; then \
		echo "$(GREEN)✓ Present$(NC)     $(PURPLE)│$(NC)"; \
		CHECKS_PASSED=$$((CHECKS_PASSED+1)); \
	else \
		echo "$(RED)✗ Missing$(NC)     $(PURPLE)│$(NC)"; \
	fi; \
	$(call live_progress_bar,60,Checking binary permissions...); \
	printf "\n$(PURPLE)│$(NC) %-20s: " "Binary Executable"; \
	if [ -x bin/guild ]; then \
		echo "$(GREEN)✓ Yes$(NC)         $(PURPLE)│$(NC)"; \
		CHECKS_PASSED=$$((CHECKS_PASSED+1)); \
	else \
		echo "$(RED)✗ No$(NC)          $(PURPLE)│$(NC)"; \
	fi; \
	$(call live_progress_bar,80,Checking binary functionality...); \
	printf "\n$(PURPLE)│$(NC) %-20s: " "Binary Runs"; \
	if [ -x bin/guild ] && bin/guild --help >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Working$(NC)     $(PURPLE)│$(NC)"; \
		CHECKS_PASSED=$$((CHECKS_PASSED+1)); \
	else \
		echo "$(RED)✗ Broken$(NC)      $(PURPLE)│$(NC)"; \
	fi; \
	$(call live_progress_bar,90,Checking go modules...); \
	printf "\n$(PURPLE)│$(NC) %-20s: " "Go Modules"; \
	if go list -m all >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Valid$(NC)       $(PURPLE)│$(NC)"; \
		CHECKS_PASSED=$$((CHECKS_PASSED+1)); \
	else \
		echo "$(RED)✗ Invalid$(NC)     $(PURPLE)│$(NC)"; \
	fi; \
	echo "$(PURPLE)├──────────────────────────────────────┤$(NC)"; \
	printf "$(PURPLE)│$(NC) $(BOLD)Health Score$(NC): "; \
	HEALTH_PERCENT=$$((CHECKS_PASSED * 100 / CHECKS_TOTAL)); \
	if [ $$HEALTH_PERCENT -eq 100 ]; then \
		printf "$(GREEN)$$HEALTH_PERCENT%% ($$CHECKS_PASSED/$$CHECKS_TOTAL)$(NC)"; \
	elif [ $$HEALTH_PERCENT -ge 60 ]; then \
		printf "$(YELLOW)$$HEALTH_PERCENT%% ($$CHECKS_PASSED/$$CHECKS_TOTAL)$(NC)"; \
	else \
		printf "$(RED)$$HEALTH_PERCENT%% ($$CHECKS_PASSED/$$CHECKS_TOTAL)$(NC)"; \
	fi; \
	printf "%$$((22 - $${#HEALTH_PERCENT} - $${#CHECKS_PASSED} - $${#CHECKS_TOTAL} - 4))s$(PURPLE)│$(NC)\n" ""; \
	echo "$(BOLD)$(PURPLE)└──────────────────────────────────────┘$(NC)"; \
	$(call live_progress_bar,100,Health check complete); \
	echo ""; \
	if [ $$CHECKS_PASSED -eq $$CHECKS_TOTAL ]; then \
		$(call status_card,System is Healthy,pass); \
	else \
		$(call status_card,System Health Issues Detected,fail); \
	fi

#──────────────────────────── COVERAGE REPORT ────────────────────────────
coverage:
	@$(call section_header,$(CLIP) Coverage Analysis)
	@mkdir -p coverage
	@$(call live_progress_bar,0,Initializing coverage analysis...); \
	sleep 0.3; \
	$(call live_progress_bar,30,Running tests with coverage...); \
	if go test -coverprofile=coverage/coverage.out -covermode=atomic ./... >/dev/null 2>&1; then \
		$(call live_progress_bar,70,Analyzing coverage data...); \
		COVERAGE=$$(go tool cover -func=coverage/coverage.out | tail -n1 | awk '{print $$3}'); \
		$(call live_progress_bar,90,Generating HTML report...); \
		go tool cover -html=coverage/coverage.out -o coverage/coverage.html 2>/dev/null; \
		$(call live_progress_bar,100,Coverage analysis complete); \
		echo ""; \
		echo "$(BOLD)$(BLUE)┌──────── Coverage Summary ────────┐$(NC)"; \
		printf "$(BLUE)│$(NC) $(BOLD)Total Coverage$(NC): "; \
		COVERAGE_NUM=$$(echo $$COVERAGE | tr -d '%'); \
		if [ $$(echo "$$COVERAGE_NUM >= 80" | bc -l) -eq 1 ]; then \
			printf "$(GREEN)$$COVERAGE$(NC)"; \
		elif [ $$(echo "$$COVERAGE_NUM >= 60" | bc -l) -eq 1 ]; then \
			printf "$(YELLOW)$$COVERAGE$(NC)"; \
		else \
			printf "$(RED)$$COVERAGE$(NC)"; \
		fi; \
		printf "%$$((19 - $${#COVERAGE}))s$(BLUE)│$(NC)\n" ""; \
		echo "$(BLUE)│$(NC) Report: coverage/coverage.html   $(BLUE)│$(NC)"; \
		echo "$(BOLD)$(BLUE)└──────────────────────────────────┘$(NC)"; \
		$(call status_card,Coverage Report Generated,pass); \
	else \
		$(call live_progress_bar,100,Coverage analysis failed); \
		$(call status_card,Coverage Analysis Failed,fail); \
	fi

#──────────────────────────── LINT / FORMAT ───────────────────────────────
install-tools:
	@$(call section_header,$(GEAR) Installing Development Tools)
	@$(call live_progress_bar,0,Starting tool installation...); \
	sleep 0.3; \
	$(call live_progress_bar,50,Installing golangci-lint...); \
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null; \
	$(call live_progress_bar,100,Installing gotestsum...); \
	go install gotest.tools/gotestsum@latest 2>/dev/null; \
	echo ""; \
	$(call status_card,Development Tools Installed,pass)

lint:
	@$(call section_header,$(SHIELD) Code Quality Check)
	@ISSUES_FOUND=0; \
	$(call live_progress_bar,0,Starting code quality checks...); \
	sleep 0.3; \
	echo ""; \
	echo "$(BOLD)$(YELLOW)┌──────── Code Quality Report ────────┐$(NC)"; \
	$(call live_progress_bar,30,Running go vet...); \
	printf "\n$(YELLOW)│$(NC) %-18s: " "Go Vet"; \
	if go vet ./... >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Clean$(NC)       $(YELLOW)│$(NC)"; \
	else \
		echo "$(RED)✗ Issues$(NC)      $(YELLOW)│$(NC)"; \
		ISSUES_FOUND=$$((ISSUES_FOUND+1)); \
	fi; \
	$(call live_progress_bar,60,Checking formatting...); \
	printf "\n$(YELLOW)│$(NC) %-18s: " "Go Format"; \
	UNFORMATTED=$$(gofmt -l . 2>/dev/null | wc -l); \
	if [ $$UNFORMATTED -eq 0 ]; then \
		echo "$(GREEN)✓ Clean$(NC)       $(YELLOW)│$(NC)"; \
	else \
		echo "$(RED)✗ $$UNFORMATTED files$(NC)    $(YELLOW)│$(NC)"; \
		ISSUES_FOUND=$$((ISSUES_FOUND+1)); \
	fi; \
	$(call live_progress_bar,90,Running golangci-lint...); \
	printf "\n$(YELLOW)│$(NC) %-18s: " "Golangci-lint"; \
	if command -v golangci-lint >/dev/null 2>&1; then \
		if golangci-lint run ./... >/dev/null 2>&1; then \
			echo "$(GREEN)✓ Clean$(NC)       $(YELLOW)│$(NC)"; \
		else \
			echo "$(RED)✗ Issues$(NC)      $(YELLOW)│$(NC)"; \
			ISSUES_FOUND=$$((ISSUES_FOUND+1)); \
		fi; \
	else \
		echo "$(GRAY)○ Not installed$(YELLOW)│$(NC)"; \
	fi; \
	echo "$(BOLD)$(YELLOW)└─────────────────────────────────────┘$(NC)"; \
	$(call live_progress_bar,100,Quality check complete); \
	echo ""; \
	if [ $$ISSUES_FOUND -eq 0 ]; then \
		$(call status_card,Code Quality: Excellent,pass); \
	else \
		$(call status_card,Code Quality: $$ISSUES_FOUND Issues Found,fail); \
	fi

format:
	@$(call section_header,$(GEAR) Code Formatting)
	@$(call live_progress_bar,0,Starting code formatting...); \
	sleep 0.3; \
	$(call live_progress_bar,50,Running go fmt...); \
	go fmt ./...; \
	$(call live_progress_bar,100,Formatting complete); \
	echo ""; \
	$(call status_card,Code Formatted Successfully,pass)

#──────────────────────── QUICK-CHECK & ALIASES ───────────────────────────
quick-check: build
	@RESULT=$$(go test -short -count=1 ./pkg/... >/dev/null 2>&1 && echo "pass" || echo "fail"); \
	if [ "$$RESULT" = "pass" ]; then \
		$(call status_card,Quick Check Passed,pass); \
	else \
		$(call status_card,Quick Check Failed,fail); \
	fi

check: quick-check

test: unit-test integration
dashboard-test: unit-test
quick-test: unit-test
build-all: clean build verify

#──────────────────────── PROVIDER DASHBOARD ──────────────────────────────
provider-test:
	@$(call section_header,$(TEST) Provider Test Suite)
	@TOTAL=0; PASSED=0; \
	for P in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		[ -d ./pkg/providers/$$P ] && TOTAL=$$((TOTAL+1)); \
	done; \
	echo ""; \
	echo "$(BOLD)$(GREEN)┌──────── Provider Tests ────────┐$(NC)"; \
	for P in mock anthropic deepseek deepinfra ollama ora openai claudecode; do \
		[ -d ./pkg/providers/$$P ] || continue; \
		printf "$(GREEN)│$(NC) %-12s: " "$$P"; \
		if go test -short -count=1 ./pkg/providers/$$P >/dev/null 2>&1; then \
			echo "$(GREEN)✓ PASS$(NC)       $(GREEN)│$(NC)"; \
			PASSED=$$((PASSED+1)); \
		else \
			echo "$(RED)✗ FAIL$(NC)       $(GREEN)│$(NC)"; \
		fi; \
	done; \
	echo "$(BOLD)$(GREEN)└────────────────────────────────┘$(NC)"; \
	echo ""; \
	if [ $$PASSED -eq $$TOTAL ]; then \
		$(call status_card,All Providers Tested Successfully,pass); \
	else \
		$(call status_card,Provider Tests: $$PASSED/$$TOTAL Passed,fail); \
	fi

#──────────────────────── DOCUMENTATION SERVER ────────────────────────────
docs-serve:
	@$(call section_header,📚 Documentation Server)
	@$(call progress_status,0,Installing pkgsite...); \
	go install golang.org/x/pkgsite/cmd/pkgsite@latest 2>/dev/null; \
	$(call progress_status,100,Starting documentation server...); \
	echo ""; \
	echo "$(BOLD)$(CYAN)Documentation server starting on http://localhost:8080$(NC)"; \
	echo "$(DIM)Press Ctrl+C to stop the server$(NC)"; \
	pkgsite -http=:8080

#──────────────────────── STATUS ─────────────────────────────────────────―
status:
	@$(call section_header,📊 Project Status)
	@echo "$(BOLD)Git Status:$(NC)"; \
	git status -s | head -10 || echo "  $(DIM)No changes$(NC)"; \
	echo ""; \
	echo "$(BOLD)Build Status:$(NC)"; \
	if [ -f bin/guild ]; then \
		echo "  $(GREEN)✓ Binary exists$(NC)"; \
		SIZE=$$(ls -lh bin/guild | awk '{print $$5}'); \
		echo "  $(DIM)Size: $$SIZE$(NC)"; \
	else \
		echo "  $(RED)✗ No binary found$(NC)"; \
	fi; \
	echo ""; \
	echo "$(BOLD)Module Status:$(NC)"; \
	if go list -m >/dev/null 2>&1; then \
		MODULE=$$(go list -m); \
		echo "  $(GREEN)✓ Module: $$MODULE$(NC)"; \
	else \
		echo "  $(RED)✗ Not a Go module$(NC)"; \
	fi

#──────────────────────────── PROTO GENERATION ────────────────────────────
proto:
	@$(call section_header,$(GEAR) Protocol Buffer Code Generation)
	@if [ ! -f scripts/generate-proto.sh ]; then \
		echo "$(RED)Error: Proto generation script not found$(NC)"; \
		echo "Expected at: scripts/generate-proto.sh"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Generating Go code from Protocol Buffer definitions...$(NC)"
	@./scripts/generate-proto.sh
	@$(call status_card,Proto Generation Complete,pass)

proto-check:
	@$(call section_header,$(SHIELD) Proto File Verification)
	@echo "$(YELLOW)Checking proto file consistency...$(NC)"
	@PROTO_FILES=$$(find proto -name "*.proto"); \
	ISSUES=0; \
	for proto in $$PROTO_FILES; do \
		echo "Checking: $$proto"; \
		if ! protoc --proto_path=. $$proto --go_out=/tmp 2>/dev/null; then \
			echo "$(RED)✗ Invalid proto file: $$proto$(NC)"; \
			ISSUES=$$((ISSUES+1)); \
		else \
			echo "$(GREEN)✓ Valid$(NC)"; \
		fi; \
	done; \
	if [ $$ISSUES -eq 0 ]; then \
		$(call status_card,All Proto Files Valid,pass); \
	else \
		$(call status_card,$$ISSUES Proto Files Have Issues,fail); \
		exit 1; \
	fi

#────────────────────────── ALIASES & SHORTCUTS ──────────────────────────
.PHONY: d b t h c l
d: dashboard    # Quick dashboard
b: build       # Quick build
t: test        # Quick test
h: health      # Quick health
c: coverage    # Quick coverage
l: lint        # Quick lint

#──────────────────────────────────────────────────────────────────────────
