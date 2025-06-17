#!/bin/bash
# Interactive Demo Mode - Guided tutorial system
# Provides step-by-step interactive demonstrations with help and recovery

source "$(dirname "$0")/lib/recording-utils.sh"

# Demo configuration
DEMO_NAME="Interactive Tutorial"
DEMO_DESCRIPTION="Guided step-by-step demonstration with interactive prompts"
RECORDING_NAME="guild-interactive-demo"

# Tutorial state management
TUTORIAL_STATE_FILE="/tmp/guild-tutorial-state.json"
CURRENT_STEP=1
TOTAL_STEPS=8

main() {
    local mode="${1:-interactive}"
    
    # Initialize recording environment
    recording_init "$RECORDING_NAME"
    
    case "$mode" in
        "record")
            run_interactive_demo_recording
            ;;
        "tutorial")
            run_guided_tutorial
            ;;
        "validate")
            validate_recording_environment
            check_guild_providers
            print_success "Interactive demo validation passed"
            ;;
        "reset")
            reset_tutorial_state
            ;;
        *)
            show_demo_menu
            ;;
    esac
}

show_demo_menu() {
    clear
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    echo "Choose your demonstration mode:"
    echo
    echo "  1. 🎓 Guided Tutorial      - Interactive step-by-step learning"
    echo "  2. 🎬 Record Demo         - Create professional recording"
    echo "  3. 👀 Preview Mode        - See what will be demonstrated"
    echo "  4. ✅ Validate Setup      - Check environment readiness"
    echo "  5. 🔄 Reset Progress      - Start tutorial from beginning"
    echo "  6. ❓ Help & Tips         - Get help and best practices"
    echo "  7. 🚪 Exit               - Exit demo system"
    echo
    
    read -p "Select option (1-7): " -n 1 -r
    echo
    echo
    
    case $REPLY in
        1) run_guided_tutorial ;;
        2) run_interactive_demo_recording ;;
        3) show_demo_preview ;;
        4) main validate ;;
        5) reset_tutorial_state ;;
        6) show_help_and_tips ;;
        7) exit 0 ;;
        *) 
            print_error "Invalid selection"
            sleep 1
            main
            ;;
    esac
}

run_guided_tutorial() {
    load_tutorial_state
    
    show_demo_banner "Guild Framework Tutorial" "Step $CURRENT_STEP of $TOTAL_STEPS"
    
    case $CURRENT_STEP in
        1) tutorial_step_initialization ;;
        2) tutorial_step_configuration ;;
        3) tutorial_step_agents ;;
        4) tutorial_step_commission_creation ;;
        5) tutorial_step_submission ;;
        6) tutorial_step_monitoring ;;
        7) tutorial_step_interaction ;;
        8) tutorial_step_completion ;;
        *) tutorial_complete ;;
    esac
}

tutorial_step_initialization() {
    print_title "Step 1: Guild Project Initialization"
    print_info "Learn how to set up a new Guild project from scratch"
    echo
    
    echo "First, let's initialize a new Guild project. This creates the necessary"
    echo "directory structure and configuration files."
    echo
    
    show_command_prompt "guild init --name 'tutorial-project' --description 'Learning Guild Framework'" \
        "Initialize a new Guild project"
    
    # Validate user ran the command
    if [[ ! -d ".guild" ]]; then
        print_warning "Guild project not initialized. Please run the command above."
        echo "Press Enter to continue when ready..."
        read -r
        return
    fi
    
    print_success "✅ Project initialized successfully!"
    echo
    print_info "What just happened:"
    echo "  • Created .guild/ directory for project data"
    echo "  • Generated guild.yaml configuration file"
    echo "  • Set up SQLite database for agent memory"
    echo "  • Configured default agent registry"
    echo
    
    advance_tutorial_step
    tutorial_navigation_prompt
}

