#!/bin/bash
# Demo Validation System - Automated testing of all demo scripts
# Ensures demos work reliably across different environments

source "$(dirname "$0")/lib/recording-utils.sh"

# Validation configuration
VALIDATION_LOG="/tmp/guild-demo-validation.log"
E2E_TEST_DIR="$(dirname "$0")/../../integration/e2e"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
VALIDATION_ERRORS=()

main() {
    local mode="${1:-full}"
    
    case "$mode" in
        "quick")
            run_quick_validation
            ;;
        "full")
            run_full_validation
            ;;
        "environment")
            validate_environment_only
            ;;
        "scripts")
            validate_scripts_only
            ;;
        "performance")
            run_performance_validation
            ;;
        "ci")
            run_ci_validation
            ;;
        *)
            show_validation_help
            exit 1
            ;;
    esac
}

run_full_validation() {
    print_title "Guild Framework Demo Validation Suite"
    print_info "Comprehensive validation of all demonstration components"
    echo
    
    # Initialize validation log
    {
        echo "Guild Demo Validation Report"
        echo "Generated: $(date)"
        echo "========================================"
        echo
    } > "$VALIDATION_LOG"
    
    # Run all validation categories
    validate_environment_comprehensive
    validate_demo_scripts
    validate_recording_tools
    validate_demo_content
    validate_integration_tests
    run_performance_checks
    
    # Generate final report
    generate_validation_report
}

run_quick_validation() {
    print_title "Quick Demo Validation"
    print_info "Essential checks for demo readiness"
    echo
    
    {
        echo "Quick Demo Validation Report"
        echo "Generated: $(date)"
        echo "========================================"
        echo
    } > "$VALIDATION_LOG"
    
    # Essential validations only
    validate_environment_essential
    validate_demo_scripts_syntax
    validate_recording_tools_available
    
    generate_quick_report
}

validate_environment_comprehensive() {
    print_status "Validating comprehensive demo environment..."
    
    local env_errors=0
    
    # System requirements
    test_case "System Requirements" validate_system_requirements
    test_case "Terminal Capabilities" validate_terminal_capabilities
    test_case "Network Connectivity" validate_network_connectivity
    test_case "File System Permissions" validate_filesystem_permissions
    
    # Guild-specific requirements
    test_case "Guild Binary Available" validate_guild_binary
    test_case "Guild Configuration" validate_guild_configuration
    test_case "Provider Setup" validate_provider_setup
    test_case "Database Connectivity" validate_database_connectivity
    
    # Development environment
    test_case "Go Development Environment" validate_go_environment
    test_case "Build System" validate_build_system
    test_case "Test Framework" validate_test_framework
    
    log_section "Environment Validation Complete"
}

validate_environment_essential() {
    print_status "Validating essential demo environment..."
    
    test_case "Guild Binary" validate_guild_binary
    test_case "Recording Tools" validate_recording_tools_available
    test_case "Terminal Size" validate_terminal_size
    test_case "Basic Permissions" validate_basic_permissions
    
    log_section "Essential Environment Validation Complete"
}

validate_demo_scripts() {
    print_status "Validating demo scripts..."
    
    local scripts_dir="$SCRIPT_DIR"
    
    # Check script syntax and executability
    test_case "Script Syntax Check" validate_all_script_syntax
    test_case "Script Permissions" validate_script_permissions
    test_case "Script Dependencies" validate_script_dependencies
    
    # Test each demo script individually  
    test_case "Quick Start Demo Validation" validate_quick_start_demo
    test_case "Complete Workflow Demo Validation" validate_complete_workflow_demo
    test_case "Feature Showcase Demo Validation" validate_feature_showcase_demo
    test_case "Interactive Demo Validation" validate_interactive_demo
    
    # Test demo master orchestrator
    test_case "Demo Master Validation" validate_demo_master
    
    log_section "Demo Scripts Validation Complete"
}

validate_demo_scripts_syntax() {
    print_status "Validating demo script syntax..."
    
    test_case "Script Syntax" validate_all_script_syntax
    test_case "Script Executable" validate_script_permissions
    
    log_section "Demo Script Syntax Validation Complete"
}

validate_recording_tools() {
    print_status "Validating recording tools and capabilities..."
    
    test_case "Asciinema Installation" validate_asciinema
    test_case "AGG Installation" validate_agg
    test_case "FFmpeg Installation" validate_ffmpeg
    test_case "Gifsicle Installation" validate_gifsicle
    
    test_case "Recording Functionality" test_recording_functionality
    test_case "GIF Generation" test_gif_generation
    test_case "File Size Optimization" test_optimization_tools
    
    log_section "Recording Tools Validation Complete"
}

validate_recording_tools_available() {
    print_status "Checking recording tool availability..."
    
    test_case "Asciinema Available" validate_asciinema
    test_case "AGG Available" validate_agg
    
    log_section "Recording Tools Check Complete"
}

