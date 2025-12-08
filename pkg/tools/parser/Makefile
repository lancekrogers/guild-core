# Makefile for Guild Tool Parser

.PHONY: all test bench fuzz coverage lint docs clean help

# Default target
all: test lint

# Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -timeout 30s ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem -run=^$$ ./...

# Run fuzz tests (requires Go 1.18+)
fuzz:
	@echo "Running fuzz tests..."
	@go test -fuzz=FuzzParser_ExtractToolCalls -fuzztime=30s .
	@go test -fuzz=FuzzParser_DetectFormat -fuzztime=30s .
	@go test -fuzz=FuzzJSONParser_Parse -fuzztime=30s .
	@go test -fuzz=FuzzXMLParser_Parse -fuzztime=30s .

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install from https://golangci-lint.run/"; \
		exit 1; \
	fi

# Generate documentation
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060 &
	@echo "Documentation server running at http://localhost:6060/pkg/github.com/guild-framework/guild-core/pkg/tools/parser/"

# Run examples
examples:
	@echo "Running examples..."
	@go test -run Example ./...

# Performance profiling
profile:
	@echo "Running CPU profile..."
	@go test -cpuprofile cpu.prof -bench=. -benchtime=10s ./...
	@echo "Running memory profile..."
	@go test -memprofile mem.prof -bench=. -benchtime=10s ./...
	@echo "View profiles with: go tool pprof cpu.prof or go tool pprof mem.prof"

# Check for race conditions
race:
	@echo "Checking for race conditions..."
	@go test -race ./...

# Update dependencies
deps:
	@echo "Updating dependencies..."
	@go mod tidy
	@go mod verify

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f coverage.out coverage.html
	@rm -f cpu.prof mem.prof
	@rm -f *.test
	@rm -rf fuzz/

# Run quick checks before commit
pre-commit: test lint
	@echo "Pre-commit checks passed!"

# Run integration tests only
integration:
	@echo "Running integration tests..."
	@go test -v -run Integration ./...

# Run property tests only
property:
	@echo "Running property tests..."
	@go test -v -run Property ./...

# Show metrics
metrics:
	@echo "Parser metrics:"
	@echo "  Lines of code: $$(find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1 | awk '{print $$1}')"
	@echo "  Test coverage: $$(go test -cover ./... 2>&1 | grep -E '^ok' | awk '{print $$5}' | sed 's/coverage://' | awk '{sum+=$$1; count++} END {if (count>0) printf "%.1f%%\n", sum/count; else print "N/A"}')"
	@echo "  Number of tests: $$(go test -list . ./... 2>/dev/null | grep -E '^Test' | wc -l)"
	@echo "  Number of benchmarks: $$(go test -list . ./... 2>/dev/null | grep -E '^Benchmark' | wc -l)"

# Help target
help:
	@echo "Guild Tool Parser - Available targets:"
	@echo "  make test        - Run all tests"
	@echo "  make coverage    - Generate coverage report"
	@echo "  make bench       - Run benchmarks"
	@echo "  make fuzz        - Run fuzz tests"
	@echo "  make lint        - Run linter"
	@echo "  make docs        - Start documentation server"
	@echo "  make examples    - Run example code"
	@echo "  make profile     - Generate CPU and memory profiles"
	@echo "  make race        - Check for race conditions"
	@echo "  make deps        - Update dependencies"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make pre-commit  - Run checks before committing"
	@echo "  make metrics     - Show code metrics"
	@echo "  make help        - Show this help message"