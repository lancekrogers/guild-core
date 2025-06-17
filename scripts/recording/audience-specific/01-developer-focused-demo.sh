#!/bin/bash
# Developer-Focused Demo - Technical audience demonstration
# Shows Guild's technical architecture and developer experience

source "$(dirname "$0")/../lib/recording-utils.sh"

# Demo configuration
DEMO_NAME="Developer-Focused Demo"
DEMO_DESCRIPTION="Technical demonstration for developers and engineers"
RECORDING_NAME="guild-developer-demo"

main() {
    local mode="${1:-interactive}"
    
    recording_init "$RECORDING_NAME"
    
    if [[ "$mode" == "record" ]]; then
        {
            run_developer_demo
        } | start_recording "$RECORDING_NAME" "Guild Framework - Developer Demo"
        
        generate_gif "$RECORD_FILE" "${RECORDING_DIR}/guild-developer-demo.gif"
        
        echo
        print_success "Developer-focused demo recorded successfully!"
        
    elif [[ "$mode" == "validate" ]]; then
        validate_recording_environment
        check_guild_providers
        print_success "Developer demo validation passed"
        
    else
        show_demo_preview
        echo
        read -p "Record this demo? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            exec "$0" record
        fi
    fi
}

run_developer_demo() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    # Step 1: Architecture Overview (30 seconds)
    print_title "Guild Framework Architecture"
    run_demo_command "guild init --name 'arch-demo' --description 'Architecture demonstration'" \
        "Initialize Guild with clean project structure"
    
    run_demo_command "find .guild -name '*.db' -o -name '*.yaml' -o -name '*.sql'" \
        "Explore Guild's internal architecture"
    
    run_demo_command "guild config show --verbose" \
        "Show detailed configuration and provider setup"
    
    # Step 2: Agent System Deep Dive (45 seconds)
    print_title "Multi-Agent System Architecture"
    run_demo_command "guild agents list --technical" \
        "Show agent implementations and capabilities"
    
    run_demo_command "guild agents inspect --agent manager" \
        "Deep dive into manager agent architecture"
    
    # Step 3: Commission Processing Pipeline (60 seconds)
    print_title "Commission Processing Pipeline"
    create_demo_commission "technical-commission.md" "Microservices Architecture"
    
    # Add technical details to commission
    cat >> technical-commission.md << 'EOF'

## Technical Architecture
- Go microservices with gRPC communication
- PostgreSQL with event sourcing
- Redis for caching and session management
- Docker containerization
- Kubernetes orchestration
- Prometheus monitoring
- Jaeger distributed tracing

## Performance Requirements
- 99.9% uptime SLA
- Sub-100ms response times
- Support for 10,000 concurrent users
- Horizontal scaling capability

## Security Requirements
- OAuth 2.0 + OIDC authentication
- mTLS for service communication
- Vault for secrets management
- RBAC authorization model
EOF

    run_demo_command "guild commission analyze technical-commission.md --technical" \
        "Show technical analysis and complexity assessment"
    
    run_demo_command "guild commission refine technical-commission.md --output refined-tech.md" \
        "AI-powered commission refinement with technical focus"
    
    # Step 4: Real-time Development Workflow (50 seconds)
    print_title "Developer Workflow Integration"
    run_demo_command "guild commission -f technical-commission.md --debug" \
        "Submit commission with debug output showing internal processes"
    
    run_demo_command "guild status --json" \
        "Machine-readable status for IDE integration"
    
    run_demo_command "guild logs --level debug --agent manager" \
        "Inspect agent decision-making process"
    
    # Step 5: Code Generation and Quality (45 seconds)
    print_title "Professional Code Generation"
    run_demo_command "guild workspace scan --metrics" \
        "Analyze generated code structure and metrics"
    
    run_demo_command "guild code review --automated --format json" \
        "Automated code review with detailed analysis"
    
    run_demo_command "guild test coverage --threshold 85" \
        "Test coverage analysis and quality gates"
    
    # Step 6: Integration and Extensibility (40 seconds)
    print_title "Developer Experience & Integration"
    run_demo_command "guild tools list --category development" \
        "Show available development tools and integrations"
    
    run_demo_command "guild api status" \
        "gRPC API status for IDE plugins and external tools"
    
    run_demo_command "guild corpus query 'authentication patterns'" \
        "Semantic code search across project knowledge base"
    
    # Step 7: Performance and Monitoring (25 seconds)
    print_title "Performance and Observability"
    run_demo_command "guild metrics performance" \
        "Show Guild performance metrics and optimization"
    
    run_demo_command "guild memory usage --detailed" \
        "Memory usage analysis for resource optimization"
    
    # Technical Summary
    echo
    print_title "🔧 Technical Advantages for Developers"
    echo
    print_success "Architecture Benefits:"
    echo "  ✅ Multi-agent coordination eliminates context switching"
    echo "  ✅ Event-driven architecture enables real-time collaboration"
    echo "  ✅ SQLite + vector search for intelligent project memory"
    echo "  ✅ gRPC APIs for high-performance tool integration"
    echo "  ✅ Plugin architecture for extensibility"
    echo
    print_success "Developer Experience:"
    echo "  ✅ Rich TUI with vim-style navigation"
    echo "  ✅ Comprehensive CLI with machine-readable output"
    echo "  ✅ Real-time status updates and progress tracking"
    echo "  ✅ Intelligent code completion and suggestions"
    echo "  ✅ Integrated testing and quality assurance"
    echo
    print_success "Integration Capabilities:"
    echo "  ✅ IDE plugins via Language Server Protocol"
    echo "  ✅ CI/CD pipeline integration"
    echo "  ✅ Git workflow integration"
    echo "  ✅ External tool ecosystem support"
    echo "  ✅ Custom agent development framework"
    echo
    print_title "Built by developers, for developers 🚀"
    
    sleep 3
}

show_demo_preview() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    print_info "This technical demo showcases:"
    echo "  • Guild's multi-agent architecture and implementation details"
    echo "  • Commission processing pipeline with technical analysis"
    echo "  • Developer workflow integration and tooling support"
    echo "  • Professional code generation with quality assurance"
    echo "  • API integration capabilities for IDE and external tools"
    echo "  • Performance characteristics and observability features"
    echo
    print_warning "Target Audience: Developers, Technical Leaders, Architects"
    print_info "Focus: Technical depth, architecture, developer experience"
    echo
    print_title "Technical Highlights:"
    echo "  • Multi-agent system implementation"
    echo "  • Event-driven architecture patterns"
    echo "  • Real-time collaboration mechanisms"
    echo "  • Performance optimization strategies"
    echo "  • Extensibility and integration APIs"
}

# Handle command line arguments
case "${1:-}" in
    "record")
        main record
        ;;
    "validate")
        main validate
        ;;
    "preview")
        show_demo_preview
        ;;
    "help"|"--help"|"-h")
        echo "Guild Developer-Focused Demo"
        echo ""
        echo "Technical demonstration for developers and engineers"
        echo "Shows architecture, implementation details, and developer experience"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  record      Record the demo automatically"
        echo "  validate    Validate environment for recording"
        echo "  preview     Show what will be recorded"
        echo "  help        Show this help message"
        ;;
    *)
        main interactive
        ;;
esac