validate_demo_content() {
    print_status "Validating demo content and scenarios..."
    
    test_case "Demo Commission Files" validate_demo_commissions
    test_case "Example Configurations" validate_example_configs
    test_case "Demo Data Consistency" validate_demo_data
    test_case "Content Quality" validate_content_quality
    
    log_section "Demo Content Validation Complete"
}

validate_integration_tests() {
    print_status "Validating E2E integration tests..."
    
    if [[ -d "$E2E_TEST_DIR" ]]; then
        test_case "E2E Test Framework" validate_e2e_framework
        test_case "Demo E2E Tests" run_demo_e2e_tests
        test_case "Test Coverage" validate_test_coverage
    else
        log_warning "E2E test directory not found: $E2E_TEST_DIR"
    fi
    
    log_section "Integration Tests Validation Complete"
}

run_performance_checks() {
    print_status "Running performance validation..."
    
    test_case "Demo Startup Time" measure_demo_startup_time
    test_case "Recording Performance" measure_recording_performance
    test_case "GIF Generation Speed" measure_gif_generation_speed
    test_case "Memory Usage" measure_memory_usage
    
    log_section "Performance Validation Complete"
}

# Individual test implementations

validate_system_requirements() {
    local errors=0
    
    # Check OS
    case "$(uname -s)" in
        Darwin|Linux)
            log_info "Operating system: $(uname -s) - Supported"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            ((errors++))
            ;;
    esac
    
    # Check architecture
    case "$(uname -m)" in
        x86_64|arm64|aarch64)
            log_info "Architecture: $(uname -m) - Supported"
            ;;
        *)
            log_warning "Architecture $(uname -m) may not be fully supported"
            ;;
    esac
    
    return $errors
}

validate_terminal_capabilities() {
    local errors=0
    
    # Check color support
    if [[ -n "$COLORTERM" ]]; then
        log_info "Color terminal support: $COLORTERM"
    else
        log_warning "COLORTERM not set - colors may not work properly"
    fi
    
    # Check Unicode support
    if echo "🏰🤖✅" | cat > /dev/null 2>&1; then
        log_info "Unicode support: Available"
    else
        log_warning "Unicode support may be limited"
    fi
    
    return $errors
}

validate_guild_binary() {
    if command -v "$GUILD_BIN" &> /dev/null; then
        local version=$($GUILD_BIN version 2>/dev/null || echo "unknown")
        log_info "Guild binary found: $GUILD_BIN (version: $version)"
        return 0
    else
        log_error "Guild binary not found: $GUILD_BIN"
        return 1
    fi
}

validate_terminal_size() {
    local cols=$(tput cols 2>/dev/null || echo "80")
    local rows=$(tput lines 2>/dev/null || echo "24")
    
    if [[ $cols -ge 120 && $rows -ge 35 ]]; then
        log_info "Terminal size: ${cols}x${rows} - Optimal for recording"
        return 0
    elif [[ $cols -ge 100 && $rows -ge 30 ]]; then
        log_warning "Terminal size: ${cols}x${rows} - Acceptable but not optimal"
        return 0
    else
        log_error "Terminal size: ${cols}x${rows} - Too small for professional recording"
        return 1
    fi
}

validate_all_script_syntax() {
    local errors=0
    
    for script in "$SCRIPT_DIR"/*.sh "$SCRIPT_DIR"/**/*.sh; do
        if [[ -f "$script" ]]; then
            if bash -n "$script" 2>/dev/null; then
                log_info "Syntax OK: $(basename "$script")"
            else
                log_error "Syntax error in: $(basename "$script")"
                ((errors++))
            fi
        fi
    done
    
    return $errors
}

validate_asciinema() {
    if command -v asciinema &> /dev/null; then
        local version=$(asciinema --version 2>/dev/null | head -1)
        log_info "Asciinema available: $version"
        return 0
    else
        log_error "Asciinema not found - required for recording"
        return 1
    fi
}

validate_agg() {
    if command -v agg &> /dev/null; then
        log_info "AGG available for GIF generation"
        return 0
    else
        log_warning "AGG not found - GIF generation will be skipped"
        return 1
    fi
}

validate_quick_start_demo() {
    local script="$SCRIPT_DIR/01-quick-start-demo.sh"
    
    if [[ -f "$script" ]]; then
        if "$script" validate &> /dev/null; then
            log_info "Quick start demo validation passed"
            return 0
        else
            log_error "Quick start demo validation failed"
            return 1
        fi
    else
        log_error "Quick start demo script not found"
        return 1
    fi
}

