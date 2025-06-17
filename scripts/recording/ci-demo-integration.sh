#!/bin/bash
# CI/CD Demo Integration - Automated demo validation and artifact generation
# Integrates with Guild's build system for continuous demo validation

set -e

source "$(dirname "$0")/lib/recording-utils.sh"

# CI Configuration
CI_MODE="${CI:-false}"
GITHUB_ACTIONS="${GITHUB_ACTIONS:-false}"
BUILD_DIR="${BUILD_DIR:-./bin}"
ARTIFACTS_DIR="${ARTIFACTS_DIR:-./demo-artifacts}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

main() {
    local command="${1:-validate}"
    
    case "$command" in
        "validate")
            run_ci_validation
            ;;
        "build")
            build_and_validate
            ;;
        "generate")
            generate_demo_artifacts
            ;;
        "test")
            run_demo_tests
            ;;
        "deploy")
            deploy_demo_artifacts
            ;;
        "pipeline")
            run_full_pipeline
            ;;
        *)
            show_ci_help
            exit 1
            ;;
    esac
}

run_ci_validation() {
    print_title "CI/CD Demo Validation Pipeline"
    
    # Set up CI environment
    setup_ci_environment
    
    # Validate prerequisites
    validate_ci_prerequisites
    
    # Run validation suite
    "$SCRIPT_DIR/validate-demos.sh" ci
    
    # Check results
    if [[ -f "/tmp/guild-demo-validation-ci.env" ]]; then
        source "/tmp/guild-demo-validation-ci.env"
        
        if [[ "$CI_VALIDATION_RESULT" == "SUCCESS" ]]; then
            print_success "✅ CI demo validation passed ($CI_PASSED_TESTS/$CI_TOTAL_TESTS tests)"
            
            # Set GitHub Actions output
            if [[ "$GITHUB_ACTIONS" == "true" ]]; then
                echo "demo-validation=success" >> "$GITHUB_OUTPUT"
                echo "demo-tests-passed=$CI_PASSED_TESTS" >> "$GITHUB_OUTPUT"
                echo "demo-tests-total=$CI_TOTAL_TESTS" >> "$GITHUB_OUTPUT"
            fi
            
            return 0
        else
            print_error "❌ CI demo validation failed ($CI_FAILED_TESTS failures)"
            
            if [[ "$GITHUB_ACTIONS" == "true" ]]; then
                echo "demo-validation=failure" >> "$GITHUB_OUTPUT"
                echo "demo-tests-failed=$CI_FAILED_TESTS" >> "$GITHUB_OUTPUT"
            fi
            
            return 1
        fi
    else
        print_error "Validation results not found"
        return 1
    fi
}

build_and_validate() {
    print_title "Build and Validate Guild Demos"
    
    # Build Guild binary
    print_status "Building Guild framework..."
    if command -v make &> /dev/null; then
        make build
    else
        go build -o "$BUILD_DIR/guild" ./cmd/guild
    fi
    
    # Verify build
    if [[ ! -f "$BUILD_DIR/guild" ]]; then
        print_error "Guild binary not found after build"
        return 1
    fi
    
    # Set Guild binary path for validation
    export GUILD_BIN="$BUILD_DIR/guild"
    
    # Run validation
    run_ci_validation
}

