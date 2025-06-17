#!/bin/bash
# Feature Showcase Demo - 2 minute focused feature demonstration
# Highlights Guild's unique capabilities and competitive advantages

source "$(dirname "$0")/lib/recording-utils.sh"

# Demo configuration
DEMO_NAME="Feature Showcase"
DEMO_DESCRIPTION="2-minute focused demonstration of Guild's unique capabilities"
RECORDING_NAME="guild-feature-showcase"

main() {
    local mode="${1:-interactive}"
    
    # Initialize recording environment
    recording_init "$RECORDING_NAME"
    
    if [[ "$mode" == "record" ]]; then
        # Start recording and run demo automatically
        {
            run_feature_showcase_demo
        } | start_recording "$RECORDING_NAME" "Guild Framework - Feature Showcase Demo"
        
        # Generate GIF
        generate_gif "$RECORD_FILE" "${RECORDING_DIR}/guild-feature-showcase.gif"
        
        echo
        print_success "Feature showcase demo recorded successfully!"
        print_info "Files created:"
        print_info "  - Recording: $RECORD_FILE"
        print_info "  - GIF: ${RECORDING_DIR}/guild-feature-showcase.gif"
        
    elif [[ "$mode" == "validate" ]]; then
        # Validate the demo can run
        validate_recording_environment
        check_guild_providers
        print_success "Feature showcase demo validation passed"
        
    else
        # Interactive mode - show what will be recorded
        show_demo_preview
        echo
        read -p "Record this demo? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            exec "$0" record
        fi
    fi
}

run_feature_showcase_demo() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    # Feature 1: Smart Commission Processing (25 seconds)
    print_title "🧠 Feature 1: AI-Powered Commission Intelligence"
    
    create_demo_commission "showcase-commission.md" "Microservices Architecture"
    
    run_demo_command "guild commission analyze showcase-commission.md" \
        "AI analyzes commission complexity and requirements"
    
    run_demo_command "guild commission breakdown showcase-commission.md" \
        "Intelligent task breakdown with agent assignments"
    
    # Feature 2: Multi-Agent Orchestration (30 seconds)
    print_title "🤖 Feature 2: Multi-Agent Development Orchestration"
    
    run_demo_command "guild agents status --live" \
        "Real-time multi-agent coordination display"
    
    run_demo_command "guild orchestrator plan --commission showcase-commission.md" \
        "Show orchestration plan with dependencies"
    
    run_demo_command "guild workshop --view kanban" \
        "Visual task management with agent assignments"
    
    # Feature 3: Interactive Development Chat (25 seconds)
    print_title "💬 Feature 3: Intelligent Development Chat"
    
    print_info "Demonstrating Guild's revolutionary chat interface..."
    echo "guild chat --enhanced"
    sleep 1
    
    type_command "/agents" "guild> "
    print_info "Rich agent directory with capabilities and status"
    sleep 2
    
    type_command "@manager /task-complexity microservices" "guild> "
    print_info "AI-powered task complexity analysis with recommendations"
    sleep 2
    
    type_command "/tools search 'authentication'" "guild> "
    print_info "Intelligent tool discovery and suggestions"
    sleep 2
    
    # Feature 4: Professional Code Generation (30 seconds)
    print_title "⚡ Feature 4: Context-Aware Code Generation"
    
    run_demo_command "guild generate --type api --spec showcase-commission.md" \
        "Generate production-ready API code from commission"
    
    run_demo_command "guild code analyze --quality" \
        "Automated code quality analysis and suggestions"
    
    run_demo_command "guild test generate --coverage 90" \
        "Generate comprehensive test suites with high coverage"
    
    # Feature 5: Rich Visual Interface (20 seconds)
    print_title "🎨 Feature 5: Professional Visual Experience"
    
    run_demo_command "guild status --visual" \
        "Beautiful visual status displays with progress indicators"
    
    run_demo_command "guild docs preview --live" \
        "Live documentation preview with rich formatting"
    
    # Feature 6: Enterprise Integration (25 seconds)
    print_title "🏢 Feature 6: Enterprise-Ready Integration"
    
    run_demo_command "guild corpus scan --project ." \
        "Intelligent codebase analysis and indexing"
    
    run_demo_command "guild memory search 'authentication patterns'" \
        "Semantic search through project knowledge base"
    
    run_demo_command "guild export --format enterprise" \
        "Professional project export with documentation"
    
    # Competitive comparison summary
    echo
    print_title "🏆 Guild vs. Traditional Development Tools"
    echo
    print_success "Guild Framework Features:"
    echo "  ✅ Multi-agent AI coordination"
    echo "  ✅ Intelligent commission processing"
    echo "  ✅ Context-aware code generation"
    echo "  ✅ Professional visual interface"
    echo "  ✅ Enterprise-ready integration"
    echo "  ✅ Interactive development chat"
    echo
    print_warning "Traditional Tools:"
    echo "  ❌ Manual coordination required"
    echo "  ❌ Basic prompt interfaces"
    echo "  ❌ Limited context awareness"
    echo "  ❌ Plain text interactions"
    echo "  ❌ Complex setup and configuration"
    echo "  ❌ Isolated tool usage"
    echo
    print_title "The Choice is Clear: Guild Framework Leads the Future!"
    
    sleep 3
}

show_demo_preview() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    print_info "This feature showcase demonstrates:"
    echo "  • AI-powered commission intelligence and analysis"
    echo "  • Multi-agent development orchestration"
    echo "  • Interactive development chat with rich features"
    echo "  • Context-aware professional code generation"
    echo "  • Beautiful visual interface and user experience"
    echo "  • Enterprise-ready integration capabilities"
    echo
    print_warning "Duration: ~2 minutes"
    print_info "Perfect for highlighting competitive advantages"
    echo
    print_title "Feature Breakdown:"
    echo "  1. Smart Commission Processing (25s) - AI intelligence"
    echo "  2. Multi-Agent Orchestration (30s) - Coordination power"
    echo "  3. Interactive Development Chat (25s) - Rich interface"
    echo "  4. Code Generation (30s) - Context-aware output"
    echo "  5. Visual Experience (20s) - Professional polish"
    echo "  6. Enterprise Integration (25s) - Business ready"
    echo
    print_success "Showcases clear superiority over traditional AI coding tools"
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
        echo "Guild Feature Showcase Demo"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  record      Record the demo automatically"
        echo "  validate    Validate environment for recording"
        echo "  preview     Show what will be recorded"
        echo "  help        Show this help message"
        echo ""
        echo "Interactive mode will start if no command provided."
        ;;
    *)
        main interactive
        ;;
esac