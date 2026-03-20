#!/bin/bash
# Manual testing script for multi-agent orchestration

set -e

echo "=== Multi-Agent Orchestration Manual Test ==="
echo

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test directory
TEST_DIR=".guild_test_$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

echo -e "${YELLOW}1. Setting up test environment...${NC}"

# Create guild.yaml
cat > guild.yaml << 'EOF'
version: "1.0"
guild:
  name: "Test Guild"
  description: "Guild for testing multi-agent orchestration"

agents:
  - id: architect
    name: "System Architect"
    type: architect
    provider: anthropic
    model: claude-3-5-sonnet-20241022
    system_prompt: |
      You are a system architect who designs software systems.
      Break down requirements into clear technical specifications.
    capabilities:
      - design
      - architecture
      - planning

  - id: developer
    name: "Senior Developer"
    type: developer
    provider: openai
    model: gpt-4
    system_prompt: |
      You are a senior developer who implements software systems.
      Write clean, tested, production-ready code.
    capabilities:
      - code
      - implement
      - test

  - id: reviewer
    name: "Code Reviewer"
    type: reviewer
    provider: anthropic
    model: claude-3-5-sonnet-20241022
    system_prompt: |
      You are a code reviewer who ensures quality and best practices.
    capabilities:
      - review
      - quality
      - security
EOF

echo -e "${GREEN}✓ Created guild.yaml${NC}"

# Create test commission
mkdir -p .guild/objectives
cat > .guild/objectives/test_api_commission.md << 'EOF'
# User Management REST API

## Objective
Build a complete REST API for user management with the following requirements:

1. CRUD operations for users
2. Authentication using JWT
3. Role-based access control
4. Input validation
5. Error handling
6. OpenAPI documentation

## Technical Requirements
- Use Go with Gin framework
- PostgreSQL database
- Docker deployment
- Unit and integration tests
- CI/CD pipeline

## Deliverables
1. API implementation
2. Database schema
3. Docker configuration
4. Test suite
5. Documentation
EOF

echo -e "${GREEN}✓ Created test commission${NC}"

echo
echo -e "${YELLOW}2. Starting Guild server...${NC}"
echo "Run in terminal 1:"
echo -e "${GREEN}guild serve${NC}"
echo

echo -e "${YELLOW}3. Testing campaign creation...${NC}"
echo "Run in terminal 2:"
cat << 'EOF'
# Create campaign
guild campaign create "API Development" \
  --commission .guild/objectives/test_api_commission.md

# Start campaign (triggers orchestration)
guild campaign start <campaign-id>

# Watch task creation
guild task list --watch
EOF

echo
echo -e "${YELLOW}4. Monitoring event flow...${NC}"
echo "Watch the guild serve logs for:"
echo "- campaign.started event"
echo "- commission.process_requested event"
echo "- agent.discovered events"
echo "- task.created events"
echo "- task.assigned events"

echo
echo -e "${YELLOW}5. Verification checklist:${NC}"
cat << 'EOF'
□ Campaign created successfully
□ Commission loaded and refined by AI
□ Tasks extracted from commission
□ Tasks created in kanban board
□ Agents registered with dispatcher
□ Tasks assigned to appropriate agents
□ Event flow visible in logs

Expected flow:
1. Campaign Start
   └─> OrchestratorCampaignBridge
       └─> CommissionProcessorBridge
           └─> CommissionIntegrationService
               ├─> Commission Refinement (AI)
               ├─> Task Extraction (AI)
               ├─> Kanban Task Creation
               └─> Agent Assignment

6. Check results:
guild task list
guild campaign status <campaign-id>
EOF

echo
echo -e "${YELLOW}Cleanup:${NC}"
echo "rm -rf $TEST_DIR"