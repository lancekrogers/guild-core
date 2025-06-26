#!/bin/bash

# Test Directory Cleanup Script for Guild Framework
# Removes development artifacts while preserving useful content
# Ensures compliance with enterprise open-source standards

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUILD_CORE_DIR="$(dirname "$SCRIPT_DIR")"

echo "🧹 Guild Framework Test Directory Cleanup"
echo "=========================================="
echo "Working directory: $GUILD_CORE_DIR"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Change to guild-core directory
cd "$GUILD_CORE_DIR"

# Verify we're in the right directory
if [[ ! -f "go.mod" ]] || ! grep -q "guild-core" go.mod; then
    print_error "Not in guild-core directory! Expected to find go.mod with guild-core module."
    exit 1
fi

print_status "Verified guild-core directory structure"

# Step 1: Preserve useful content
print_status "Step 1: Preserving useful test content..."

# Create directories for preserved content
mkdir -p scripts examples/manual_testing

# Preserve test_elena.sh if it exists
if [[ -f "test-elena-welcome/test_elena.sh" ]]; then
    print_status "Preserving test_elena.sh → scripts/test_elena_welcome.sh"
    cp "test-elena-welcome/test_elena.sh" "scripts/test_elena_welcome.sh"
    chmod +x "scripts/test_elena_welcome.sh"
fi

# Preserve manual test files
if [[ -f "test_manual/test_chat.go" ]]; then
    print_status "Preserving test_chat.go → examples/manual_testing/"
    cp "test_manual/test_chat.go" "examples/manual_testing/"
fi

# Look for any other potentially useful files
print_status "Scanning for other useful content..."
useful_files_found=0

for test_dir in test-* manual-test hash-test; do
    if [[ -d "$test_dir" ]]; then
        # Look for non-empty files that might be useful
        while IFS= read -r -d '' file; do
            if [[ -s "$file" ]] && [[ ! "$file" =~ \.(log|tmp|bak)$ ]]; then
                print_warning "Found potentially useful file: $file"
                echo "  → Consider manually reviewing before deletion"
                ((useful_files_found++))
            fi
        done < <(find "$test_dir" -type f -print0 2>/dev/null || true)
    fi
done

if [[ $useful_files_found -gt 0 ]]; then
    print_warning "Found $useful_files_found potentially useful files in test directories"
    read -p "Continue with cleanup? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Cleanup cancelled. Review files manually."
        exit 0
    fi
fi

# Step 2: Remove test directories
print_status "Step 2: Removing temporary test directories..."

# List of test directories to remove
test_dirs=(
    "test-campaign-fix"
    "test-chat-fix"
    "test-chat-modular"
    "test-daemon-design"
    "test-elena-welcome"
    "test-enhanced-init"
    "test-enter-fix"
    "test-final"
    "test-guild-chat"
    "test-guild-init"
    "test-init"
    "test-unified-init-fixed"
    "test_manual"
    "manual-test"
    "hash-test"
    "guild-core/test-detection"
    "guild-core/test-enhanced-init"
)

removed_count=0
for dir in "${test_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        print_status "Removing: $dir"
        rm -rf "$dir"
        ((removed_count++))
    fi
done

print_success "Removed $removed_count test directories"

# Step 3: Remove coverage files
print_status "Step 3: Removing coverage artifacts..."

coverage_files=(
    "client_coverage.out"
    "daemon_coverage.out"
    "coverage.out"
    "*.out"
)

coverage_removed=0
for pattern in "${coverage_files[@]}"; do
    for file in $pattern; do
        if [[ -f "$file" ]]; then
            print_status "Removing coverage file: $file"
            rm "$file"
            ((coverage_removed++))
        fi
    done
done

if [[ $coverage_removed -gt 0 ]]; then
    print_success "Removed $coverage_removed coverage files"
else
    print_status "No coverage files found to remove"
fi

# Step 4: Update .gitignore
print_status "Step 4: Updating .gitignore..."

gitignore_entries=(
    "# Test artifacts"
    "test-*/"
    "manual-test/"
    "hash-test/"
    ""
    "# Coverage files"
    "*.out"
    "coverage/"
    ""
    "# Build artifacts"
    "bin/guild"
    ".test-*.tmp"
    ""
)

# Check if .gitignore exists
if [[ ! -f ".gitignore" ]]; then
    print_status "Creating .gitignore"
    touch .gitignore
fi

# Add entries if they don't exist
gitignore_updated=false
for entry in "${gitignore_entries[@]}"; do
    if [[ -n "$entry" ]] && ! grep -Fxq "$entry" .gitignore; then
        echo "$entry" >> .gitignore
        gitignore_updated=true
    elif [[ -z "$entry" ]]; then
        echo "" >> .gitignore
    fi
done

if [[ "$gitignore_updated" = true ]]; then
    print_success "Updated .gitignore with test artifact patterns"
else
    print_status ".gitignore already contains necessary patterns"
fi

# Step 5: Verification
print_status "Step 5: Verifying cleanup..."

# Check for remaining test artifacts
remaining_artifacts=()

# Look for test-* directories
while IFS= read -r -d '' dir; do
    remaining_artifacts+=("$dir")
done < <(find . -maxdepth 1 -name "test-*" -type d -print0 2>/dev/null || true)

# Look for coverage files
while IFS= read -r -d '' file; do
    remaining_artifacts+=("$file")
done < <(find . -maxdepth 1 -name "*.out" -type f -print0 2>/dev/null || true)

if [[ ${#remaining_artifacts[@]} -eq 0 ]]; then
    print_success "✅ Cleanup completed successfully!"
    print_success "✅ No remaining test artifacts found"
    print_success "✅ Repository now meets enterprise standards"
else
    print_warning "⚠️  Some artifacts may remain:"
    for artifact in "${remaining_artifacts[@]}"; do
        echo "  - $artifact"
    done
fi

# Final summary
echo
echo "📊 Cleanup Summary"
echo "=================="
echo "• Test directories removed: $removed_count"
echo "• Coverage files removed: $coverage_removed"
echo "• Useful content preserved in scripts/ and examples/"
echo "• .gitignore updated with prevention patterns"
echo
print_success "Guild Framework test directory cleanup complete!"
print_status "Ready for enterprise open-source release! 🚀"