tutorial_step_configuration() {
    print_title "Step 2: Provider Configuration"
    print_info "Configure AI providers for your Guild agents"
    echo
    
    echo "Guild supports multiple AI providers. Let's check your current configuration"
    echo "and set up providers if needed."
    echo
    
    show_command_prompt "guild config show" \
        "Display current Guild configuration"
    
    echo
    print_info "Configuration includes:"
    echo "  • AI provider settings (OpenAI, Anthropic, etc.)"
    echo "  • Agent behavior preferences"
    echo "  • Project-specific settings"
    echo "  • Tool integration configuration"
    echo
    
    if ! guild config show | grep -q "provider:"; then
        print_warning "No providers configured yet."
        echo
        echo "For this tutorial, we'll use the mock provider to avoid API costs."
        echo "In production, you would configure real AI providers."
        export GUILD_MOCK_PROVIDER=true
        print_info "Mock provider enabled for tutorial"
    fi
    
    advance_tutorial_step
    tutorial_navigation_prompt
}

tutorial_step_agents() {
    print_title "Step 3: Understanding Guild Agents"
    print_info "Explore the multi-agent development team"
    echo
    
    echo "Guild uses specialized AI agents that work together like a development team."
    echo "Each agent has specific capabilities and roles."
    echo
    
    show_command_prompt "guild agents list" \
        "List all available agents"
    
    echo
    show_command_prompt "guild agents capabilities --agent manager" \
        "Show detailed capabilities of the manager agent"
    
    echo
    print_info "Key agent types:"
    echo "  • 👔 Manager   - Plans tasks and coordinates agents"
    echo "  • 💻 Developer - Writes code and implements features"
    echo "  • 🔍 Reviewer  - Reviews code and ensures quality"
    echo "  • 📝 Writer    - Creates documentation and specifications"
    echo "  • 🧪 Tester    - Generates tests and validates functionality"
    echo
    
    advance_tutorial_step
    tutorial_navigation_prompt
}

tutorial_step_commission_creation() {
    print_title "Step 4: Creating Development Commissions"
    print_info "Learn to create professional development requests"
    echo
    
    echo "Commissions are how you request work from your Guild. Think of them as"
    echo "detailed project specifications that agents can understand and execute."
    echo
    
    print_info "Let's create a sample commission file:"
    
    # Create tutorial commission
    cat > tutorial-commission.md << 'EOF'
# User Profile Management API

Build a REST API for user profile management with the following features:

## Requirements
- User registration and authentication
- Profile CRUD operations
- Input validation and error handling
- SQLite database integration
- RESTful endpoint design

## Technical Specifications
- Go/Gin framework
- JWT authentication
- Structured logging
- Unit tests with 80%+ coverage
- API documentation

## Success Criteria
- All endpoints properly documented
- Error responses follow RFC 7807 format
- Database migrations included
- Comprehensive test coverage
EOF

    print_success "Created tutorial-commission.md"
    echo
    
    show_command_prompt "cat tutorial-commission.md" \
        "Review the commission file"
    
    echo
    print_info "Good commissions include:"
    echo "  • Clear requirements and specifications"
    echo "  • Technical constraints and preferences"
    echo "  • Success criteria and acceptance tests"
    echo "  • Context about the project and goals"
    echo
    
    advance_tutorial_step
    tutorial_navigation_prompt
}

tutorial_step_submission() {
    print_title "Step 5: Submitting Commissions to Guild"
    print_info "Submit your commission for AI agent processing"
    echo
    
    echo "Now let's submit the commission to Guild. The agents will analyze it,"
    echo "break it down into tasks, and begin coordinated development work."
    echo
    
    show_command_prompt "guild commission -f tutorial-commission.md" \
        "Submit commission to Guild for processing"
    
    echo
    print_info "What happens during submission:"
    echo "  • Manager agent analyzes the commission"
    echo "  • Tasks are extracted and prioritized"
    echo "  • Agents are assigned based on capabilities"
    echo "  • Work begins with coordination and planning"
    echo
    
    # Give time for processing
    print_warning "Agent processing may take a moment..."
    sleep 3
    
    advance_tutorial_step
    tutorial_navigation_prompt
}

tutorial_step_monitoring() {
    print_title "Step 6: Monitoring Agent Progress"
    print_info "Track your Guild's development progress"
    echo
    
    echo "Guild provides multiple ways to monitor agent activity and progress."
    echo "Let's explore the different monitoring tools available."
    echo
    
    show_command_prompt "guild status" \
        "Check overall Guild status and agent activity"
    
    echo
    show_command_prompt "guild workshop" \
        "View the Kanban board with task assignments"
    
    echo
    print_info "Monitoring tools:"
    echo "  • guild status     - Overall project status"
    echo "  • guild workshop   - Visual Kanban board"
    echo "  • guild agents status - Individual agent status"
    echo "  • guild logs       - Detailed activity logs"
    echo
    
    advance_tutorial_step
    tutorial_navigation_prompt
}

