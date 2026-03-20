# Enhanced Guild Initialization Process

## Overview

The Guild Framework provides a streamlined initialization process that automatically detects your project type and sets up appropriate agent configurations. This guide covers the new enhanced init process introduced in development phase

## Quick Start (30 Seconds to Productive)

```bash
# Fast installation (no go vet delays)
make install

# Initialize workspace with Elena agent
guild init my-project
cd my-project

# Set API key and start chatting
export ANTHROPIC_API_KEY="your-key"
guild chat

# Optional: Advanced configuration
guild setup-wizard
```

## Project Initialization

### Basic Initialization

```bash
# Initialize in current directory
guild init

# Initialize with specific project name
guild init my-project

# Force reinitialize existing project
guild init --force
```

### Campaign Structure

After initialization, you'll have this structure:

```
my-project/
├── .campaign/                    # Guild configuration directory
│   ├── campaign.yaml            # Campaign metadata
│   ├── memory.db               # SQLite database for persistence
│   ├── agents/                 # Agent configurations
│   │   ├── elena-guild-master.yaml
│   │   ├── marcus-developer.yaml (if Go/JS/Python project)
│   │   └── vera-tester.yaml
│   └── guilds/                 # Guild definitions (optional)
│       └── main-guild.yaml
├── commissions/               # Commission files (objectives)
│   └── .gitkeep
└── [your project files]
```

## Project Type Detection

Guild automatically detects your project type and configures appropriate agents:

### Go Projects

**Detected by:** `go.mod`, `*.go` files
**Agents configured:**

- Elena (Guild Master) - Project coordination
- Marcus (Developer) - Go development specialist
- Vera (Tester) - Testing and QA

**Example agent specialization:**

```yaml
# marcus-developer.yaml
languages:
  - go
capabilities:
  - backend-development
  - api-development
  - performance-optimization
tools:
  - file
  - git
  - shell
  - go-test
```

### JavaScript/Node.js Projects

**Detected by:** `package.json`, `*.js`, `*.ts` files
**Agents configured:**

- Elena (Guild Master) - Project coordination
- Marcus (Developer) - Full-stack JavaScript specialist
- Vera (Tester) - Frontend/backend testing

### Python Projects

**Detected by:** `requirements.txt`, `pyproject.toml`, `*.py` files
**Agents configured:**

- Elena (Guild Master) - Project coordination
- Marcus (Developer) - Python backend specialist
- Vera (Tester) - Python testing specialist

### Generic Projects

**For any other project type:**

- Elena (Guild Master) - General project coordination
- Additional agents can be added via `guild setup-wizard`

## Campaign Configuration

### Campaign Metadata (`campaign.yaml`)

```yaml
name: my-project
description: "Project description"
version: "1.0.0"

# Guild configuration
guilds:
  - main-guild

settings:
  default_guild: main-guild
  
# Project-specific settings
project:
  type: go  # auto-detected
  languages:
    - go
  frameworks:
    - gin
    - gorm
```

### Guild Configuration (`guilds/main-guild.yaml`)

```yaml
name: Main Development Guild
description: "Primary development team for the project"
purpose: "Full-stack development and delivery"

# Guild members
agents:
  - elena-guild-master
  - marcus-developer
  - vera-tester

# Coordination settings
coordination:
  max_parallel_tasks: 3
  review_required: true
  auto_handoff: true

# Guild manager
manager: elena-guild-master
```

## Agent Configuration Examples

### Elena - Guild Master (Manager)

Elena is automatically configured as your project coordinator:

```yaml
id: elena-guild-master
name: Elena Guild Master
type: manager
provider: anthropic
model: claude-3-sonnet-20240229

capabilities:
  - project-planning
  - team-coordination
  - resource-allocation
  - strategic-planning

personality:
  formality: adaptive
  assertiveness: 8
  empathy: 9
  
backstory:
  experience: "15 years in project management"
  guild_rank: "Master Coordinator"
```

### Marcus - Developer (Worker)

Marcus is configured based on your project type:

