# Agent Configuration Guide

## Overview

Guild agents are configured through YAML files that define their capabilities, personality, tools, and behavior. This guide covers comprehensive agent configuration for the Guild Framework.

## Agent Configuration Structure

### Basic Agent Configuration

```yaml
# Required fields
id: agent-unique-id
name: Human Readable Name
type: manager|worker|specialist
provider: anthropic|openai|ollama|claudecode
model: model-name
capabilities:
  - capability-1
  - capability-2

# Optional fields  
description: "Agent description"
tools:
  - tool-1
  - tool-2
max_tokens: 4096
temperature: 0.7
cost_magnitude: 1-8
context_window: 200000
context_reset: truncate|summarize
```

### Agent Types

#### Manager Agents
**Purpose:** Strategic planning, coordination, task delegation
**Characteristics:**
- High-level decision making
- Team coordination
- Resource allocation
- Project planning

```yaml
type: manager
capabilities:
  - project-planning
  - team-coordination
  - resource-allocation
  - strategic-planning
context_reset: summarize  # Preserve context for continuity
```

#### Worker Agents  
**Purpose:** Implementation, execution, specialized tasks
**Characteristics:**
- Technical implementation
- Problem solving
- Task completion
- Specialized skills

```yaml
type: worker
capabilities:
  - backend-development
  - frontend-development
  - api-development
context_reset: truncate  # Can restart fresh for new tasks
```

#### Specialist Agents
**Purpose:** Domain expertise, specialized knowledge
**Characteristics:**
- Deep domain knowledge
- Specialized tools
- Expert consultation
- Quality assurance

```yaml
type: specialist
capabilities:
  - security-audit
  - performance-optimization
  - quality-assurance
context_reset: summarize  # Preserve domain knowledge
```

## Capability System

### Core Capabilities

```yaml
# Development Capabilities
capabilities:
  - backend-development
  - frontend-development
  - full-stack-development
  - api-development
  - database-design
  - cloud-architecture
  
# Project Management
  - project-planning
  - team-coordination
  - resource-allocation
  - strategic-planning
  - risk-management
  
# Quality Assurance
  - test-automation
  - quality-assurance
  - performance-testing
  - security-testing
  - code-review
  
# Specialized Domains
  - devops-automation
  - security-audit
  - data-analysis
  - ui-ux-design
```

### Language-Specific Capabilities

```yaml
# Go Development
capabilities:
  - go-development
  - go-performance-optimization
  - go-concurrency
  - go-microservices

# JavaScript Development  
capabilities:
  - javascript-development
  - typescript-development
  - react-development
  - node-development

# Python Development
capabilities:
  - python-development
  - django-development
  - fastapi-development
  - data-science
```

## Tool Access Control

### Tool Configuration Options

```yaml
# Allow all tools (empty list)
tools: []

# Explicit allow list
tools:
  - file
  - git
  - shell
  - http

# Specialized tools
tools:
  - code-analyzer
  - docker
  - kubectl
  - database-cli
```

### Tool Categories

#### File System Tools
```yaml
tools:
  - file        # File read/write operations
  - glob        # Pattern matching
  - grep        # Text search
```

#### Development Tools
```yaml
tools:
  - git         # Version control
  - shell       # Command execution
  - code-analyzer # Static analysis
  - test-runner   # Test execution
```

#### Infrastructure Tools
```yaml
tools:
  - docker      # Container operations
  - kubectl     # Kubernetes management
  - terraform   # Infrastructure as code
  - ansible     # Configuration management
```

#### Communication Tools
```yaml
tools:
  - http        # HTTP requests
  - grpc        # gRPC communication
  - websocket   # Real-time communication
```

## Provider Configuration

### Anthropic (Claude)

```yaml
provider: anthropic
model: claude-3-sonnet-20240229
# Alternative models:
# - claude-3-opus-20240229    # Most capable, highest cost
# - claude-3-haiku-20240307   # Fast, lower cost
cost_magnitude: 3  # Auto-detected for Claude Sonnet
context_window: 200000
```

### OpenAI (GPT)

