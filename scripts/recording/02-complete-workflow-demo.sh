#!/bin/bash
# Complete Workflow Demo - 5 minute comprehensive demonstration
# Shows full Guild development lifecycle from commission to completion

source "$(dirname "$0")/lib/recording-utils.sh"

# Demo configuration
DEMO_NAME="Complete Development Workflow"
DEMO_DESCRIPTION="5-minute comprehensive demonstration of Guild's full capabilities"
RECORDING_NAME="guild-complete-workflow"

main() {
    local mode="${1:-interactive}"
    
    # Initialize recording environment
    recording_init "$RECORDING_NAME"
    
    if [[ "$mode" == "record" ]]; then
        # Start recording and run demo automatically
        {
            run_complete_workflow_demo
        } | start_recording "$RECORDING_NAME" "Guild Framework - Complete Workflow Demo"
        
        # Generate GIF
        generate_gif "$RECORD_FILE" "${RECORDING_DIR}/guild-complete-workflow.gif"
        
        echo
        print_success "Complete workflow demo recorded successfully!"
        print_info "Files created:"
        print_info "  - Recording: $RECORD_FILE"
        print_info "  - GIF: ${RECORDING_DIR}/guild-complete-workflow.gif"
        
    elif [[ "$mode" == "validate" ]]; then
        # Validate the demo can run
        validate_recording_environment
        check_guild_providers
        print_success "Complete workflow demo validation passed"
        
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

run_complete_workflow_demo() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    # Step 1: Project Setup (20 seconds)
    print_title "Step 1: Professional Project Setup"
    run_demo_command "guild init --name 'e-commerce-api' --description 'Full-featured e-commerce REST API'" \
        "Initialize new Guild project with professional configuration"
    
    run_demo_command "guild config show" \
        "Show Guild configuration and available providers"
    
    # Step 2: Agent Overview (25 seconds)
    print_title "Step 2: Multi-Agent Development Team"
    run_demo_command "guild agents list --detailed" \
        "Show detailed information about available AI agents"
    
    run_demo_command "guild agents capabilities --agent manager" \
        "Show manager agent capabilities and specializations"
    
    # Step 3: Commission Creation (40 seconds)
    print_title "Step 3: Professional Commission Development"
    create_demo_commission "e-commerce-commission.md" "E-Commerce Platform API"
    
    run_demo_command "cat e-commerce-commission.md" \
        "Review comprehensive development commission"
    
    run_demo_command "guild commission refine e-commerce-commission.md" \
        "Refine commission with AI-powered analysis"
    
    # Step 4: Multi-Agent Coordination (60 seconds)
    print_title "Step 4: Guild Orchestration in Action"
    run_demo_command "guild commission -f e-commerce-commission.md --watch" \
        "Submit commission and watch real-time agent coordination"
    
    sleep 5  # Allow time for agents to start working
    
    run_demo_command "guild status --live" \
        "Monitor live agent status and task progress"
    
    run_demo_command "guild workshop" \
        "View Kanban board with task breakdown and assignments"
    
    # Step 5: Interactive Development (45 seconds)
    print_title "Step 5: Interactive Development Chat"
    print_info "Demonstrating Guild's interactive development environment..."
    
    # Simulate chat interaction
    echo "guild chat --campaign e-commerce"
    echo "Starting interactive development session..."
    sleep 2
    
    type_command "@manager What's the current status of the user authentication module?" "guild> "
    print_info "Manager agent provides detailed status update..."
    sleep 3
    
    type_command "@developer Can you show me the database schema for users?" "guild> "
    print_info "Developer agent shows generated schema with explanations..."
    sleep 3
    
    type_command "/tools status" "guild> "
    print_info "Display active tool executions and progress..."
    sleep 2
    
    # Step 6: Code Generation and Review (35 seconds)
    print_title "Step 6: Generated Code and Quality Assurance"
    run_demo_command "find . -name '*.go' -o -name '*.sql' | head -10" \
        "Show generated code files"
    
    run_demo_command "guild code review --summary" \
        "Run automated code review on generated files"
    
    run_demo_command "guild test run --coverage" \
        "Execute generated tests with coverage report"
    
    # Step 7: Documentation and Deployment (25 seconds)
    print_title "Step 7: Professional Documentation"
    run_demo_command "guild docs generate" \
        "Generate comprehensive API documentation"
    
    run_demo_command "guild export --format readme" \
        "Export professional README with setup instructions"
    
    # Step 8: Final Status (20 seconds)
    print_title "Step 8: Project Completion Status"
    run_demo_command "guild status --final" \
        "Show final project status and completion metrics"
    
    run_demo_command "guild metrics" \
        "Display development metrics and time savings"
    
    # Final demonstration summary
    echo
    print_success "🎉 Complete Guild Workflow Demonstration Finished!"
    print_title "What You Just Witnessed:"
    echo "  ✅ Professional project initialization"
    echo "  ✅ Multi-agent team coordination"
    echo "  ✅ AI-powered commission refinement"
    echo "  ✅ Real-time development orchestration"
    echo "  ✅ Interactive development environment"
    echo "  ✅ Automated code generation & review"
    echo "  ✅ Professional documentation generation"
    echo "  ✅ Comprehensive quality assurance"
    echo
    print_info "Guild Framework: Where AI agents become your development team"
    print_title "Ready to 10x your development productivity!"
    
    sleep 3
}

show_demo_preview() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    print_info "This comprehensive demo showcases:"
    echo "  • Professional project setup and configuration"
    echo "  • Multi-agent development team coordination"
    echo "  • AI-powered commission refinement process"
    echo "  • Real-time orchestration and task management"
    echo "  • Interactive development chat environment"
    echo "  • Automated code generation and review"
    echo "  • Professional documentation generation"
    echo "  • Quality assurance and testing integration"
    echo
    print_warning "Duration: ~5 minutes"
    print_info "Perfect for detailed product demonstrations and sales presentations"
    echo
    print_title "Demonstration Flow:"
    echo "  1. Project Setup (20s) - Professional initialization"
    echo "  2. Agent Overview (25s) - Multi-agent capabilities"
    echo "  3. Commission Creation (40s) - AI-powered planning"
    echo "  4. Multi-Agent Coordination (60s) - Live orchestration"
    echo "  5. Interactive Development (45s) - Chat-driven workflow"
    echo "  6. Code Generation & Review (35s) - Quality assurance"
    echo "  7. Documentation (25s) - Professional deliverables"
    echo "  8. Final Status (20s) - Completion metrics"
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
        echo "Guild Complete Workflow Demo"
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