#!/bin/bash
# Demo Master - Professional demonstration orchestrator
# Coordinates all Guild Framework demonstrations and recording

source "$(dirname "$0")/lib/recording-utils.sh"

# Master demo configuration
VERSION="1.0.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Available demo scripts
declare -A DEMO_SCRIPTS=(
    ["quick"]="01-quick-start-demo.sh"
    ["complete"]="02-complete-workflow-demo.sh"
    ["features"]="03-feature-showcase-demo.sh"
    ["interactive"]="interactive-demo.sh"
    ["developer"]="audience-specific/01-developer-focused-demo.sh"
    ["executive"]="audience-specific/02-executive-demo.sh"
    ["sales"]="audience-specific/03-sales-demo.sh"
)

declare -A DEMO_DESCRIPTIONS=(
    ["quick"]="30-second quick start impression demo"
    ["complete"]="5-minute comprehensive workflow demonstration"
    ["features"]="2-minute feature showcase highlighting advantages"
    ["interactive"]="Guided step-by-step tutorial system"
    ["developer"]="Technical demonstration for developers and engineers"
    ["executive"]="Business value and ROI demonstration for executives"
    ["sales"]="Compelling demonstration for prospects and customers"
)

main() {
    local command="${1:-menu}"
    
    case "$command" in
        "menu")
            show_master_menu
            ;;
        "record-all")
            record_all_demos
            ;;
        "validate-all")
            validate_all_demos
            ;;
        "generate-readme")
            generate_readme_materials
            ;;
        "social-media")
            generate_social_media_clips
            ;;
        "clean")
            clean_demo_artifacts
            ;;
        "status")
            show_demo_status
            ;;
        "help")
            show_help
            ;;
        *)
            if [[ -n "${DEMO_SCRIPTS[$command]}" ]]; then
                run_demo "$command" "${@:2}"
            else
                print_error "Unknown command: $command"
                show_help
                exit 1
            fi
            ;;
    esac
}

show_master_menu() {
    clear
    print_title "Guild Framework Demo Master v$VERSION"
    echo "Professional demonstration orchestrator for Guild Framework"
    echo
    echo "═══════════════════════════════════════════════════════════════════════════════════════════════════════════════════"
    echo
    
    print_info "📺 Available Demonstrations:"
    echo
    for demo in "${!DEMO_SCRIPTS[@]}"; do
        local script="${DEMO_SCRIPTS[$demo]}"
        local desc="${DEMO_DESCRIPTIONS[$demo]}"
        printf "  %-12s - %s\n" "$demo" "$desc"
    done
    
    echo
    print_info "🎬 Batch Operations:"
    echo "  record-all     - Record all demonstrations automatically"
    echo "  validate-all   - Validate all demo environments"
    echo "  generate-readme - Create README-ready materials"
    echo "  social-media   - Generate social media clips and content"
    echo
    print_info "🛠️  Utilities:"
    echo "  status         - Show current demo status and files"
    echo "  clean          - Clean up demo artifacts and recordings"
    echo "  help           - Show detailed help information"
    echo
    echo "═══════════════════════════════════════════════════════════════════════════════════════════════════════════════════"
    echo
    
    read -p "Enter command or demo name: " -r
    echo
    
    if [[ -n "$REPLY" ]]; then
        main "$REPLY"
    fi
}

run_demo() {
    local demo_name="$1"
    local demo_args=("${@:2}")
    
    local script_path="$SCRIPT_DIR/${DEMO_SCRIPTS[$demo_name]}"
    
    if [[ ! -f "$script_path" ]]; then
        print_error "Demo script not found: $script_path"
        return 1
    fi
    
    print_status "Running demo: $demo_name"
    print_info "Script: ${DEMO_SCRIPTS[$demo_name]}"
    print_info "Description: ${DEMO_DESCRIPTIONS[$demo_name]}"
    echo
    
    # Make script executable and run it
    chmod +x "$script_path"
    "$script_path" "${demo_args[@]}"
}

