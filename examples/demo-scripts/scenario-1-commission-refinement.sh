#!/bin/bash
# Scenario 1: Commission Refinement Process
# Duration: 2-3 minutes
# Purpose: Show how Guild AI refines a high-level commission into actionable tasks

echo "🏰 Guild Framework Demo - Scenario 1: Commission Refinement"
echo "=========================================================="
echo ""
echo "📋 Loading e-commerce platform commission..."
echo ""

# Set demo environment
export GUILD_CONFIG="examples/config/e-commerce-guild.yaml"

# Show the commission file
echo "📄 Commission Overview:"
head -n 20 examples/commissions/e-commerce-platform.md
echo ""
echo "... (full commission contains detailed requirements)"
echo ""
read -p "Press Enter to start refinement process..."

# Run commission refinement
echo ""
echo "🤖 Starting AI-powered commission refinement..."
echo "$ guild commission refine examples/commissions/e-commerce-platform.md"
echo ""

# Simulate refinement output (in real demo, this would be actual command)
cat << 'EOF'
🔄 Analyzing commission document...
✅ Identified 6 specialized agent roles needed
✅ Detected 8 major technical components
✅ Found 15 user stories to implement
✅ Extracted 12 non-functional requirements

📊 Agent Assignment Summary:
- ServiceArchitect: 5 tasks (API design, service communication)
- CatalogCraftsman: 4 tasks (product CRUD, search, inventory)
- PaymentSentinel: 3 tasks (Stripe integration, security, webhooks)
- OrderOrchestrator: 4 tasks (order flow, state management, notifications)
- GatewayGuardian: 3 tasks (authentication, rate limiting, routing)
- DeploymentMarshal: 3 tasks (Docker, Kubernetes, monitoring)

💾 Refined objectives saved to: .guild/objectives/refined/e-commerce-platform/

Would you like to view the detailed task breakdown? (y/n)
EOF

read -p "> " view_details

if [[ "$view_details" == "y" ]]; then
    echo ""
    echo "📋 Detailed Task Breakdown:"
    echo ""
    cat << 'EOF'
## ServiceArchitect Tasks

### Task 1: Design Authentication Service API
- Design RESTful endpoints for user registration, login, logout
- Define JWT token structure and refresh token flow
- Create OpenAPI specification
- Estimated effort: 2 hours

### Task 2: Design Product Catalog Service API
- Define CRUD endpoints for products
- Design search and filter query parameters
- Create gRPC service definitions for internal communication
- Estimated effort: 3 hours

[Additional tasks would be shown here...]
EOF
fi

echo ""
echo "✅ Commission refinement complete!"
echo ""
echo "💡 Next steps:"
echo "1. Review refined objectives in .guild/objectives/refined/"
echo "2. Start campaign with: guild campaign start e-commerce"
echo "3. Begin development with: guild chat --campaign e-commerce"
echo ""
echo "🎯 This refinement process demonstrates Guild's ability to:"
echo "- Parse complex project requirements"
echo "- Identify specialized agent roles needed"
echo "- Break down work into manageable tasks"
echo "- Assign tasks based on agent expertise"