```yaml
provider: openai
model: gpt-4
# Alternative models:
# - gpt-4-turbo               # Latest GPT-4 with larger context
# - gpt-3.5-turbo            # Faster, lower cost
cost_magnitude: 5   # Auto-detected for GPT-4
context_window: 32000
```

### Local Models (Ollama)

```yaml
provider: ollama
model: llama2
# Alternative models:
# - codellama                 # Code-specialized
# - mistral                   # General purpose
cost_magnitude: 0   # No API costs
context_window: 4096
```

### Claude Code Integration

```yaml
provider: claudecode
model: sonnet
cost_magnitude: 0   # Uses local Claude Code installation
context_window: 200000
```

## Advanced Configuration

### Cost Magnitude (Fibonacci Scale)

Cost magnitude controls intelligent agent selection based on task complexity:

```yaml
# Tool-only agents (no LLM calls)
cost_magnitude: 0

# Cheap API usage
cost_magnitude: 1

# Low-mid cost (GPT-3.5, Claude Haiku)
cost_magnitude: 2

# Mid cost (Claude Sonnet)
cost_magnitude: 3

# High cost (GPT-4)
cost_magnitude: 5

# Most expensive (Claude Opus)
cost_magnitude: 8
```

### Context Management

```yaml
# Context window size (tokens)
context_window: 200000  # Use 0 for auto-detection

# Context reset behavior when window exceeded
context_reset: summarize  # Preserve important context
# or
context_reset: truncate   # Start fresh
```

### Performance Tuning

```yaml
# Token limits
max_tokens: 4096  # Response length limit

# Creativity control
temperature: 0.7  # 0.0 = deterministic, 1.0 = creative

# Custom settings per provider
settings:
  top_p: 0.9
  frequency_penalty: 0.0
  presence_penalty: 0.0
```

## Personality and Backstory

### Personality Configuration

```yaml
personality:
  # Communication style
  formality: formal|casual|adaptive
  detail_level: concise|detailed|adaptive
  humor_level: none|occasional|frequent
  
  # Working style
  approach_style: methodical|creative|balanced
  risk_tolerance: conservative|moderate|aggressive
  decision_making: data-driven|intuitive|hybrid
  
  # Interaction patterns (1-10 scale)
  assertiveness: 7
  empathy: 8
  patience: 9
  
  # Medieval personality traits (1-10 scale)
  honor: 9
  wisdom: 8
  craftsmanship: 10
  
  # Specific traits
  traits:
    - name: analytical
      strength: 0.9
      description: "Methodical problem solver"
    - name: collaborative
      strength: 0.85
      description: "Works well with teams"
```

### Backstory Configuration

```yaml
backstory:
  # Professional background
  experience: "15 years in distributed systems"
  previous_roles:
    - "CTO at startup"
    - "Google SRE"
  expertise: "Microservices and cloud architecture"
  achievements:
    - "Led 20+ successful product launches"
    - "Reduced system latency by 70%"
  
  # Personal touches
  philosophy: "Simple solutions to complex problems"
  interests:
    - "Performance engineering"
    - "Team mentorship"
  background: "Computer Science PhD, distributed systems focus"
  
  # Communication style
  communication_style: "Direct and technical, but supportive"
  teaching_style: "Hands-on with real examples"
  
  # Medieval guild identity
  guild_rank: "Master Artisan"
  specialties:
    - "Performance Optimization"
    - "System Design"
```

### Specialization Configuration

```yaml
specialization:
  # Domain expertise
  domain: fintech
  sub_domains:
    - payment-processing
    - regulatory-compliance
  expertise_level: expert  # novice|intermediate|expert|master
  
  # Knowledge areas
  core_knowledge:
    - "PCI DSS compliance"
    - "Payment gateway integration"
    - "Financial regulations"
  familiar:
    - "Cryptocurrency protocols"
    - "Banking APIs"
  learning:
    - "Central Bank Digital Currencies"
    - "DeFi protocols"
  
  # Preferred approaches
  methodologies:
    - "Agile development"
    - "Test-driven development"
    - "Domain-driven design"
  technologies:
    - "Go, Python, PostgreSQL"
    - "Kubernetes, Docker"
    - "Payment APIs, Banking protocols"
  principles:
    - "Security by design"
    - "Regulatory compliance first"
    - "Audit trail everything"
  
  # Medieval specialization
  craft: "Financial Engineering"
  tools:
    - "Security analysis frameworks"
    - "Compliance validation tools"
  materials:
    - "Financial data"
    - "Regulatory requirements"
    - "Security policies"
```

