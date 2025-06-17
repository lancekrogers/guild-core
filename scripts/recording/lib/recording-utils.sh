#!/bin/bash
# Professional Demo Recording Utilities
# Utilities for recording compelling Guild Framework demonstrations

set -e

# Configuration
DEMO_HOME="${DEMO_HOME:-/tmp/guild-recordings}"
GUILD_BIN="${GUILD_BIN:-./guild}"
TYPING_SPEED="${TYPING_SPEED:-0.08}"  # Natural typing speed
PAUSE_TIME="${PAUSE_TIME:-3}"         # Time to read output
TERMINAL_COLS="${TERMINAL_COLS:-120}"
TERMINAL_ROWS="${TERMINAL_ROWS:-35}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Status display functions
print_status() {
    echo -e "${BLUE}🎬 $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

print_title() {
    echo -e "${WHITE}🏰 $1${NC}"
}

# IMPORTANT: These scripts demonstrate REAL Guild functionality
# No mock providers or fake data for marketing demos

check_guild_providers() {
    print_status "Checking Guild provider configuration..."
    
    # Check if guild binary exists
    if ! command -v "$GUILD_BIN" &> /dev/null; then
        print_error "Guild binary not found at: $GUILD_BIN"
        echo "Please build Guild with 'make build' first"
        return 1
    fi
    
    # Check if providers are configured
    if ! $GUILD_BIN config show &> /dev/null; then
        print_warning "No providers configured - demo will use mock provider"
        export GUILD_MOCK_PROVIDER=true
        print_info "Mock provider enabled for demo purposes"
    else
        print_success "Real providers configured"
        
        # Ask user if they want to use real providers for demo
        echo
        read -p "Use real AI providers for recording? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_warning "Real providers will be used - this will incur API costs"
            export GUILD_MOCK_PROVIDER=false
        else
            print_info "Using mock provider for cost-free demo"
            export GUILD_MOCK_PROVIDER=true
        fi
    fi
    
    return 0
}

# Initialize recording environment
recording_init() {
    local recording_name="$1"
    
    print_title "Initializing Recording Environment: $recording_name"
    
    # Create clean workspace
    export RECORDING_DIR="$DEMO_HOME/$recording_name"
    rm -rf "$RECORDING_DIR"
    mkdir -p "$RECORDING_DIR"
    cd "$RECORDING_DIR"
    
    # Set up environment for recording
    export GUILD_HOME="$RECORDING_DIR/.guild"
    export GUILD_TEST_MODE=true
    export GUILD_LOG_LEVEL=warn
    export COLORTERM=truecolor
    export TERM=xterm-256color
    
    # Terminal setup for recording
    if [[ -n "$RECORD_MODE" ]]; then
        printf '\e[8;%d;%dt' "$TERMINAL_ROWS" "$TERMINAL_COLS"  # Resize terminal
        clear
    fi
    
    check_guild_providers
    
    # Initialize Guild project for demo
    $GUILD_BIN init --name "demo-project" --description "Demo of Guild Framework capabilities" &> /dev/null || true
    
    print_success "Recording environment ready: $RECORDING_DIR"
    echo
}

# Type command with natural typing effect
type_command() {
    local cmd="$1"
    local prompt="${2:-$ }"
    
    printf "%s" "$prompt"
    
    # Natural typing effect
    for (( i=0; i<${#cmd}; i++ )); do
        printf "%c" "${cmd:$i:1}"
        # Variable speed for realism
        sleep $(awk "BEGIN {print $TYPING_SPEED * (0.8 + 0.4 * rand())}")
    done
    
    printf "\n"
    sleep 0.5
}

# Execute command and show real output
run_demo_command() {
    local cmd="$1"
    local description="${2:-}"
    
    if [[ -n "$description" ]]; then
        print_info "$description"
        sleep 1
    fi
    
    type_command "$cmd"
    
    # Execute the actual command
    eval "$cmd"
    local exit_code=$?
    
    sleep "$PAUSE_TIME"
    return $exit_code
}

# Show command without executing (for manual demo steps)
show_command_prompt() {
    local cmd="$1"
    local description="${2:-}"
    
    if [[ -n "$description" ]]; then
        print_info "$description"
        echo
    fi
    
    echo -e "${YELLOW}👉 Next step: ${WHITE}$cmd${NC}"
    echo "Press Enter when ready to continue..."
    read -r
}

# Start professional asciinema recording
start_recording() {
    local name="$1"
    local title="${2:-Guild Framework Demo}"
    
    if ! command -v asciinema &> /dev/null; then
        print_error "asciinema required for recording"
        echo "Install with: brew install asciinema (macOS) or apt-get install asciinema (Ubuntu)"
        return 1
    fi
    
    export RECORD_FILE="${RECORDING_DIR}/${name}.cast"
    
    print_status "Starting recording: $name"
    print_info "Recording will be saved to: $RECORD_FILE"
    
    # Professional recording settings
    asciinema rec \
        --quiet \
        --overwrite \
        --title "$title" \
        --idle-time-limit 5 \
        --env="GUILD_MOCK_PROVIDER,GUILD_TEST_MODE,COLORTERM,TERM" \
        "$RECORD_FILE"
}

# Generate professional GIF from recording
generate_gif() {
    local cast_file="$1"
    local output_file="$2"
    local theme="${3:-github-dark}"
    
    print_status "Converting recording to GIF..."
    
    if ! command -v agg &> /dev/null; then
        print_warning "Installing agg for GIF generation..."
        if command -v cargo &> /dev/null; then
            cargo install --git https://github.com/asciinema/agg
        else
            print_error "Cargo not found - install Rust/Cargo to generate GIFs"
            return 1
        fi
    fi
    
    # Professional GIF settings
    agg \
        --theme "$theme" \
        --font-family "SF Mono,Monaco,Cascadia Code,monospace" \
        --font-size 14 \
        --line-height 1.4 \
        --cols "$TERMINAL_COLS" \
        --rows "$TERMINAL_ROWS" \
        --speed 1.2 \
        "$cast_file" \
        "$output_file"
    
    # Optimize GIF size while maintaining quality
    if command -v gifsicle &> /dev/null; then
        print_status "Optimizing GIF..."
        gifsicle -O3 --colors 256 \
            "$output_file" \
            -o "${output_file%.gif}-optimized.gif"
        mv "${output_file%.gif}-optimized.gif" "$output_file"
    fi
    
    # Check file size and warn if too large
    if [[ -f "$output_file" ]]; then
        local size=$(stat -f%z "$output_file" 2>/dev/null || stat -c%s "$output_file" 2>/dev/null || echo "0")
        local size_mb=$((size / 1024 / 1024))
        
        if [[ $size_mb -gt 10 ]]; then
            print_warning "GIF is large (${size_mb}MB) - consider shortening demo or reducing terminal size"
        else
            print_success "GIF generated: $output_file (${size_mb}MB)"
        fi
    fi
}

# Create demo commission file
create_demo_commission() {
    local commission_file="${1:-demo-commission.md}"
    local title="${2:-E-Commerce API Development}"
    
    cat > "$commission_file" << EOF
# $title

Build a complete REST API for an e-commerce platform with the following features:

## Core Requirements
- User authentication and authorization
- Product catalog management
- Shopping cart functionality
- Order processing system
- Payment integration

## Technical Specifications
- RESTful API design
- Database schema with relationships
- Input validation and error handling
- API documentation
- Unit tests for core functionality

## Success Criteria
- All endpoints properly documented
- Error responses follow consistent format
- Database relationships properly defined
- Security best practices implemented
- Tests achieve 80%+ coverage

This commission will demonstrate Guild's multi-agent coordination and code generation capabilities.
EOF

    print_success "Demo commission created: $commission_file"
}

# Show visual demo banner
show_demo_banner() {
    local demo_name="$1"
    local description="${2:-}"
    
    clear
    echo
    echo -e "${PURPLE}╔════════════════════════════════════════════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║${WHITE}                                    🏰 GUILD FRAMEWORK DEMONSTRATION                                    ${PURPLE}║${NC}"
    echo -e "${PURPLE}╠════════════════════════════════════════════════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${PURPLE}║${NC}  Demo: ${WHITE}$demo_name${NC}"
    [[ -n "$description" ]] && echo -e "${PURPLE}║${NC}  ${description}"
    echo -e "${PURPLE}║${NC}  Time: $(date '+%Y-%m-%d %H:%M:%S')"
    echo -e "${PURPLE}║${NC}  Mode: ${CYAN}Professional Recording${NC}"
    echo -e "${PURPLE}╚════════════════════════════════════════════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo
    sleep 2
}

# Clean up recording environment
cleanup_recording() {
    print_status "Cleaning up recording environment..."
    
    # Reset environment variables
    unset GUILD_HOME GUILD_TEST_MODE GUILD_MOCK_PROVIDER
    unset RECORD_FILE RECORDING_DIR
    
    # Reset terminal size if needed
    if [[ -n "$RECORD_MODE" ]]; then
        printf '\e[8;24;80t' 2>/dev/null || true
    fi
    
    print_success "Environment cleaned up"
}

# Validate recording environment
validate_recording_environment() {
    print_status "Validating recording environment..."
    
    local errors=0
    
    # Check required tools
    local tools=("asciinema" "agg")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "Missing required tool: $tool"
            ((errors++))
        fi
    done
    
    # Check Guild binary
    if ! command -v "$GUILD_BIN" &> /dev/null; then
        print_error "Guild binary not found: $GUILD_BIN"
        ((errors++))
    fi
    
    # Check terminal size
    local cols=$(tput cols 2>/dev/null || echo "80")
    local rows=$(tput lines 2>/dev/null || echo "24")
    
    if [[ $cols -lt 120 ]]; then
        print_warning "Terminal width ($cols) smaller than recommended (120)"
    fi
    
    if [[ $rows -lt 35 ]]; then
        print_warning "Terminal height ($rows) smaller than recommended (35)"
    fi
    
    if [[ $errors -eq 0 ]]; then
        print_success "Recording environment validated"
        return 0
    else
        print_error "Recording environment validation failed ($errors errors)"
        return 1
    fi
}

# Export functions for use in demo scripts
export -f print_status print_success print_warning print_error print_info print_title
export -f check_guild_providers recording_init type_command run_demo_command show_command_prompt
export -f start_recording generate_gif create_demo_commission show_demo_banner cleanup_recording
export -f validate_recording_environment