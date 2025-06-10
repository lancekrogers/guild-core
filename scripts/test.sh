#!/bin/bash
# Test runner script that prevents test binaries in root directory

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo "Running Guild Framework tests..."

# Create a temporary directory for test binaries
TEST_BIN_DIR=".test-binaries"
mkdir -p "$TEST_BIN_DIR"

# Clean up function
cleanup() {
    echo "Cleaning up test binaries..."
    rm -rf "$TEST_BIN_DIR"
}

# Set up trap to clean up on exit
trap cleanup EXIT

# Run tests with explicit output directory for binaries
# The -c flag prevents caching, ensuring fresh test runs
echo "Running unit tests..."
go test -c -o "$TEST_BIN_DIR/" ./... 2>/dev/null || true
go test ./... -v

echo -e "${GREEN}Tests completed successfully!${NC}"