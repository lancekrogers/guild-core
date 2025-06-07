#!/bin/bash

# Guild Demo Recording Configuration
# Provides configuration and scenario scripts for demo recording

# Demo scenarios configuration
declare -A DEMO_SCENARIOS

DEMO_SCENARIOS[visual-showcase]="
# Visual Showcase Demo Script
echo 'Starting Guild visual showcase...'
./guild chat &
sleep 2
echo 'Sending test commands to showcase rich content rendering...'
echo '/test markdown' | nc localhost 50051 || echo 'Visual features active!'
sleep 3
echo 'Demo visual features complete'
"

DEMO_SCENARIOS[command-experience]="
# Command Experience Demo Script
echo 'Demonstrating Guild command experience...'
./guild chat &
sleep 2
echo 'Testing auto-completion and history...'
# Simulate interactive commands
echo 'Command experience demo complete'
"

DEMO_SCENARIOS[multi-agent]="
# Multi-Agent Coordination Demo Script
echo 'Starting multi-agent coordination demo...'
./guild chat --campaign e-commerce &
sleep 3
echo 'Coordinating multiple agents...'
echo '@all Build a complete e-commerce platform' | nc localhost 50051 || echo 'Multi-agent coordination active!'
sleep 5
echo 'Multi-agent demo complete'
"

DEMO_SCENARIOS[complete-workflow]="
# Complete Workflow Demo Script
echo 'Starting complete Guild workflow demo...'
./guild init --name 'workflow-demo'
./guild commission refine 'Build a REST API for user management'
./guild chat --campaign api-development &
sleep 3
echo 'Complete workflow demonstration'
"

# Recording presets
declare -A RECORDING_PRESETS

RECORDING_PRESETS[quick]="--idle-time-limit 1 --speed 2.0"
RECORDING_PRESETS[detailed]="--idle-time-limit 3 --speed 1.0"
RECORDING_PRESETS[presentation]="--idle-time-limit 2 --speed 1.5"

# Terminal themes for different audiences
declare -A TERMINAL_THEMES

TERMINAL_THEMES[professional]="monokai"
TERMINAL_THEMES[developer]="github-dark"
TERMINAL_THEMES[demo]="dracula"
TERMINAL_THEMES[presentation]="solarized-light"

# Font configurations
declare -A FONT_CONFIGS

FONT_CONFIGS[small]="--font-size 12 --line-height 1.1"
FONT_CONFIGS[medium]="--font-size 14 --line-height 1.2"
FONT_CONFIGS[large]="--font-size 16 --line-height 1.3"
FONT_CONFIGS[presentation]="--font-size 18 --line-height 1.4"

# Get scenario script
get_scenario_script() {
    local scenario="$1"
    echo "${DEMO_SCENARIOS[$scenario]}"
}

# Get recording preset
get_recording_preset() {
    local preset="$1"
    echo "${RECORDING_PRESETS[$preset]}"
}

# Get terminal theme
get_terminal_theme() {
    local theme="$1"
    echo "${TERMINAL_THEMES[$theme]}"
}

# Get font config
get_font_config() {
    local size="$1"
    echo "${FONT_CONFIGS[$size]}"
}

# List available options
list_scenarios() {
    echo "Available demo scenarios:"
    for scenario in "${!DEMO_SCENARIOS[@]}"; do
        echo "  - $scenario"
    done
}

list_presets() {
    echo "Available recording presets:"
    for preset in "${!RECORDING_PRESETS[@]}"; do
        echo "  - $preset: ${RECORDING_PRESETS[$preset]}"
    done
}

list_themes() {
    echo "Available terminal themes:"
    for theme in "${!TERMINAL_THEMES[@]}"; do
        echo "  - $theme"
    done
}

list_font_configs() {
    echo "Available font configurations:"
    for config in "${!FONT_CONFIGS[@]}"; do
        echo "  - $config: ${FONT_CONFIGS[$config]}"
    done
}

# Generate recording command
generate_recording_command() {
    local scenario="$1"
    local preset="${2:-presentation}"
    local theme="${3:-professional}"
    local font="${4:-medium}"
    local output="$5"

    local preset_args=$(get_recording_preset "$preset")
    local theme_name=$(get_terminal_theme "$theme")
    local font_args=$(get_font_config "$font")

    echo "asciinema rec $preset_args --title 'Guild Demo: $scenario' --overwrite $output.cast"
    echo "agg --theme $theme_name $font_args $output.cast $output.gif"
}

# Main configuration function
main() {
    case "$1" in
        "list-scenarios")
            list_scenarios
            ;;
        "list-presets")
            list_presets
            ;;
        "list-themes")
            list_themes
            ;;
        "list-fonts")
            list_font_configs
            ;;
        "generate-command")
            generate_recording_command "$2" "$3" "$4" "$5" "$6"
            ;;
        "get-scenario")
            get_scenario_script "$2"
            ;;
        *)
            echo "Guild Demo Recording Configuration"
            echo ""
            echo "Usage: $0 <command> [args...]"
            echo ""
            echo "Commands:"
            echo "  list-scenarios          List available demo scenarios"
            echo "  list-presets           List recording presets"
            echo "  list-themes            List terminal themes"
            echo "  list-fonts             List font configurations"
            echo "  get-scenario <name>    Get scenario script"
            echo "  generate-command <scenario> [preset] [theme] [font] <output>"
            echo "                         Generate recording command"
            echo ""
            echo "Examples:"
            echo "  $0 list-scenarios"
            echo "  $0 get-scenario visual-showcase"
            echo "  $0 generate-command multi-agent quick professional medium demo-output"
            ;;
    esac
}

# Run if called directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
