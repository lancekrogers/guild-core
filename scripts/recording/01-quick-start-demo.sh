#!/bin/bash
# Quick Start Demo - 30 second impression demo
# Shows Guild's most impressive features in minimal time

source "$(dirname "$0")/lib/recording-utils.sh"

# Demo configuration
DEMO_NAME="Quick Start"
DEMO_DESCRIPTION="30-second showcase of Guild's core capabilities"
RECORDING_NAME="guild-quick-start"

main() {
    local mode="${1:-interactive}"
    
    # Initialize recording environment
    recording_init "$RECORDING_NAME"
    
    if [[ "$mode" == "record" ]]; then
        # Start recording and run demo automatically
        {
            run_quick_start_demo
        } | start_recording "$RECORDING_NAME" "Guild Framework - Quick Start Demo"
        
        # Generate GIF
        generate_gif "$RECORD_FILE" "${RECORDING_DIR}/guild-quick-start.gif"
        
        echo
        print_success "Quick start demo recorded successfully!"
        print_info "Files created:"
        print_info "  - Recording: $RECORD_FILE"
        print_info "  - GIF: ${RECORDING_DIR}/guild-quick-start.gif"
        
    elif [[ "$mode" == "validate" ]]; then
        # Validate the demo can run
        validate_recording_environment
        check_guild_providers
        print_success "Quick start demo validation passed"
        
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

run_quick_start_demo() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    # Step 1: Initialize Guild (2 seconds)
    print_title "Step 1: Initialize Guild Framework"
    run_demo_command "guild init --name 'quick-demo' --description 'Quick demo project'" \
        "Initialize a new Guild project"
    
    # Step 2: Show agents (3 seconds)  
    print_title "Step 2: Available AI Agents"
    run_demo_command "guild agents list" \
        "List all available AI agents"
    
    # Step 3: Create commission (8 seconds)
    print_title "Step 3: Create Development Commission"
    create_demo_commission "quick-task.md" "User Authentication API"
    run_demo_command "cat quick-task.md" \
        "Show the development commission"
    
    # Step 4: Submit commission (10 seconds)
    print_title "Step 4: Submit to Guild"
    run_demo_command "guild commission -f quick-task.md" \
        "Submit commission to Guild for processing"
    
    # Step 5: Show progress (7 seconds)
    print_title "Step 5: Agent Coordination"
    run_demo_command "guild status" \
        "Check agent status and task progress"
    
    # Final message
    echo
    print_success "🎉 Guild Framework Demo Complete!"
    print_info "In 30 seconds, you saw:"
    echo "  ✅ Project initialization"
    echo "  ✅ Multi-agent system"
    echo "  ✅ Commission processing"
    echo "  ✅ Real-time coordination"
    echo
    print_title "Ready to transform your development workflow!"
    sleep 2
}

show_demo_preview() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    print_info "This demo will showcase:"
    echo "  • Quick Guild project initialization"
    echo "  • Multi-agent system overview"
    echo "  • Commission-based development"
    echo "  • Real-time agent coordination"
    echo "  • Professional development workflow"
    echo
    print_warning "Duration: ~30 seconds"
    print_info "Perfect for first impressions and social media"
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
        echo "Guild Quick Start Demo"
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