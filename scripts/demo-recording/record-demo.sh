#!/bin/bash

# Guild Demo Recording Script
# Records professional demos of Guild multi-agent development framework

set -e

echo "🎬 Guild Demo Recording Setup"
echo "=============================="

# Configuration
DEMO_DIR="demo-assets"
TITLE="Guild: Multi-Agent Development Framework"
IDLE_TIME_LIMIT=2
SPEED=1.5
FONT_FAMILY="SF Mono,Monaco,Consolas,monospace"
FONT_SIZE=14
LINE_HEIGHT=1.2
THEME="monokai"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Helper functions
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
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

# Check dependencies
check_dependencies() {
    print_status "Checking recording dependencies..."

    local missing_tools=()

    if ! command -v asciinema &> /dev/null; then
        missing_tools+=("asciinema")
    fi

    if ! command -v agg &> /dev/null; then
        missing_tools+=("agg")
    fi

    if [ ${#missing_tools[@]} -eq 0 ]; then
        print_success "All recording tools available"
    else
        print_error "Missing tools: ${missing_tools[*]}"
        echo ""
        echo "Install missing tools:"
        echo "  macOS: brew install asciinema agg"
        echo "  Ubuntu: apt-get install asciinema && npm install -g @asciinema/agg"
        exit 1
    fi
}

# Pre-flight checks
run_preflight_checks() {
    print_status "Running pre-flight checks..."

    # Check if guild binary exists
    if ! command -v ./guild &> /dev/null && ! command -v guild &> /dev/null; then
        print_error "Guild binary not found. Build with 'make build' first."
        exit 1
    fi

    # Run demo-check if available
    if ./guild demo-check --api-keys --performance 2>/dev/null; then
        print_success "Demo environment validation passed"
    else
        print_warning "Demo validation had issues - proceeding anyway"
    fi

    print_success "Pre-flight checks completed"
}

# Terminal setup
setup_terminal() {
    print_status "Setting up terminal environment..."

    # Resize terminal to demo-friendly size
    printf '\e[8;40;120t' 2>/dev/null || true

    # Clear screen
    clear

    # Set environment variables for best visual experience
    export GUILD_THEME="medieval"
    export COLORTERM="truecolor"
    export TERM="xterm-256color"

    # Disable command history during recording
    export HISTFILE=""

    print_success "Terminal configured for recording"
}

# Prepare demo environment
prepare_demo_environment() {
    print_status "Preparing demo environment..."

    # Create demo assets directory
    mkdir -p "$DEMO_DIR"

    # Start with clean Guild state
    if [ -d ".guild" ]; then
        print_warning "Removing existing .guild directory for clean demo"
        rm -rf .guild
    fi

    # Initialize fresh Guild project
    if command -v ./guild &> /dev/null; then
        ./guild init --quiet --name "demo-guild" --description "Demo project for Guild framework"
    elif command -v guild &> /dev/null; then
        guild init --quiet --name "demo-guild" --description "Demo project for Guild framework"
    else
        print_error "Guild command not found"
        exit 1
    fi

    print_success "Demo environment prepared"
}

# Pre-warm caches
prewarm_caches() {
    print_status "Pre-warming caches for smooth demo..."

    # Run a quick commission refinement to warm up systems
    if [ -f "examples/commissions/task-management-api.md" ]; then
        ./guild commission refine examples/commissions/task-management-api.md --output /tmp/guild-prewarm > /dev/null 2>&1 || true
        rm -rf /tmp/guild-prewarm
    fi

    # Pre-load any other components
    ./guild agents list > /dev/null 2>&1 || true

    print_success "Caches pre-warmed"
}

# Record individual scenario
record_scenario() {
    local scenario_name="$1"
    local scenario_file="$2"
    local output_name="$3"

    print_status "Recording scenario: $scenario_name"

    echo ""
    echo "🎬 Recording will start in 3 seconds..."
    echo "   Scenario: $scenario_name"
    echo "   Output: $DEMO_DIR/$output_name.cast"
    echo ""

    sleep 3

    # Record with asciinema
    asciinema rec \
        --title "$TITLE - $scenario_name" \
        --idle-time-limit "$IDLE_TIME_LIMIT" \
        --overwrite \
        "$DEMO_DIR/$output_name.cast"

    print_success "Recording completed: $output_name.cast"
}

# Convert to GIF
convert_to_gif() {
    local input_file="$1"
    local output_file="$2"

    print_status "Converting $input_file to GIF..."

    agg \
        --theme "$THEME" \
        --font-family "$FONT_FAMILY" \
        --font-size "$FONT_SIZE" \
        --line-height "$LINE_HEIGHT" \
        --speed "$SPEED" \
        "$DEMO_DIR/$input_file.cast" \
        "$DEMO_DIR/$output_file.gif"

    print_success "GIF created: $output_file.gif"
}

# Record all scenarios
record_all_scenarios() {
    local scenarios=(
        "Visual Showcase:demo-visual-showcase"
        "Command Experience:demo-command-experience"
        "Multi-Agent Coordination:demo-multi-agent"
        "Complete Workflow:demo-complete-workflow"
    )

    for scenario_info in "${scenarios[@]}"; do
        IFS=':' read -ra SCENARIO <<< "$scenario_info"
        local name="${SCENARIO[0]}"
        local file="${SCENARIO[1]}"

        echo ""
        echo "🎯 Next scenario: $name"
        read -p "Press Enter to start recording, or 's' to skip: " -n 1 -r
        echo ""

        if [[ ! $REPLY =~ ^[Ss]$ ]]; then
            record_scenario "$name" "" "$file"

            # Ask about GIF conversion
            read -p "Convert to GIF? (y/N): " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                convert_to_gif "$file" "$file"
            fi
        else
            print_warning "Skipped: $name"
        fi
    done
}

# Interactive recording session
interactive_recording() {
    print_status "Starting interactive recording session..."

    echo ""
    echo "🎭 Guild Demo Recording Session"
    echo "==============================="
    echo ""
    echo "Available recording modes:"
    echo "  1. Quick visual showcase (2 minutes)"
    echo "  2. Command experience demo (90 seconds)"
    echo "  3. Multi-agent coordination (3 minutes)"
    echo "  4. Complete workflow demo (5 minutes)"
    echo "  5. Custom recording"
    echo "  6. Record all scenarios"
    echo ""

    read -p "Select recording mode (1-6): " -n 1 -r
    echo ""

    case $REPLY in
        1)
            record_scenario "Visual Showcase" "" "demo-visual-showcase"
            convert_to_gif "demo-visual-showcase" "demo-visual-showcase"
            ;;
        2)
            record_scenario "Command Experience" "" "demo-command-experience"
            convert_to_gif "demo-command-experience" "demo-command-experience"
            ;;
        3)
            record_scenario "Multi-Agent Coordination" "" "demo-multi-agent"
            convert_to_gif "demo-multi-agent" "demo-multi-agent"
            ;;
        4)
            record_scenario "Complete Workflow" "" "demo-complete-workflow"
            convert_to_gif "demo-complete-workflow" "demo-complete-workflow"
            ;;
        5)
            read -p "Enter scenario name: " scenario_name
            read -p "Enter output filename (without extension): " output_name
            record_scenario "$scenario_name" "" "$output_name"
            read -p "Convert to GIF? (y/N): " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                convert_to_gif "$output_name" "$output_name"
            fi
            ;;
        6)
            record_all_scenarios
            ;;
        *)
            print_error "Invalid selection"
            exit 1
            ;;
    esac
}