```yaml
id: marcus-developer
name: Marcus Developer
type: worker
provider: openai
model: gpt-4

# Adapts based on project detection
capabilities:
  - backend-development    # Always included
  - go-development        # Added for Go projects
  - javascript-development # Added for JS projects
  - python-development    # Added for Python projects
  
tools:
  - file
  - git
  - shell
  # Project-specific tools added automatically
```

## First Chat Experience

After initialization, start chatting immediately:

```bash
guild chat
```

### Sample First Interactions

**Elena (Project Planning):**

```
You: "Hello Elena, I want to build a REST API for user management"

Elena: "Hello! I'm Elena, your guild master. A user management REST API is a great project! Let me help you plan this systematically. 

I suggest we break this down into these phases:
1. Database design for user entities
2. API endpoint design and documentation  
3. Authentication and authorization
4. Implementation and testing
5. Deployment and monitoring

Marcus can handle the Go implementation, and Vera will ensure comprehensive testing. Shall we start with requirements gathering?"
```

**Marcus (Technical Implementation):**

```
You: "@marcus I need help implementing JWT authentication"

Marcus: "Hey! JWT authentication is definitely something I can help with. For a Go project, I recommend using the `golang-jwt/jwt` library.

Here's what we'll need:
1. JWT middleware for protected routes
2. Token generation on login
3. Token validation and refresh
4. Secure key management

Want me to implement a secure JWT setup with proper error handling and token rotation?"
```

**Vera (Quality Assurance):**

```
You: "@vera please test the authentication system"

Vera: "Good day. I shall establish comprehensive testing for the authentication system. My testing approach will include:

1. Unit tests for JWT token generation and validation
2. Integration tests for protected endpoints
3. Security tests for token manipulation attempts
4. Load tests for authentication performance
5. Edge case validation (expired tokens, malformed requests)

I recommend establishing these tests before implementation to ensure quality standards are met."
```

## Advanced Configuration

### Setup Wizard

For advanced configuration options:

```bash
guild setup-wizard
```

The wizard will guide you through:

- Custom agent selection
- Provider configuration (API keys, models)
- Tool access controls
- Performance settings
- Integration options

### Manual Agent Addition

Add specialized agents for specific needs:

```bash
# Add a security specialist
guild agent add security-specialist --type specialist --capabilities security-audit,penetration-testing

# Add a DevOps specialist  
guild agent add devops-specialist --type specialist --capabilities deployment,monitoring,infrastructure
```

## Troubleshooting

### Common Issues

**Issue: Init fails with permission errors**

```bash
# Check directory permissions
ls -la
# Ensure you have write access to current directory
```

**Issue: Project type not detected correctly**

```bash
# Force specific project type
guild init --type go
guild init --type javascript
guild init --type python
```

**Issue: Missing dependencies**

```bash
# Reinstall Guild
make install
# Or full development setup
make build
```

### Validation

Verify your initialization was successful:

```bash
# Check project status
guild status

# List available agents
guild agent list

# Test configuration
guild config validate
```

## Next Steps

1. **Start Development**: Use `guild chat` to begin working with your agents
2. **Create Commissions**: Define your project goals in `commissions/` directory
3. **Customize Agents**: Modify agent configurations in `.campaign/agents/`
4. **Add Tools**: Enable additional tools via `guild setup-wizard`
5. **Monitor Progress**: Use `guild status` to track project progress

## Best Practices

### Project Organization

- Keep commission files focused on specific features
- Use descriptive names for commissions and tasks
- Regular status checks with `guild status`

### Agent Utilization

- Use Elena for planning and coordination
- Direct technical questions to Marcus
- Involve Vera early for testing strategy

### Configuration Management

- Version control your `.campaign/` directory
- Document custom agent modifications
- Regular backups of `memory.db` for important projects

## Performance Targets

The enhanced init process meets these performance goals:

- ✅ **Init time < 2 seconds** for most projects
- ✅ **Memory usage < 100MB** during initialization
- ✅ **Agent response time < 500ms** for basic queries
- ✅ **Session persistence** survives restarts