record_all_demos() {
    print_title "Recording All Guild Framework Demonstrations"
    print_warning "This will record all demos automatically. Ensure your environment is ready."
    echo
    
    # Validate environment first
    if ! validate_recording_environment; then
        print_error "Environment validation failed. Please fix issues before recording."
        return 1
    fi
    
    read -p "Continue with recording all demos? (y/N): " -n 1 -r
    echo
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Recording cancelled."
        return 0
    fi
    
    local total_demos=${#DEMO_SCRIPTS[@]}
    local current_demo=1
    local failed_demos=()
    local successful_demos=()
    
    for demo_name in "${!DEMO_SCRIPTS[@]}"; do
        print_title "Recording Demo $current_demo/$total_demos: $demo_name"
        print_info "${DEMO_DESCRIPTIONS[$demo_name]}"
        echo
        
        if run_demo "$demo_name" "record"; then
            successful_demos+=("$demo_name")
            print_success "✅ $demo_name recorded successfully"
        else
            failed_demos+=("$demo_name")
            print_error "❌ $demo_name recording failed"
        fi
        
        echo
        ((current_demo++))
        
        # Pause between demos
        if [[ $current_demo -le $total_demos ]]; then
            sleep 2
        fi
    done
    
    # Show final results
    echo
    print_title "Recording Session Complete"
    echo
    
    if [[ ${#successful_demos[@]} -gt 0 ]]; then
        print_success "Successfully recorded (${#successful_demos[@]}):"
        for demo in "${successful_demos[@]}"; do
            echo "  ✅ $demo"
        done
    fi
    
    if [[ ${#failed_demos[@]} -gt 0 ]]; then
        print_error "Failed recordings (${#failed_demos[@]}):"
        for demo in "${failed_demos[@]}"; do
            echo "  ❌ $demo"
        done
    fi
    
    echo
    print_info "Use 'demo-master.sh status' to see generated files"
}

validate_all_demos() {
    print_title "Validating All Demo Environments"
    echo
    
    local validation_passed=true
    
    # Global validation
    print_status "Running global environment validation..."
    if validate_recording_environment; then
        print_success "✅ Global environment validation passed"
    else
        print_error "❌ Global environment validation failed"
        validation_passed=false
    fi
    
    echo
    
    # Individual demo validation
    for demo_name in "${!DEMO_SCRIPTS[@]}"; do
        print_status "Validating $demo_name demo..."
        
        if run_demo "$demo_name" "validate" &> /dev/null; then
            print_success "✅ $demo_name validation passed"
        else
            print_error "❌ $demo_name validation failed"
            validation_passed=false
        fi
    done
    
    echo
    if [[ "$validation_passed" == "true" ]]; then
        print_success "🎉 All demo validations passed!"
        print_info "Environment is ready for professional recording"
    else
        print_error "⚠️  Some validations failed"
        print_info "Please fix issues before recording demos"
        return 1
    fi
}

generate_readme_materials() {
    print_title "Generating README-Ready Demonstration Materials"
    echo
    
    local readme_dir="$DEMO_HOME/readme-materials"
    mkdir -p "$readme_dir"
    
    print_status "Creating professional README content..."
    
    # Generate README section
    cat > "$readme_dir/README-demo-section.md" << 'EOF'
## 🎬 Live Demonstrations

See Guild Framework in action with these professional demonstrations:

### Quick Start (30 seconds)
Perfect for first impressions and social media sharing.

![Guild Quick Start](docs/images/guild-quick-start.gif)

**What you'll see:**
- Instant project initialization
- Multi-agent coordination
- Professional development workflow
- Real-time progress tracking

### Complete Workflow (5 minutes)
Comprehensive demonstration of Guild's full capabilities.

![Guild Complete Workflow](docs/images/guild-complete-workflow.gif)

**Demonstration highlights:**
- Professional project setup and configuration
- AI-powered commission analysis and planning
- Multi-agent development team coordination
- Interactive development chat environment
- Automated code generation and quality assurance
- Professional documentation and deliverables

### Feature Showcase (2 minutes)
Focused demonstration of Guild's unique advantages over traditional tools.

![Guild Feature Showcase](docs/images/guild-feature-showcase.gif)

**Key differentiators:**
- Intelligent commission processing with AI analysis
- Multi-agent orchestration with real-time coordination
- Context-aware code generation with enterprise quality
- Professional visual interface with rich interactions
- Enterprise-ready integration capabilities

### Interactive Tutorial
Step-by-step guided learning experience for new users.

**Get hands-on experience:**
```bash
# Start the interactive tutorial
./scripts/recording/interactive-demo.sh tutorial

# Or try the quick demo
./scripts/recording/01-quick-start-demo.sh
```

---

*All demonstrations use real Guild functionality - no mocked responses or fake data.*
EOF

    # Generate marketing copy
    cat > "$readme_dir/marketing-copy.md" << 'EOF'
# Guild Framework Marketing Materials

## Elevator Pitch (30 seconds)
"Guild Framework transforms development teams with AI agents that work together like human developers. Instead of struggling with isolated AI tools, Guild provides a coordinated multi-agent system that plans, codes, reviews, and documents your projects professionally. It's like having a senior development team available 24/7."

## Value Propositions

### For Individual Developers
- **10x Productivity**: Multi-agent coordination eliminates context switching
- **Professional Quality**: Enterprise-grade code generation and documentation  
- **Learning Accelerator**: AI agents explain decisions and teach best practices
- **24/7 Availability**: Never blocked waiting for team members

### For Development Teams  
- **Consistent Standards**: Automated enforcement of coding standards and practices
- **Knowledge Sharing**: Centralized project intelligence and documentation
- **Onboarding Speed**: New team members productive from day one
- **Technical Debt Reduction**: Continuous refactoring and quality improvements

### For Engineering Managers
- **Predictable Delivery**: AI-powered estimation and progress tracking
- **Resource Optimization**: Intelligent task distribution and load balancing
- **Quality Assurance**: Automated code review and testing integration
- **Visibility**: Real-time insights into development progress and bottlenecks

## Competitive Advantages

| Feature | Guild Framework | Traditional AI Tools |
|---------|----------------|-------------------|
| Multi-Agent Coordination | ✅ Built-in orchestration | ❌ Manual coordination |
| Context Awareness | ✅ Project-wide intelligence | ❌ Limited context |
| Professional Interface | ✅ Rich visual experience | ❌ Plain text only |
| Enterprise Integration | ✅ Production-ready | ❌ Prototype quality |
| Learning Curve | ✅ Guided tutorials | ❌ Trial and error |
| Team Collaboration | ✅ Shared intelligence | ❌ Individual tools |

## Social Media Copy

### Twitter/X Posts
"🤖 Just watched @GuildFramework coordinate 5 AI agents to build a complete API in minutes. This isn't the future - it's available now. Multi-agent development is here. #AI #Development #Productivity"

"📈 10x productivity isn't hype when AI agents work together like a real dev team. Guild Framework shows what coordinated AI can accomplish. Demo: [link] #AIAgent #DevTools"

### LinkedIn Posts  
"I've been exploring AI development tools, and Guild Framework stands out for its multi-agent approach. Instead of juggling separate AI tools, Guild coordinates specialized agents that plan, code, review, and document together. The productivity gains are remarkable."

### YouTube Descriptions
"Watch Guild Framework transform development with coordinated AI agents. This 5-minute demo shows how multi-agent systems create professional applications faster than traditional development workflows. See real agents planning, coding, reviewing, and documenting a complete project."
EOF

    # Copy demo GIFs if they exist
    if [[ -d "$DEMO_HOME" ]]; then
        find "$DEMO_HOME" -name "*.gif" -exec cp {} "$readme_dir/" \;
        print_info "Copied demo GIFs to README materials"
    fi
    
    print_success "README materials generated in: $readme_dir"
    print_info "Files created:"
    echo "  • README-demo-section.md - Ready-to-use README content"
    echo "  • marketing-copy.md - Marketing materials and social media copy"
    echo "  • *.gif - Demo GIF files (if available)"
    echo
    print_info "Next steps:"
    echo "  1. Copy README-demo-section.md content to your README.md"
    echo "  2. Move GIF files to docs/images/ directory"
    echo "  3. Use marketing copy for website and social media"
}

generate_social_media_clips() {
    print_title "Generating Social Media Content and Clips"
    echo
    
    local social_script="$SCRIPT_DIR/social-media-generator.sh"
    
    if [[ -f "$social_script" ]]; then
        print_status "Running comprehensive social media content generator..."
        "$social_script" generate-all
    else
        print_warning "Social media generator not found, creating basic clips..."
        generate_basic_social_clips
    fi
}

generate_basic_social_clips() {
    local social_dir="$DEMO_HOME/social-media"
    mkdir -p "$social_dir"
    
    print_status "Creating optimized social media clips..."
    
    # Check for existing recordings
    local found_recordings=false
    
    for demo_name in "${!DEMO_SCRIPTS[@]}"; do
        local gif_file="$DEMO_HOME/$demo_name-*/guild-$demo_name-*.gif"
        
        if ls $gif_file 2>/dev/null; then
            found_recordings=true
            
            print_info "Processing $demo_name for social media..."
            
            # Copy and optimize for different platforms
            for gif in $gif_file; do
                local basename=$(basename "$gif" .gif)
                
                # Twitter/X optimized (max 15MB, shorter clips)
                if command -v gifsicle &> /dev/null; then
                    gifsicle --resize-fit 640x480 --optimize=3 --colors 128 \
                        "$gif" -o "$social_dir/${basename}-twitter.gif"
                fi
                
                # LinkedIn optimized (professional format)
                if command -v gifsicle &> /dev/null; then
                    gifsicle --resize-fit 800x600 --optimize=3 --colors 256 \
                        "$gif" -o "$social_dir/${basename}-linkedin.gif"
                fi
                
                # Instagram/TikTok optimized (square format)
                if command -v gifsicle &> /dev/null; then
                    gifsicle --resize-fit 640x640 --optimize=3 --colors 128 \
                        "$gif" -o "$social_dir/${basename}-instagram.gif"
                fi
            done
        fi
    done
    
    if [[ "$found_recordings" == "false" ]]; then
        print_warning "No demo recordings found. Please record demos first:"
        echo "  ./demo-master.sh record-all"
        return 1
    fi
    
    # Generate basic social media post templates
    cat > "$social_dir/social-media-posts.md" << 'EOF'
# Social Media Post Templates

## Twitter/X Posts (280 characters)

### Quick Start Demo
🚀 NEW: Watch Guild Framework coordinate AI agents like a real dev team! 

30-second demo shows:
✅ Multi-agent planning
✅ Coordinated development  
✅ Professional output

This is the future of development 🔥

[GIF: guild-quick-start-twitter.gif]

#AI #Development #Productivity

### Feature Showcase
🤖 Why Guild Framework beats traditional AI coding tools:

✅ Multi-agent coordination (not isolated prompts)
✅ Professional visual interface (not plain text)
✅ Enterprise-ready output (not prototypes)

See the difference: [GIF]

#AIAgent #DevTools #Enterprise

### Complete Workflow
📈 5 minutes to see how AI agents can replace entire development workflows:

🎯 Commission planning
🤖 Multi-agent coordination
💻 Professional code generation
✅ Quality assurance

Game-changing productivity: [GIF]

#Productivity #AI #SoftwareDevelopment

## LinkedIn Posts (3000 characters)

### Professional Announcement
I've been exploring the next generation of development tools, and Guild Framework represents a fundamental shift in how we think about AI-assisted development.

Unlike traditional AI coding tools that provide isolated assistance, Guild Framework coordinates specialized AI agents that work together like a real development team:

🏗️ Manager agents analyze requirements and create project plans
💻 Developer agents implement features with full context awareness  
🔍 Reviewer agents ensure code quality and best practices
📝 Documentation agents create professional deliverables

The 5-minute demo shows what this coordination accomplishes - it's not just faster development, it's fundamentally better development.

Key differentiators I've observed:
• Multi-agent orchestration eliminates context switching overhead
• Professional visual interface improves developer experience significantly
• Enterprise-ready output quality from day one
• Shared project intelligence accelerates team onboarding

For engineering leaders evaluating AI development tools, Guild Framework deserves serious consideration. The productivity gains are measurable, but the quality improvements might be even more valuable.

What questions do you have about multi-agent development workflows?

[Attach: guild-complete-workflow-linkedin.gif]

### Technical Deep Dive
Technical leaders: If you're evaluating AI development tools, here's what makes Guild Framework architecturally different:

🏛️ MULTI-AGENT ORCHESTRATION
Traditional tools: Single AI instance per interaction
Guild: Specialized agents with role-based coordination

🧠 CONTEXT MANAGEMENT  
Traditional tools: Limited conversation context
Guild: Project-wide knowledge graph with semantic search

🎨 INTERFACE DESIGN
Traditional tools: Plain text chat interfaces
Guild: Rich visual components with real-time status

🏢 ENTERPRISE INTEGRATION
Traditional tools: Prototype-quality outputs
Guild: Production-ready code with professional documentation

The architectural choices create compound benefits:
• Reduced cognitive load for developers
• Consistent quality across all project components  
• Accelerated onboarding for new team members
• Measurable productivity improvements

Demo shows these principles in action: [GIF]

How are you evaluating AI tools for your development teams?

#AI #SoftwareArchitecture #EngineeringLeadership #DevTools

## Instagram/TikTok Captions

### Quick Visual Demo
🤖 AI agents working together like a real dev team ✨

Watch how Guild Framework coordinates:
• Planning 🎯
• Coding 💻  
• Review ✅
• Documentation 📝

All automatically! 🚀

This is why multi-agent > single AI 

#AI #Coding #Tech #Programming #Developer #Productivity #TechTok #Innovation

### Behind the Scenes
POV: You just discovered multi-agent development 🤯

❌ Old way: Manual AI prompting
✅ New way: Coordinated AI agents

Guild Framework = Game changer 📈

#TechTrends #AI #Development #Innovation #Productivity #DevLife #TechTok
EOF

    print_success "Social media materials generated in: $social_dir"
    print_info "Files created:"
    echo "  • Optimized GIFs for different platforms"
    echo "  • social-media-posts.md with ready-to-use content"
    echo "  • Platform-specific formatting and hashtags"
}

show_demo_status() {
    print_title "Guild Framework Demo Status"
    echo
    
    # Check environment
    print_info "🌍 Environment Status:"
    if validate_recording_environment &> /dev/null; then
        print_success "✅ Recording environment ready"
    else
        print_warning "⚠️  Recording environment needs attention"
    fi
    
    if check_guild_providers &> /dev/null; then
        print_success "✅ Guild providers configured"
    else
        print_warning "⚠️  Guild providers need configuration"
    fi
    
    echo
    
    # Check demo recordings
    print_info "📹 Demo Recordings:"
    local recordings_found=false
    
    if [[ -d "$DEMO_HOME" ]]; then
        for demo_name in "${!DEMO_SCRIPTS[@]}"; do
            local demo_dir="$DEMO_HOME/*$demo_name*"
            local cast_files=$(find $demo_dir -name "*.cast" 2>/dev/null | wc -l)
            local gif_files=$(find $demo_dir -name "*.gif" 2>/dev/null | wc -l)
            
            if [[ $cast_files -gt 0 || $gif_files -gt 0 ]]; then
                recordings_found=true
                printf "  %-12s - %d recordings, %d GIFs\n" "$demo_name" "$cast_files" "$gif_files"
            else
                printf "  %-12s - No recordings\n" "$demo_name"
            fi
        done
    else
        echo "  No recordings directory found"
    fi
    
    echo
    
    # Show file sizes and locations
    if [[ "$recordings_found" == "true" ]]; then
        print_info "📁 Generated Files:"
        find "$DEMO_HOME" -name "*.cast" -o -name "*.gif" 2>/dev/null | while read -r file; do
            local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
            local size_mb=$((size / 1024 / 1024))
            printf "  %s (%dMB)\n" "$(basename "$file")" "$size_mb"
        done
    fi
    
    echo
    print_info "🚀 Next Steps:"
    if [[ "$recordings_found" == "false" ]]; then
        echo "  1. Run: ./demo-master.sh record-all"
        echo "  2. Generate materials: ./demo-master.sh generate-readme"
        echo "  3. Create social clips: ./demo-master.sh social-media"
    else
        echo "  1. Generate README materials: ./demo-master.sh generate-readme"
        echo "  2. Create social media clips: ./demo-master.sh social-media"
        echo "  3. Upload GIFs to documentation"
        echo "  4. Share on social media platforms"
    fi
}

clean_demo_artifacts() {
    print_title "Cleaning Demo Artifacts"
    print_warning "This will remove all demo recordings and generated materials."
    echo
    
    if [[ -d "$DEMO_HOME" ]]; then
        echo "Files to be removed:"
        find "$DEMO_HOME" -type f | head -20
        echo
        
        read -p "Continue with cleanup? (y/N): " -n 1 -r
        echo
        
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$DEMO_HOME"
            print_success "Demo artifacts cleaned"
        else
            print_info "Cleanup cancelled"
        fi
    else
        print_info "No demo artifacts to clean"
    fi
    
    # Clean temporary files
    rm -f /tmp/guild-tutorial-state.json
    rm -f /tmp/guild-prewarm*
    
    print_success "Temporary files cleaned"
}

show_help() {
    cat << 'EOF'
Guild Framework Demo Master v1.0.0

USAGE:
    demo-master.sh [COMMAND] [OPTIONS]

COMMANDS:
    Individual Demos:
        quick           Run quick start demo (30 seconds)
        complete        Run complete workflow demo (5 minutes)  
        features        Run feature showcase demo (2 minutes)
        interactive     Run interactive tutorial system

    Batch Operations:
        record-all      Record all demonstrations automatically
        validate-all    Validate all demo environments
        generate-readme Create README-ready materials
        social-media    Generate social media optimized clips

    Utilities:
        status          Show demo status and generated files
        clean           Clean up demo artifacts and recordings
        menu            Show interactive menu (default)
        help            Show this help information

EXAMPLES:
    # Interactive menu
    ./demo-master.sh

    # Record specific demo
    ./demo-master.sh quick record

    # Validate environment
    ./demo-master.sh validate-all

    # Generate all materials
    ./demo-master.sh record-all
    ./demo-master.sh generate-readme
    ./demo-master.sh social-media

DEMO OPTIONS:
    Each demo script supports these options:
        record      - Record the demo automatically
        validate    - Validate environment for recording
        preview     - Show what will be demonstrated
        help        - Show demo-specific help

ENVIRONMENT VARIABLES:
    DEMO_HOME           Demo recordings directory (/tmp/guild-recordings)
    GUILD_BIN           Guild binary path (./guild)
    GUILD_MOCK_PROVIDER Use mock provider for demos (true/false)

For more information, visit: https://github.com/guild-ventures/guild-framework
EOF
}

# Handle command line arguments
main "$@"