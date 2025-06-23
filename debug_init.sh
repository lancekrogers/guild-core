#!/bin/bash
# Debug init to see what's happening

set -e

echo "=== Debug Guild Init ==="

TEST_DIR="test-init-$(date +%s)"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

echo "Working directory: $(pwd)"

# Run init with strace if available to see file operations
if command -v strace &> /dev/null; then
    echo "Running with strace..."
    strace -e trace=open,openat,mkdir,mkdirat,write -o init_trace.log ../guild init --quick
else
    echo "Running without strace..."
    ../guild init --quick --verbose 2>&1 | tee init_output.log
fi

echo
echo "=== Checking results ==="

# Check all files created
echo "All files in directory:"
find . -type f -o -type d | sort

# Check parent directory for any .guild or .campaign
echo
echo "Checking parent directory:"
ls -la ../ | grep -E "guild|campaign" || echo "No guild/campaign directories in parent"

# Check if files were created elsewhere
echo
echo "Checking home directory:"
ls -la ~/ | grep -E "\.guild|\.campaign" || echo "No guild/campaign in home"

echo
echo "Directory preserved at: $(pwd)"