# Show final results
show_results() {
    print_success "Demo recording session completed!"

    echo ""
    echo "📦 Generated files in $DEMO_DIR/:"
    ls -la "$DEMO_DIR/" | grep -E '\.(cast|gif)$' || echo "  No demo files found"

    echo ""
    echo "🚀 Next steps:"
    echo "  1. Review recordings in $DEMO_DIR/"
    echo "  2. Upload GIFs to documentation"
    echo "  3. Share cast files for interactive playback"
    echo "  4. Create social media posts with GIFs"

    echo ""
    echo "📋 Tips for sharing:"
    echo "  - Upload .gif files to GitHub/docs for embedding"
    echo "  - Share .cast files with 'asciinema play filename.cast'"
    echo "  - Use .gif files in README.md and marketing materials"
}

# Cleanup function
cleanup() {
    print_status "Cleaning up..."

    # Restore environment
    unset GUILD_THEME COLORTERM HISTFILE

    # Reset terminal if needed
    printf '\e[8;24;80t' 2>/dev/null || true
}

# Main execution
main() {
    echo "🏰 Guild Framework Demo Recording"
    echo "=================================="
    echo ""

    # Set up cleanup trap
    trap cleanup EXIT

    # Run all setup steps
    check_dependencies
    run_preflight_checks
    setup_terminal
    prepare_demo_environment
    prewarm_caches

    # Start interactive recording
    interactive_recording

    # Show results
    show_results
}

# Handle command line arguments
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Guild Demo Recording Script"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --help, -h     Show this help message"
    echo "  --check        Run pre-flight checks only"
    echo "  --clean        Clean demo environment and exit"
    echo ""
    echo "Interactive mode will start if no options provided."
    exit 0
elif [ "$1" = "--check" ]; then
    check_dependencies
    run_preflight_checks
    echo "✅ All checks passed - ready for recording"
    exit 0
elif [ "$1" = "--clean" ]; then
    print_status "Cleaning demo environment..."
    rm -rf .guild demo-assets /tmp/guild-prewarm
    print_success "Demo environment cleaned"
    exit 0
else
    # Run main interactive session
    main
fi
