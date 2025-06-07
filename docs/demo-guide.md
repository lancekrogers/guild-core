# Guild Framework Demo Guide

## Overview

This guide provides comprehensive instructions for demonstrating the Guild Framework's multi-agent AI capabilities using the e-commerce platform example. The demos showcase how specialized AI agents work together to build complex software systems.

## Pre-Demo Setup

### Terminal Configuration

1. **Terminal Size**:
   - Width: 120-140 characters
   - Height: 40-50 lines
   - Font: Use a clear monospace font (SF Mono, Fira Code, etc.)

2. **Color Theme**:
   - Recommended: Monokai, Dracula, or Nord
   - Ensure good contrast for recording

3. **Multiple Terminals**:
   - Prepare 2-3 terminal windows/tabs
   - One for `guild campaign watch`
   - One for `guild chat`
   - One for showing output/logs

### Environment Setup

```bash
# Set up environment variables
export OPENAI_API_KEY="your-api-key"
export GUILD_CONFIG="examples/config/e-commerce-guild.yaml"

# Initialize Guild project (if not already done)
cd /path/to/demo/directory
guild init

# Verify setup
guild info
```

### Pre-Demo Checklist

- [ ] API keys configured and working
- [ ] Guild binary in PATH
- [ ] E-commerce commission file present
- [ ] Agent configuration loaded
- [ ] Terminal properly sized
- [ ] Recording software ready (if recording)

## Demo Scenarios

### Scenario 1: Commission Refinement (2-3 minutes)

**Purpose**: Show AI-powered project planning and task breakdown

**Key Points**:
- Guild understands complex project requirements
- AI automatically identifies needed specialists
- Tasks are intelligently distributed
- Realistic effort estimates provided

**Script**:
```bash
./examples/demo-scripts/scenario-1-commission-refinement.sh
```

**Talking Points**:
- "Guild analyzes the entire project specification"
- "It identifies that we need 6 different specialists"
- "Each agent has specific expertise areas"
- "Tasks are automatically assigned based on agent capabilities"

### Scenario 2: Multi-Agent Coordination (3-4 minutes)

**Purpose**: Demonstrate agents working together in real-time

**Setup**: Open two terminals side by side

**Terminal 1**:
```bash
guild campaign watch e-commerce
```

**Terminal 2**:
```bash
guild chat --campaign e-commerce
```

**Demo Commands**:
```
Create the user authentication API with JWT tokens and OAuth2 support
```

**Key Points**:
- Multiple agents activate based on the request
- Real-time status updates in the monitor
- Agents work in parallel when possible
- Each contributes their expertise

**Talking Points**:
- "Notice how multiple agents start thinking"
- "ServiceArchitect handles API design"
- "GatewayGuardian configures security"
- "DeploymentMarshal prepares containers"
- "They coordinate without stepping on each other"

### Scenario 3: API Development Deep Dive (4-5 minutes)

**Purpose**: Show depth of agent expertise

**Demo Command**:
```
@service-architect Design a complete REST API for the product catalog with search, filtering, and pagination
```

**Key Points**:
- Production-quality code generation
- Best practices automatically applied
- Database optimization included
- Complete with documentation

**Talking Points**:
- "ServiceArchitect provides enterprise-grade solutions"
- "Notice the attention to performance"
- "Includes database indices for search"
- "OpenAPI documentation generated automatically"

### Scenario 4: Security-First Payment Integration (3-4 minutes)

**Purpose**: Demonstrate specialized security expertise

**Demo Command**:
```
@payment-sentinel Integrate Stripe payment processing with webhook handling and PCI compliance
```

**Key Points**:
- Security is built-in, not bolted on
- PCI compliance requirements met
- Comprehensive error handling
- Audit logging included

**Talking Points**:
- "PaymentSentinel prioritizes security"
- "Never stores sensitive card data"
- "Implements webhook signature verification"
- "Includes fraud detection logic"

### Scenario 5: Production Deployment (4-5 minutes)

**Purpose**: Show DevOps and infrastructure expertise

**Demo Command**:
```
@deployment-marshal Create production Kubernetes deployment with monitoring and auto-scaling
```

**Key Points**:
- Production-ready configurations
- Monitoring built-in from start
- Auto-scaling based on metrics
- Zero-downtime deployment strategy

**Talking Points**:
- "DeploymentMarshal creates battle-tested configs"
- "Includes health checks and graceful shutdown"
- "Prometheus metrics exposed automatically"
- "Horizontal pod autoscaling configured"

## Demo Flow Best Practices

### Pacing

1. **Introduction** (30 seconds)
   - Brief explanation of Guild Framework
   - Mention multi-agent architecture
   - Set expectations

2. **Demo Execution** (2-5 minutes per scenario)
   - Run commands deliberately
   - Pause to highlight key outputs
   - Explain what's happening

3. **Wrap-up** (30 seconds)
   - Summarize what was demonstrated
   - Mention next steps
   - Invite questions

### Common Questions and Answers

**Q: How do agents know when to activate?**
A: Agents monitor conversations and activate based on keywords, mentions, and task relevance to their expertise.

**Q: Can I customize agent behaviors?**
A: Yes, agents are configured through YAML files and prompt templates. You can adjust their expertise, tools, and behavior.

**Q: How does this compare to single-agent tools?**
A: Guild's multi-agent approach allows for specialized expertise, parallel work, and more complex problem-solving than single-agent systems.

**Q: Is this just for e-commerce?**
A: No, this is just an example. Guild can be configured for any software development project with custom agents.

## Troubleshooting

### Common Issues

1. **Agents not responding**
   - Check API key configuration
   - Verify agent configuration is loaded
   - Check `guild campaign list` shows active campaign

2. **Slow responses**
   - Normal for complex tasks (10-30 seconds)
   - Can adjust provider timeout settings
   - Consider using faster models for demos

3. **Terminal formatting issues**
   - Ensure terminal width is sufficient
   - Check Unicode support
   - Try different color themes

### Demo Recovery

If something goes wrong:
1. Stay calm and explain it's a live demo
2. Use `guild chat --resume` to recover session
3. Have backup screenshots ready
4. Can restart with a simpler command

## Recording Demos

### Recommended Tools

1. **asciinema** - Terminal recording
   ```bash
   asciinema rec demo.cast
   # Run demo
   # Ctrl+D to stop
   ```

2. **agg** - Convert to GIF
   ```bash
   agg --theme monokai demo.cast demo.gif
   ```

3. **OBS Studio** - Full screen recording with audio

### Recording Tips

- Do a practice run first
- Clear terminal before starting
- Speak clearly and at moderate pace
- Edit out long pauses in post
- Keep total length under 10 minutes

## Post-Demo Resources

Direct interested parties to:
- GitHub repository
- Documentation site
- Example configurations
- Community Discord/Slack

Remember: The goal is to show how Guild makes complex development tasks manageable through intelligent agent coordination!
