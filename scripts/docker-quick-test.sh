#!/bin/bash
# Ultra-quick Docker tests for rapid iteration

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🚀 Guild Quick Docker Test${NC}"

# Build if needed
if [[ "$(docker images -q guild-user:latest 2> /dev/null)" == "" ]]; then
    echo "Building image..."
    docker build -f Dockerfile.user -t guild-user:latest . > /dev/null 2>&1
fi

# Test command
TEST_CMD=${1:-"guild --version"}

echo -e "Testing: ${GREEN}$TEST_CMD${NC}"
echo "---"

# Run test
docker run --rm guild-user:latest bash -c "$TEST_CMD" || {
    echo -e "${RED}❌ Test failed${NC}"
    exit 1
}

echo "---"
echo -e "${GREEN}✅ Test passed${NC}"