#!/bin/bash
# Demo script to showcase the new build system

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}        Guild Framework - Build System Demo${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo -e "${YELLOW}This demo shows the new visual build system in action.${NC}"
echo -e "${YELLOW}Watch the beautiful progress bars and status indicators!${NC}"
echo
echo "Press Enter to start the demo..."
read

# Clean build
echo -e "\n${GREEN}1. First, let's clean any existing build artifacts:${NC}"
echo "   Running: make clean"
echo
sleep 1
make -f Makefile.simple clean

echo
echo "Press Enter to continue..."
read

# Build
echo -e "\n${GREEN}2. Now let's build the Guild CLI with visual progress:${NC}"
echo "   Running: make build"
echo
sleep 1
make -f Makefile.simple build

echo
echo "Press Enter to continue..."
read

# Test
echo -e "\n${GREEN}3. Run the test suite with live progress tracking:${NC}"
echo "   Running: make test"
echo
sleep 1
# Just show a quick demo, don't run all tests
go run tools/buildtool/main.go test | head -50
echo -e "\n${YELLOW}... (truncated for demo)${NC}"

echo
echo "Press Enter to continue..."
read

# Show direct usage
echo -e "\n${GREEN}4. You can also use the build tool directly:${NC}"
echo
echo "   For verbose output:"
echo "   ${BLUE}go run tools/buildtool/main.go -v build${NC}"
echo
echo "   For CI environments:"
echo "   ${BLUE}go run tools/buildtool/main.go -no-color test${NC}"
echo
echo "   For everything at once:"
echo "   ${BLUE}go run tools/buildtool/main.go all${NC}"

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Demo complete!${NC}"
echo
echo "The new build system provides:"
echo "  ✓ Reliable progress bars that never break"
echo "  ✓ Beautiful visual feedback"
echo "  ✓ Consistent behavior across platforms"
echo "  ✓ Easy to maintain Go code instead of complex shell scripts"
echo
echo -e "${YELLOW}To switch to the new system:${NC}"
echo "  cp Makefile.simple Makefile"
echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"