## Configuration Examples

### Elena - Project Manager

```yaml
id: elena-guild-master
name: Elena Guild Master
type: manager
provider: anthropic
model: claude-3-sonnet-20240229
description: "Strategic project coordinator and team leader"

capabilities:
  - project-planning
  - team-coordination
  - resource-allocation
  - strategic-planning
  - stakeholder-communication

tools:
  - file
  - git
  - http
  - project-planner
  - kanban-board

cost_magnitude: 3
context_reset: summarize
temperature: 0.7

personality:
  formality: adaptive
  assertiveness: 8
  empathy: 9
  approach_style: methodical

backstory:
  experience: "15 years in project management"
  guild_rank: "Master Coordinator"
  philosophy: "Clear communication enables extraordinary results"
```

### Marcus - Backend Developer

```yaml
id: marcus-developer
name: Marcus Developer
type: worker
provider: openai
model: gpt-4
description: "Senior backend engineer and cloud architect"

capabilities:
  - backend-development
  - cloud-architecture
  - api-development
  - database-design
  - performance-optimization

tools:
  - file
  - git
  - shell
  - docker
  - kubectl

languages:
  - go
  - python
  - sql

cost_magnitude: 5
context_reset: truncate
temperature: 0.3

personality:
  formality: casual
  assertiveness: 7
  craftsmanship: 10
  approach_style: methodical

backstory:
  experience: "12 years in software development"
  guild_rank: "Master Craftsman"
  expertise: "Distributed systems and high-performance backends"
```

### Vera - QA Specialist

```yaml
id: vera-tester
name: Vera Tester
type: specialist
provider: anthropic
model: claude-3-haiku
description: "Quality assurance specialist and test engineer"

capabilities:
  - test-automation
  - quality-assurance
  - performance-testing
  - security-testing

tools:
  - file
  - git
  - test-runner
  - browser-automation
  - load-tester

cost_magnitude: 1
context_reset: truncate
temperature: 0.2

personality:
  formality: formal
  assertiveness: 6
  patience: 10
  approach_style: methodical

backstory:
  experience: "10 years in quality assurance"
  guild_rank: "Master Inspector"
  philosophy: "Quality is built in, not bolted on"
```

## Validation and Testing

### Configuration Validation

```bash
# Validate agent configuration
guild config validate

# Test specific agent
guild agent test elena-guild-master

# Validate all agents
guild agent validate-all
```

### Agent Testing

```yaml
# Test configuration in agent YAML
test_scenarios:
  - name: "greeting_test"
    input: "Hello, can you help me?"
    expected_response_contains: ["help", "assist"]
  
  - name: "capability_test"
    input: "What can you do?"
    expected_response_contains: ["capabilities", "specializ"]
```

## Best Practices

### Agent Design Principles

1. **Single Responsibility**: Each agent should have a clear, focused role
2. **Appropriate Scope**: Match agent capabilities to actual needs
3. **Cost Awareness**: Use cost_magnitude to control expensive model usage
4. **Tool Security**: Only grant necessary tool access
5. **Context Management**: Choose appropriate context reset strategy

### Configuration Management

1. **Version Control**: Include `.campaign/` in git repository
2. **Documentation**: Document custom agent modifications
3. **Validation**: Regular configuration validation
4. **Backup**: Backup agent configurations before major changes
5. **Testing**: Test agent configurations in safe environments

### Performance Optimization

1. **Model Selection**: Choose appropriate models for task complexity
2. **Context Window**: Optimize for your use case
3. **Temperature**: Lower for deterministic, higher for creative tasks
4. **Tool Access**: Minimize tool sets for security and performance
5. **Cost Management**: Use cost_magnitude for intelligent selection