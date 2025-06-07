#!/bin/bash

# Guild-Core Pre-commit Setup Script
# This script installs and configures pre-commit hooks for code quality

set -e

echo "🏰 Guild-Core Pre-commit Setup"
echo "=============================="

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper functions
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if pre-commit is installed
check_pre_commit() {
    if command -v pre-commit &> /dev/null; then
        print_success "pre-commit is already installed ($(pre-commit --version))"
        return 0
    else
        return 1
    fi
}

# Install pre-commit
install_pre_commit() {
    echo "Installing pre-commit..."
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            brew install pre-commit
        else
            print_error "Homebrew not found. Install with: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
            echo "Or install pre-commit with pip: pip install pre-commit"
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        if command -v pip3 &> /dev/null; then
            pip3 install pre-commit
        elif command -v pip &> /dev/null; then
            pip install pre-commit
        else
            print_error "pip not found. Install with: sudo apt-get install python3-pip (Ubuntu/Debian) or equivalent"
            exit 1
        fi
    else
        print_error "Unsupported OS. Please install pre-commit manually: https://pre-commit.com/#install"
        exit 1
    fi
}

# Check Go tools
check_go_tools() {
    echo ""
    echo "Checking Go tools..."
    
    # Check go fmt (built-in)
    if command -v go &> /dev/null; then
        print_success "go fmt available"
    else
        print_error "Go not installed. Please install Go first: https://golang.org/dl/"
        exit 1
    fi
    
    # Check golangci-lint
    if command -v golangci-lint &> /dev/null; then
        print_success "golangci-lint is installed ($(golangci-lint --version | head -1))"
    else
        print_warning "golangci-lint not found. Installing..."
        # Install golangci-lint
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.61.0
        print_success "golangci-lint installed"
    fi
    
    # Check goimports-reviser
    if command -v goimports-reviser &> /dev/null; then
        print_success "goimports-reviser is installed"
    else
        print_warning "goimports-reviser not found. Installing..."
        go install -v github.com/incu6us/goimports-reviser/v3@latest
        print_success "goimports-reviser installed"
    fi
}

# Install pre-commit hooks
setup_hooks() {
    echo ""
    echo "Setting up pre-commit hooks..."
    
    # Install the pre-commit hooks
    pre-commit install
    print_success "Pre-commit hooks installed"
    
    # Install commit-msg hook for conventional commits (optional)
    pre-commit install --hook-type commit-msg 2>/dev/null || true
    
    # Run pre-commit on all files to check current state
    echo ""
    echo "Running initial pre-commit check (this may take a moment)..."
    if pre-commit run --all-files; then
        print_success "All pre-commit checks passed!"
    else
        print_warning "Some pre-commit checks failed. This is normal for initial setup."
        print_warning "Run 'pre-commit run --all-files' to see details and fix issues."
    fi
}

# Create git hooks directory if it doesn't exist
ensure_git_hooks_dir() {
    if [ ! -d ".git/hooks" ]; then
        mkdir -p .git/hooks
        print_success "Created .git/hooks directory"
    fi
}

# Main execution
main() {
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository. Please run from the guild-core root directory."
        exit 1
    fi
    
    # Check if .pre-commit-config.yaml exists
    if [ ! -f ".pre-commit-config.yaml" ]; then
        print_error ".pre-commit-config.yaml not found. Please run from the guild-core root directory."
        exit 1
    fi
    
    # Install pre-commit if needed
    if ! check_pre_commit; then
        install_pre_commit
    fi
    
    # Check Go tools
    check_go_tools
    
    # Ensure git hooks directory exists
    ensure_git_hooks_dir
    
    # Setup hooks
    setup_hooks
    
    echo ""
    print_success "Pre-commit setup complete! 🎉"
    echo ""
    echo "Pre-commit will now run automatically before each commit."
    echo "To run manually: pre-commit run --all-files"
    echo "To skip temporarily: git commit --no-verify"
    echo ""
    echo "Recommended: Add to your Makefile:"
    echo "  make pre-commit    # Run pre-commit checks"
    echo "  make pre-commit-update  # Update pre-commit hooks"
}

# Run main function
main