tutorial_step_interaction() {
    print_title "Step 7: Interactive Development"
    print_info "Engage with your agents through Guild's chat interface"
    echo
    
    echo "Guild's interactive chat lets you communicate directly with agents,"
    echo "ask questions, provide guidance, and collaborate on development."
    echo
    
    print_warning "For this tutorial, we'll simulate the chat interface."
    print_info "In practice, you would run: guild chat"
    echo
    
    print_info "Common chat interactions:"
    echo
    type_command "@manager What's the current status of the API development?" "guild> "
    print_info "→ Manager provides detailed progress update"
    echo
    
    type_command "@developer Can you explain the authentication flow?" "guild> "
    print_info "→ Developer explains implementation details"
    echo
    
    type_command "/tools status" "guild> "
    print_info "→ Shows active tool executions and progress"
    echo
    
    type_command "/help" "guild> "
    print_info "→ Shows available commands and agent functions"
    echo
    
    print_success "Interactive development enables real-time collaboration!"
    
    advance_tutorial_step
    tutorial_navigation_prompt
}

tutorial_step_completion() {
    print_title "Step 8: Project Completion and Review"
    print_info "Review deliverables and finalize your project"
    echo
    
    echo "Let's review what the Guild has accomplished and examine the"
    echo "generated code, documentation, and tests."
    echo
    
    show_command_prompt "find . -type f -name '*.go' -o -name '*.md' -o -name '*.sql' | head -10" \
        "List generated project files"
    
    echo
    show_command_prompt "guild metrics" \
        "Show development metrics and productivity gains"
    
    echo
    show_command_prompt "guild export --format summary" \
        "Generate project summary and documentation"
    
    echo
    print_success "🎉 Tutorial Complete!"
    print_title "You've learned:"
    echo "  ✅ Guild project initialization"
    echo "  ✅ Provider configuration"
    echo "  ✅ Multi-agent system overview"
    echo "  ✅ Commission creation and submission"
    echo "  ✅ Progress monitoring and tracking"
    echo "  ✅ Interactive development workflows"
    echo "  ✅ Project completion and review"
    echo
    
    reset_tutorial_state
    
    echo "Ready to start your own Guild projects!"
    echo
    read -p "Press Enter to return to main menu..." -r
    main
}

tutorial_complete() {
    print_success "🎉 Tutorial already completed!"
    echo "Use 'reset' to start over or try recording a demo."
    tutorial_navigation_prompt
}

tutorial_navigation_prompt() {
    echo
    echo "Navigation options:"
    echo "  [Enter] - Continue to next step"
    echo "  [b]     - Go back to previous step"
    echo "  [r]     - Restart tutorial"
    echo "  [m]     - Return to main menu"
    echo "  [h]     - Show help"
    echo "  [q]     - Quit"
    echo
    
    read -p "Choose action: " -n 1 -r
    echo
    
    case $REPLY in
        ""|$'\n')
            # Continue - already handled by advancing step
            run_guided_tutorial
            ;;
        b|B)
            if [[ $CURRENT_STEP -gt 1 ]]; then
                ((CURRENT_STEP--))
                save_tutorial_state
                run_guided_tutorial
            else
                print_warning "Already at first step"
                tutorial_navigation_prompt
            fi
            ;;
        r|R)
            reset_tutorial_state
            run_guided_tutorial
            ;;
        m|M)
            main
            ;;
        h|H)
            show_help_and_tips
            tutorial_navigation_prompt
            ;;
        q|Q)
            exit 0
            ;;
        *)
            print_warning "Invalid option"
            tutorial_navigation_prompt
            ;;
    esac
}

advance_tutorial_step() {
    ((CURRENT_STEP++))
    save_tutorial_state
}

save_tutorial_state() {
    echo "{\"current_step\": $CURRENT_STEP, \"total_steps\": $TOTAL_STEPS}" > "$TUTORIAL_STATE_FILE"
}