# Test case runner
test_case() {
    local name="$1"
    local test_function="$2"
    
    ((TOTAL_TESTS++))
    
    print_info "Testing: $name"
    
    if $test_function; then
        print_success "✅ $name - PASSED"
        ((PASSED_TESTS++))
        log_test_result "PASS" "$name"
    else
        print_error "❌ $name - FAILED"
        ((FAILED_TESTS++))
        VALIDATION_ERRORS+=("$name")
        log_test_result "FAIL" "$name"
    fi
}

# Logging functions
log_test_result() {
    local result="$1"
    local test_name="$2"
    
    {
        echo "[$result] $test_name"
    } >> "$VALIDATION_LOG"
}

log_section() {
    local section="$1"
    
    {
        echo
        echo "=== $section ==="
        echo
    } >> "$VALIDATION_LOG"
}

log_info() {
    echo "  INFO: $1" >> "$VALIDATION_LOG"
}

log_warning() {
    echo "  WARN: $1" >> "$VALIDATION_LOG"
}

log_error() {
    echo "  ERROR: $1" >> "$VALIDATION_LOG"
}

generate_validation_report() {
    print_title "Demo Validation Results"
    echo
    
    print_info "Test Summary:"
    echo "  Total Tests: $TOTAL_TESTS"
    echo "  Passed: $PASSED_TESTS"
    echo "  Failed: $FAILED_TESTS"
    echo "  Success Rate: $(( (PASSED_TESTS * 100) / TOTAL_TESTS ))%"
    echo
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        print_success "🎉 All validation tests passed!"
        print_info "Demo environment is ready for professional recording"
        
        {
            echo
            echo "VALIDATION RESULT: SUCCESS"
            echo "All $TOTAL_TESTS tests passed"
            echo "Environment ready for demo recording"
        } >> "$VALIDATION_LOG"
        
    else
        print_error "⚠️ Some validation tests failed"
        print_info "Failed tests:"
        for error in "${VALIDATION_ERRORS[@]}"; do
            echo "  • $error"
        done
        
        {
            echo
            echo "VALIDATION RESULT: FAILURE"
            echo "$FAILED_TESTS of $TOTAL_TESTS tests failed"
            echo "Issues must be resolved before recording"
        } >> "$VALIDATION_LOG"
        
        echo
        print_info "Please review the validation log: $VALIDATION_LOG"
        
        return 1
    fi
    
    print_info "Full validation log saved to: $VALIDATION_LOG"
}

generate_quick_report() {
    print_title "Quick Validation Results"
    echo
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        print_success "✅ Quick validation passed ($PASSED_TESTS/$TOTAL_TESTS tests)"
        print_info "Ready for demo recording"
    else
        print_error "❌ Quick validation failed ($FAILED_TESTS failures)"
        print_info "Fix critical issues before recording"
        return 1
    fi
}

run_ci_validation() {
    print_title "CI/CD Demo Validation"
    print_info "Automated validation for continuous integration"
    echo
    
    # CI-specific validations (non-interactive)
    export GUILD_MOCK_PROVIDER=true
    export NO_COLOR=1
    
    validate_environment_essential
    validate_demo_scripts_syntax
    
    # Generate machine-readable output
    local exit_code=0
    if [[ $FAILED_TESTS -gt 0 ]]; then
        exit_code=1
    fi
    
    {
        echo "CI_VALIDATION_RESULT=$([[ $exit_code -eq 0 ]] && echo "SUCCESS" || echo "FAILURE")"
        echo "CI_TOTAL_TESTS=$TOTAL_TESTS"
        echo "CI_PASSED_TESTS=$PASSED_TESTS"
        echo "CI_FAILED_TESTS=$FAILED_TESTS"
    } > "/tmp/guild-demo-validation-ci.env"
    
    return $exit_code
}

show_validation_help() {
    cat << 'EOF'
Guild Demo Validation System

USAGE:
    validate-demos.sh [MODE]

VALIDATION MODES:
    quick           Essential checks for demo readiness
    full            Comprehensive validation of all components  
    environment     Validate environment setup only
    scripts         Validate demo scripts only
    performance     Run performance benchmarks
    ci              CI/CD compatible validation

VALIDATION CATEGORIES:
    • System requirements and compatibility
    • Terminal capabilities and configuration
    • Guild framework installation and setup
    • Demo script syntax and functionality
    • Recording tools and dependencies
    • Demo content and data consistency
    • E2E integration tests
    • Performance benchmarks

OUTPUT:
    Validation results are logged to /tmp/guild-demo-validation.log
    CI mode generates machine-readable output in /tmp/guild-demo-validation-ci.env

EXAMPLES:
    # Quick check before recording
    ./validate-demos.sh quick

    # Full validation before release
    ./validate-demos.sh full

    # CI pipeline integration
    ./validate-demos.sh ci && source /tmp/guild-demo-validation-ci.env

For more information about Guild Framework demos:
    ./demo-master.sh help
EOF
}

# Handle command line arguments
main "$@"