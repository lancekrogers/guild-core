#!/bin/bash
# Docker-based testing script for Guild Framework

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test modes
MODE=${1:-all}
VERBOSE=${VERBOSE:-false}

echo -e "${GREEN}🐳 Guild Docker Test Runner${NC}"
echo "================================"

# Build the test image
build_image() {
    echo -e "${YELLOW}Building test image...${NC}"
    docker build -f Dockerfile.test -t guild-test:latest .
}

# Run unit tests
run_unit_tests() {
    echo -e "${YELLOW}Running unit tests in container...${NC}"
    docker run --rm \
        -v "$(pwd)/test-results:/test-results" \
        -e TEST_OUTPUT=/test-results \
        guild-test:latest \
        make test
}

# Run integration tests with daemon
run_integration_tests() {
    echo -e "${YELLOW}Running integration tests...${NC}"
    docker-compose -f docker-compose.test.yml up --abort-on-container-exit guild-integration
    docker-compose -f docker-compose.test.yml down
}

# Run specific test package
run_package_test() {
    local package=$1
    echo -e "${YELLOW}Running tests for package: $package${NC}"
    docker run --rm \
        -v "$(pwd)/test-results:/test-results" \
        guild-test:latest \
        go test -v "./pkg/$package/..."
}

# Interactive test shell
test_shell() {
    echo -e "${YELLOW}Starting interactive test shell...${NC}"
    docker run --rm -it \
        -v "$(pwd):/guild" \
        -v "guild-test-home:/home/guild/.guild" \
        -w /guild \
        guild-test:latest \
        /bin/bash
}

# Clean up test artifacts
cleanup() {
    echo -e "${YELLOW}Cleaning up...${NC}"
    docker-compose -f docker-compose.test.yml down -v
    docker volume rm guild-test-home 2>/dev/null || true
    docker volume rm guild-test-results 2>/dev/null || true
    rm -rf test-results/
}

# Main execution
case "$MODE" in
    unit)
        build_image
        run_unit_tests
        ;;
    integration)
        build_image
        run_integration_tests
        ;;
    package)
        build_image
        run_package_test "$2"
        ;;
    shell)
        build_image
        test_shell
        ;;
    clean)
        cleanup
        ;;
    all)
        build_image
        run_unit_tests
        run_integration_tests
        ;;
    *)
        echo "Usage: $0 {unit|integration|package <name>|shell|clean|all}"
        exit 1
        ;;
esac

echo -e "${GREEN}✅ Docker tests completed${NC}"