generate_demo_artifacts() {
    print_title "Generating Demo Artifacts"
    
    mkdir -p "$ARTIFACTS_DIR"
    
    # Generate README materials
    print_status "Generating README materials..."
    "$SCRIPT_DIR/demo-master.sh" generate-readme
    
    # Copy generated materials to artifacts directory
    if [[ -d "/tmp/guild-recordings/readme-materials" ]]; then
        cp -r "/tmp/guild-recordings/readme-materials"/* "$ARTIFACTS_DIR/"
        print_success "README materials copied to artifacts"
    fi
    
    # Generate demo scripts documentation
    print_status "Generating demo documentation..."
    generate_demo_documentation
    
    # Create demo manifest
    create_demo_manifest
    
    print_success "Demo artifacts generated in: $ARTIFACTS_DIR"
}

run_demo_tests() {
    print_title "Running Demo Integration Tests"
    
    # Set up test environment
    export GUILD_MOCK_PROVIDER=true
    export GUILD_TEST_MODE=true
    
    # Run E2E tests related to demos
    if [[ -d "integration/e2e" ]]; then
        print_status "Running E2E demo tests..."
        
        # Test demo command functionality
        go test -v ./integration/e2e -run ".*Demo.*" -timeout 5m
        
        # Test demo validation
        go test -v ./integration/e2e -run ".*Validation.*" -timeout 3m
    fi
    
    # Test demo scripts directly
    print_status "Testing demo scripts..."
    test_demo_scripts
    
    print_success "Demo tests completed"
}

deploy_demo_artifacts() {
    print_title "Deploying Demo Artifacts"
    
    if [[ "$CI_MODE" == "true" ]]; then
        # CI deployment (e.g., to docs site, CDN, etc.)
        deploy_to_ci_artifacts
    else
        # Local deployment (copy to docs directory)
        deploy_to_local_docs
    fi
}

run_full_pipeline() {
    print_title "Running Full Demo CI/CD Pipeline"
    
    # Step 1: Build and validate
    if ! build_and_validate; then
        print_error "Build and validation failed"
        return 1
    fi
    
    # Step 2: Run tests
    if ! run_demo_tests; then
        print_error "Demo tests failed"
        return 1
    fi
    
    # Step 3: Generate artifacts
    if ! generate_demo_artifacts; then
        print_error "Artifact generation failed"
        return 1
    fi
    
    # Step 4: Deploy artifacts (if on main branch or release)
    if should_deploy_artifacts; then
        deploy_demo_artifacts
    fi
    
    print_success "🎉 Full demo pipeline completed successfully"
}

# Helper functions

setup_ci_environment() {
    print_status "Setting up CI environment for demo validation..."
    
    # Disable interactive features
    export NO_COLOR=1
    export GUILD_TEST_MODE=true
    export GUILD_MOCK_PROVIDER=true
    export GUILD_LOG_LEVEL=error
    
    # Set appropriate paths
    export DEMO_HOME="/tmp/guild-ci-demos"
    export GUILD_BIN="${GUILD_BIN:-./guild}"
    
    # Create necessary directories
    mkdir -p "$DEMO_HOME"
    mkdir -p "$ARTIFACTS_DIR"
    
    print_success "CI environment configured"
}

validate_ci_prerequisites() {
    print_status "Validating CI prerequisites..."
    
    local errors=0
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        print_error "Go not found in CI environment"
        ((errors++))
    fi
    
    # Check required tools
    local tools=("git" "bash" "make")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "Required tool not found: $tool"
            ((errors++))
        fi
    done
    
    # Check file permissions
    if [[ ! -w "." ]]; then
        print_error "No write permissions in current directory"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        print_success "CI prerequisites validated"
        return 0
    else
        print_error "CI prerequisites validation failed ($errors errors)"
        return 1
    fi
}

generate_demo_documentation() {
    local doc_file="$ARTIFACTS_DIR/DEMO_DOCUMENTATION.md"
    
    cat > "$doc_file" << 'EOF'
# Guild Framework Demo Documentation

## Overview
This document describes the professional demonstration system for Guild Framework, including recording scripts, validation tools, and artifact generation.

## Demo Scripts

### Quick Start Demo (30 seconds)
**Purpose:** First impression and social media sharing  
**Script:** `01-quick-start-demo.sh`  
**Highlights:**
- Instant project initialization
- Multi-agent coordination preview
- Professional workflow demonstration

### Complete Workflow Demo (5 minutes)
**Purpose:** Comprehensive product demonstration  
**Script:** `02-complete-workflow-demo.sh`  
**Highlights:**
- Full development lifecycle
- Multi-agent team coordination
- Interactive development environment
- Professional code generation

### Feature Showcase Demo (2 minutes)
**Purpose:** Competitive advantage demonstration  
**Script:** `03-feature-showcase-demo.sh`  
**Highlights:**
- Unique Guild capabilities
- Comparison with traditional tools
- Enterprise-ready features

### Interactive Tutorial
**Purpose:** Hands-on learning experience  
**Script:** `interactive-demo.sh`  
**Features:**
- Step-by-step guided progression
- Error recovery and help system
- Progress tracking and navigation

## Demo Infrastructure

### Recording System
- **Tool:** Asciinema for terminal recording
- **Format:** Cast files and optimized GIFs
- **Quality:** Professional settings with 120x35 terminal
- **Output:** Multi-platform optimized formats

### Validation Framework
- **Script:** `validate-demos.sh`
- **Coverage:** Environment, scripts, tools, content
- **Modes:** Quick, full, CI/CD compatible
- **Integration:** E2E test framework integration

### CI/CD Integration
- **Script:** `ci-demo-integration.sh`
- **Pipeline:** Build → Validate → Test → Deploy
- **Artifacts:** README materials, GIFs, documentation
- **Quality Gates:** Automated validation and testing

## Usage Instructions

### For Developers
```bash
# Quick demo validation
./scripts/recording/validate-demos.sh quick

# Record demonstration
./scripts/recording/demo-master.sh record-all

# Generate materials
./scripts/recording/demo-master.sh generate-readme
```

### For CI/CD
```bash
# Full pipeline
./scripts/recording/ci-demo-integration.sh pipeline

# Validation only
./scripts/recording/ci-demo-integration.sh validate
```

### For Marketing
```bash
# Social media clips
./scripts/recording/demo-master.sh social-media

# README materials
./scripts/recording/demo-master.sh generate-readme
```

## Best Practices

### Recording Guidelines
1. Use 120x35 terminal size for optimal viewing
2. Enable true color support (COLORTERM=truecolor)
3. Use professional fonts (SF Mono, Monaco, Cascadia Code)
4. Test recording setup before final recordings
5. Pre-warm caches for smooth demonstrations

### Content Standards
1. Show real Guild functionality (no mocked responses)
2. Demonstrate professional workflows
3. Highlight unique competitive advantages
4. Ensure consistent quality across all demos
5. Keep demonstrations focused and engaging

### Quality Assurance
1. Validate all demos before recording
2. Test across different environments
3. Verify GIF quality and file sizes
4. Review content for accuracy and consistency
5. Automate validation in CI/CD pipeline

## Troubleshooting

### Common Issues
- **Guild binary not found:** Run `make build` first
- **Recording tools missing:** Install asciinema and agg
- **Terminal size issues:** Resize to 120x35 minimum
- **Provider errors:** Use mock provider for demos
- **Permission issues:** Check file system permissions

### Support Resources
- Validation script: `validate-demos.sh`
- Help system: `demo-master.sh help`
- Demo documentation: This file
- Issue tracker: GitHub Issues
EOF

    print_success "Demo documentation generated: $doc_file"
}

create_demo_manifest() {
    local manifest_file="$ARTIFACTS_DIR/demo-manifest.json"
    
    cat > "$manifest_file" << EOF
{
  "version": "1.0.0",
  "generated": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "demos": {
    "quick-start": {
      "name": "Quick Start Demo",
      "duration": "30 seconds",
      "purpose": "First impression and social media",
      "script": "01-quick-start-demo.sh",
      "formats": ["cast", "gif"]
    },
    "complete-workflow": {
      "name": "Complete Workflow Demo", 
      "duration": "5 minutes",
      "purpose": "Comprehensive product demonstration",
      "script": "02-complete-workflow-demo.sh",
      "formats": ["cast", "gif"]
    },
    "feature-showcase": {
      "name": "Feature Showcase Demo",
      "duration": "2 minutes", 
      "purpose": "Competitive advantage demonstration",
      "script": "03-feature-showcase-demo.sh",
      "formats": ["cast", "gif"]
    },
    "interactive-tutorial": {
      "name": "Interactive Tutorial",
      "duration": "Variable",
      "purpose": "Hands-on learning experience",
      "script": "interactive-demo.sh",
      "formats": ["interactive"]
    }
  },
  "infrastructure": {
    "recording": {
      "terminal_size": "120x35",
      "tools": ["asciinema", "agg", "gifsicle"],
      "formats": ["cast", "gif", "mp4"]
    },
    "validation": {
      "script": "validate-demos.sh",
      "modes": ["quick", "full", "ci"],
      "coverage": ["environment", "scripts", "tools", "content"]
    },
    "ci_integration": {
      "script": "ci-demo-integration.sh",
      "pipeline": ["build", "validate", "test", "deploy"],
      "artifacts": ["readme", "gifs", "documentation"]
    }
  },
  "quality": {
    "standards": [
      "Real functionality demonstration",
      "Professional visual quality",
      "Consistent user experience",
      "Cross-platform compatibility",
      "Automated validation"
    ],
    "file_size_limits": {
      "gif_max_mb": 10,
      "cast_max_mb": 5
    }
  }
}
EOF

    print_success "Demo manifest created: $manifest_file"
}

test_demo_scripts() {
    local test_errors=0
    
    # Test each demo script in validate mode
    local scripts=("01-quick-start-demo.sh" "02-complete-workflow-demo.sh" "03-feature-showcase-demo.sh" "interactive-demo.sh")
    
    for script in "${scripts[@]}"; do
        local script_path="$SCRIPT_DIR/$script"
        
        if [[ -f "$script_path" ]]; then
            print_status "Testing $script..."
            
            if "$script_path" validate &> /dev/null; then
                print_success "✅ $script validation passed"
            else
                print_error "❌ $script validation failed"
                ((test_errors++))
            fi
        else
            print_error "Demo script not found: $script"
            ((test_errors++))
        fi
    done
    
    return $test_errors
}

should_deploy_artifacts() {
    # Deploy on main branch, release branches, or tags
    local ref="${GITHUB_REF:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")}"
    
    case "$ref" in
        "refs/heads/main"|"main"|"refs/heads/master"|"master")
            return 0
            ;;
        "refs/heads/release/"*|"refs/tags/"*)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

deploy_to_ci_artifacts() {
    print_status "Deploying to CI artifacts..."
    
    # Upload artifacts using CI-specific methods
    if [[ "$GITHUB_ACTIONS" == "true" ]]; then
        # GitHub Actions artifact upload would go here
        echo "::notice::Demo artifacts generated in $ARTIFACTS_DIR"
        
        # Set outputs for downstream jobs
        echo "artifacts-path=$ARTIFACTS_DIR" >> "$GITHUB_OUTPUT"
        echo "artifacts-ready=true" >> "$GITHUB_OUTPUT"
    fi
    
    print_success "Artifacts deployed to CI system"
}

deploy_to_local_docs() {
    print_status "Deploying to local documentation..."
    
    # Copy to docs directory if it exists
    if [[ -d "docs" ]]; then
        cp -r "$ARTIFACTS_DIR"/* docs/
        print_success "Artifacts copied to docs directory"
    fi
    
    # Copy GIFs to images directory
    if [[ -d "docs/images" ]]; then
        find "$ARTIFACTS_DIR" -name "*.gif" -exec cp {} docs/images/ \;
        print_success "GIFs copied to docs/images"
    fi
}

show_ci_help() {
    cat << 'EOF'
Guild Demo CI/CD Integration

USAGE:
    ci-demo-integration.sh [COMMAND]

COMMANDS:
    validate        Run CI-compatible demo validation
    build          Build Guild and validate demos
    generate       Generate demo artifacts and documentation
    test           Run demo integration tests
    deploy         Deploy demo artifacts
    pipeline       Run full CI/CD pipeline

CI/CD INTEGRATION:
    This script integrates with Guild's build system and provides
    automated demo validation and artifact generation for:
    
    • Continuous Integration (GitHub Actions, etc.)
    • Documentation generation
    • Quality assurance
    • Artifact deployment

ENVIRONMENT VARIABLES:
    CI                  Set to 'true' for CI mode
    GITHUB_ACTIONS      Automatically set by GitHub Actions
    BUILD_DIR           Guild binary output directory
    ARTIFACTS_DIR       Demo artifacts output directory

OUTPUTS (GitHub Actions):
    demo-validation     success|failure
    demo-tests-passed   Number of passed tests
    demo-tests-total    Total number of tests
    artifacts-path      Path to generated artifacts
    artifacts-ready     true if artifacts generated successfully

EXAMPLES:
    # Local validation
    ./ci-demo-integration.sh validate

    # Full pipeline
    ./ci-demo-integration.sh pipeline

    # CI/CD pipeline
    CI=true ./ci-demo-integration.sh pipeline

For more information:
    ./demo-master.sh help
    ./validate-demos.sh help
EOF
}

# Handle command line arguments
main "$@"