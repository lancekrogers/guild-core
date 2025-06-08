#!/bin/bash

# E-commerce Platform Quick Demo (2 minutes)
# This script demonstrates Guild's multi-agent coordination capabilities

set -e

echo "🏗️  Guild Framework E-commerce Demo"
echo "====================================="
echo

# Setup phase - Initialize guild project
echo "📋 Phase 1: Project Initialization"
echo "Setting up Guild project with e-commerce commission..."
echo

# Initialize guild project
guild init

# Copy commission file to the project
echo "📄 Copying e-commerce commission to project..."
cp ../../.guild/commissions/e-commerce-platform.md .guild/commissions/

# Copy guild configuration
echo "⚙️  Setting up agent configuration..."
cp ../../.guild/guild.yaml .guild/

echo "✅ Project initialized with 6 specialized agents"
echo "   - @service-architect (System Design & API Architecture)"
echo "   - @frontend-specialist (React & TypeScript)"
echo "   - @backend-specialist (Go & Database Integration)"
echo "   - @devops-specialist (Docker & Kubernetes)"
echo "   - @qa-specialist (Testing & Quality Assurance)"
echo "   - @documentation-specialist (Technical Writing)"
echo

# Commission refinement phase
echo "📋 Phase 2: Commission Refinement"
echo "Refining the e-commerce commission for specialized agents..."
echo

guild commission refine .guild/commissions/e-commerce-platform.md

echo "✅ Commission refined and ready for agent coordination"
echo

# Demo execution instructions
echo "📋 Phase 3: Demo Execution Commands"
echo "Run these commands to demonstrate multi-agent coordination:"
echo
echo "1. Start agent coordination:"
echo "   guild chat --campaign e-commerce"
echo
echo "2. Demonstrate agent specialization with these example commands:"
echo "   @service-architect Start by designing the API architecture for our e-commerce platform"
echo "   @frontend-specialist Create the React component structure for the product catalog"
echo "   @backend-specialist Implement the authentication service with JWT"
echo "   @devops-specialist Set up the Docker containerization strategy"
echo "   @qa-specialist Design the testing strategy for our microservices"
echo "   @documentation-specialist Create the API documentation structure"
echo
echo "3. Show multi-agent coordination:"
echo "   @frontend-specialist @backend-specialist Coordinate on user authentication implementation"
echo "   @devops-specialist @qa-specialist Work together on testing environment setup"
echo
echo "🎯 Expected Demo Outcomes:"
echo "- Rich markdown rendering of API designs and code"
echo "- Real-time agent status indicators during processing"
echo "- Specialized responses showing domain expertise"
echo "- Professional formatting with syntax highlighting"
echo "- Multi-agent coordination and handoffs"
echo
echo "📊 Demo Success Metrics:"
echo "- API documentation with proper OpenAPI formatting"
echo "- React components with TypeScript definitions"
echo "- Go microservice code with error handling"
echo "- Docker configurations with multi-stage builds"
echo "- Test suites with comprehensive coverage"
echo "- Technical documentation with diagrams"
echo
echo "🚀 Demo completed! Project ready for agent interaction."