load_tutorial_state() {
    if [[ -f "$TUTORIAL_STATE_FILE" ]]; then
        CURRENT_STEP=$(grep -o '"current_step": [0-9]*' "$TUTORIAL_STATE_FILE" | cut -d' ' -f2)
        TOTAL_STEPS=$(grep -o '"total_steps": [0-9]*' "$TUTORIAL_STATE_FILE" | cut -d' ' -f2)
    fi
}

reset_tutorial_state() {
    rm -f "$TUTORIAL_STATE_FILE"
    CURRENT_STEP=1
    print_success "Tutorial progress reset"
    sleep 1
}

run_interactive_demo_recording() {
    print_title "Recording Interactive Demo"
    print_warning "This will record your screen. Make sure your terminal is properly sized."
    echo
    
    read -p "Start recording? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        {
            run_guided_tutorial
        } | start_recording "$RECORDING_NAME" "Guild Framework - Interactive Tutorial"
        
        generate_gif "$RECORD_FILE" "${RECORDING_DIR}/guild-interactive-demo.gif"
        
        print_success "Interactive demo recorded successfully!"
    fi
}

show_demo_preview() {
    show_demo_banner "Interactive Demo Preview" "What you'll learn in the guided tutorial"
    
    print_info "The interactive tutorial covers:"
    echo "  1. 🚀 Project Initialization - Set up Guild from scratch"
    echo "  2. ⚙️  Configuration - Configure providers and settings"
    echo "  3. 🤖 Agent Overview - Understanding the multi-agent system"
    echo "  4. 📝 Commission Creation - Writing effective development requests"
    echo "  5. 🎯 Submission Process - Getting agents to work on your project"
    echo "  6. 📊 Progress Monitoring - Tracking development progress"
    echo "  7. 💬 Interactive Development - Real-time collaboration with agents"
    echo "  8. ✅ Project Completion - Reviewing deliverables and results"
    echo
    print_warning "Interactive features:"
    echo "  • Step-by-step guided progression"
    echo "  • Help and tips at each stage"
    echo "  • Error recovery and retry options"
    echo "  • Navigation controls (back/forward/restart)"
    echo "  • Progress tracking and state management"
    echo
    
    read -p "Press Enter to return to menu..." -r
    main
}

show_help_and_tips() {
    clear
    print_title "Guild Demo Help & Tips"
    echo
    
    print_info "Getting the Best Demo Experience:"
    echo
    echo "🖥️  Terminal Setup:"
    echo "  • Use a terminal at least 120x35 characters"
    echo "  • Enable true color support (COLORTERM=truecolor)"
    echo "  • Use a monospace font (SF Mono, Monaco, Cascadia Code)"
    echo
    echo "🎬 Recording Tips:"
    echo "  • Close unnecessary applications to reduce distractions"
    echo "  • Ensure good lighting if recording your screen"
    echo "  • Speak clearly if adding narration"
    echo "  • Test your setup with a short recording first"
    echo
    echo "🐛 Troubleshooting:"
    echo "  • If Guild commands fail, check provider configuration"
    echo "  • Use mock provider (GUILD_MOCK_PROVIDER=true) for demos"
    echo "  • Run 'guild demo-check' to validate your environment"
    echo "  • Check logs with 'guild logs' if something goes wrong"
    echo
    echo "⚡ Performance Tips:"
    echo "  • Pre-warm caches before recording"
    echo "  • Use SSD storage for faster file operations"
    echo "  • Close resource-intensive applications"
    echo "  • Ensure stable internet connection for real providers"
    echo
    
    read -p "Press Enter to continue..." -r
}

# Handle command line arguments
case "${1:-}" in
    "record")
        main record
        ;;
    "tutorial")
        main tutorial
        ;;
    "validate")
        main validate
        ;;
    "reset")
        main reset
        ;;
    "help"|"--help"|"-h")
        echo "Guild Interactive Demo System"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  tutorial    Start guided interactive tutorial"
        echo "  record      Record interactive demo session"
        echo "  validate    Validate environment for demo"
        echo "  reset       Reset tutorial progress"
        echo "  help        Show this help message"
        echo ""
        echo "Interactive menu will start if no command provided."
        ;;
    *)
        main
        ;;
esac