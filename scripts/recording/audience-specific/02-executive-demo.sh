#!/bin/bash
# Executive Demo - Business value and ROI demonstration
# Shows Guild's business impact and competitive advantages

source "$(dirname "$0")/../lib/recording-utils.sh"

# Demo configuration
DEMO_NAME="Executive Business Demo"
DEMO_DESCRIPTION="Business value and ROI demonstration for executives"
RECORDING_NAME="guild-executive-demo"

main() {
    local mode="${1:-interactive}"
    
    recording_init "$RECORDING_NAME"
    
    if [[ "$mode" == "record" ]]; then
        {
            run_executive_demo
        } | start_recording "$RECORDING_NAME" "Guild Framework - Executive Demo"
        
        generate_gif "$RECORD_FILE" "${RECORDING_DIR}/guild-executive-demo.gif"
        
        echo
        print_success "Executive demo recorded successfully!"
        
    elif [[ "$mode" == "validate" ]]; then
        validate_recording_environment
        check_guild_providers
        print_success "Executive demo validation passed"
        
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

run_executive_demo() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    # Step 1: Business Problem Statement (20 seconds)
    print_title "The Development Productivity Challenge"
    echo
    print_warning "Traditional Development Challenges:"
    echo "  ❌ Developer productivity plateauing despite AI tools"
    echo "  ❌ Inconsistent code quality across teams"
    echo "  ❌ Long onboarding times for new developers"
    echo "  ❌ Context switching overhead in complex projects"
    echo "  ❌ Fragmented AI tool ecosystem"
    echo
    print_info "Industry stats: 40% of developer time spent on non-coding activities"
    print_info "Average onboarding: 6-12 months to full productivity"
    sleep 4
    
    # Step 2: Guild Solution Overview (30 seconds)
    print_title "Guild Framework: The Complete Solution"
    run_demo_command "guild init --name 'enterprise-demo' --description 'Enterprise development transformation'" \
        "Initialize enterprise-grade development environment"
    
    echo
    print_success "🏰 Guild Framework Delivers:"
    echo "  ✅ 10x Developer Productivity Improvement"
    echo "  ✅ Consistent Enterprise-Grade Code Quality"
    echo "  ✅ 90% Reduction in Onboarding Time"
    echo "  ✅ Unified Multi-Agent Development Platform"
    echo "  ✅ Real-Time Project Intelligence"
    sleep 3
    
    # Step 3: ROI Demonstration (45 seconds)
    print_title "Measurable Business Impact"
    
    # Create a business-focused commission
    cat > business-commission.md << 'EOF'
# Customer Portal Modernization
Transform legacy customer portal into modern cloud-native application

## Business Objectives
- Increase customer satisfaction scores by 40%
- Reduce support tickets by 60% 
- Enable 50% faster feature delivery
- Support 10x user growth
- Improve mobile experience ratings to 4.5+ stars

## Timeline: Traditional vs Guild
- Traditional approach: 6-9 months, 8 developers
- Guild approach: 2-3 months, 3 developers + Guild

## Expected ROI
- Development cost savings: $800K-1.2M
- Time to market advantage: 3-6 months
- Quality improvement: 85% fewer post-launch bugs
EOF

    run_demo_command "cat business-commission.md" \
        "Real business project specification"
    
    run_demo_command "guild business-analysis business-commission.md" \
        "AI-powered business impact analysis"
    
    echo
    print_success "📊 Projected ROI Analysis:"
    echo "  💰 Development Cost Savings: $800K - $1.2M"
    echo "  ⏱️  Time to Market: 3-6 months faster"
    echo "  🎯 Quality Improvement: 85% fewer bugs"
    echo "  👥 Team Efficiency: 3 developers vs 8 traditional"
    echo "  📈 Productivity Gain: 300-400% improvement"
    sleep 4
    
    # Step 4: Competitive Advantage (35 seconds)
    print_title "Competitive Market Position"
    run_demo_command "guild commission -f business-commission.md --track-metrics" \
        "Start development with real-time metrics tracking"
    
    echo
    print_title "🏆 Guild vs Traditional AI Development Tools"
    echo
    echo "┌─────────────────────┬─────────────────┬─────────────────────┐"
    echo "│ Feature             │ Guild Framework │ Traditional Tools   │"
    echo "├─────────────────────┼─────────────────┼─────────────────────┤"
    echo "│ Multi-Agent System  │ ✅ Built-in      │ ❌ Manual assembly │"
    echo "│ Project Intelligence│ ✅ Full context  │ ❌ Limited scope   │"
    echo "│ Quality Assurance   │ ✅ Automated     │ ❌ Manual process  │"
    echo "│ Team Coordination   │ ✅ Real-time     │ ❌ Fragmented      │"
    echo "│ Enterprise Ready    │ ✅ Day 1         │ ❌ Months of setup │"
    echo "│ Learning Curve      │ ✅ Intuitive     │ ❌ Steep           │"
    echo "└─────────────────────┴─────────────────┴─────────────────────┘"
    sleep 4
    
    # Step 5: Risk Mitigation (25 seconds)
    print_title "Risk Mitigation and Reliability"
    run_demo_command "guild status --business-metrics" \
        "Real-time project health and risk indicators"
    
    echo
    print_success "🛡️ Enterprise Risk Management:"
    echo "  ✅ Predictable delivery timelines with AI planning"
    echo "  ✅ Consistent code quality across all projects"
    echo "  ✅ Automated compliance and security scanning"
    echo "  ✅ Real-time project visibility and control"
    echo "  ✅ Reduced dependency on individual developers"
    sleep 3
    
    # Step 6: Implementation Strategy (30 seconds)
    print_title "Implementation and Adoption Strategy"
    run_demo_command "guild onboarding-plan --team-size 12 --timeline 'Q2 2025'" \
        "Generate team adoption roadmap"
    
    echo
    print_info "📋 Recommended Implementation Phases:"
    echo "  Phase 1 (Month 1): Pilot team of 3-5 developers"
    echo "  Phase 2 (Month 2-3): Expand to full development teams"
    echo "  Phase 3 (Month 4-6): Organization-wide deployment"
    echo "  Phase 4 (Month 6+): Advanced customization and optimization"
    echo
    print_success "Expected payback period: 3-6 months"
    sleep 3
    
    # Step 7: Call to Action (20 seconds)
    print_title "Next Steps and Investment Decision"
    run_demo_command "guild roi-calculator --team-size 15 --project-complexity high" \
        "Calculate ROI for your specific organization"
    
    echo
    print_title "🚀 Ready to Transform Your Development Organization?"
    echo
    print_success "Immediate Actions:"
    echo "  1️⃣  Schedule pilot project with Guild Framework"
    echo "  2️⃣  Identify high-impact development initiative"
    echo "  3️⃣  Measure baseline metrics for comparison"
    echo "  4️⃣  Begin team training and adoption program"
    echo
    print_info "Contact: guild-sales@company.com"
    print_info "Schedule demo: https://guild.company/demo"
    echo
    print_title "The future of development is multi-agent. The future is Guild."
    
    sleep 3
}

show_demo_preview() {
    show_demo_banner "$DEMO_NAME" "$DEMO_DESCRIPTION"
    
    print_info "This executive demo showcases:"
    echo "  • Clear business problem definition and market opportunity"
    echo "  • Quantifiable ROI and productivity improvements"
    echo "  • Competitive advantages and market differentiation"
    echo "  • Risk mitigation and reliability assurances"
    echo "  • Implementation strategy and adoption roadmap"
    echo "  • Call to action with next steps"
    echo
    print_warning "Target Audience: CTOs, VPs of Engineering, Technical Executives"
    print_info "Focus: Business value, ROI, competitive advantage, implementation"
    echo
    print_title "Key Business Messages:"
    echo "  • 10x productivity improvement with measurable ROI"
    echo "  • Competitive advantage through advanced AI coordination"
    echo "  • Risk mitigation through predictable, reliable delivery"
    echo "  • Clear implementation path with proven results"
    echo "  • Future-proofing development organization"
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
        echo "Guild Executive Business Demo"
        echo ""
        echo "Business value and ROI demonstration for executives"
        echo "Shows measurable impact, competitive advantages, and implementation strategy"
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