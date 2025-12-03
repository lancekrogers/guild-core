# Guild Chat Interface Demo

## 🏰 Guild Chat - Your Portal to AI Agent Communication

The Guild Chat interface is the most complete feature in Guild. While the project has build errors in several packages, the chat functionality demonstrates the framework's potential.

### 🚀 Launch Guild Chat

```bash
# From your guild project directory
../bin/guild chat

# Note: The --campaign and --session flags may not be fully implemented
```

**Important**: You must have at least one API key configured:

```bash
export ANTHROPIC_API_KEY="your-key"
# or
export OPENAI_API_KEY="your-key"
```

### 💬 Chat Features Implemented

#### **Agent Communication**

While the chat interface displays agent personas, the actual multi-agent orchestration is not yet fully implemented. Currently, messages are handled by a single LLM provider.

#### **Slash Commands**

Some slash commands are defined but may not be fully functional:

```
/help                    # Show available commands (may work)
/exit                    # Exit chat (or Ctrl+C)
/clear                   # Clear chat history

# These may not be fully implemented:
/status                  # Show campaign and agent status
/agents                  # List all available agents
/prompt list             # Show active prompt layers
```

#### **Keyboard Shortcuts**

```
Ctrl+P    # Quick prompt layer view
Ctrl+A    # Quick agent list
Ctrl+S    # Quick status view
Ctrl+H    # Toggle help
Ctrl+C    # Exit chat
```

### 🎨 Medieval Guild Theming

The interface uses consistent Guild terminology:

- **Agents** → "Artisans" (Backend Craftsman, Frontend Artisan, etc.)
- **Chat** → "Guild Chat Chamber"
- **Commands** → "Guild Commands"
- **Status** → "Guild Status"

### 📋 Available Test Agents

Our demo configuration includes:

1. **Guild Master Architect** (`guild-master`)
   - Type: Manager
   - Capabilities: Architecture, planning, coordination

2. **Backend Code Craftsman** (`backend-craftsman`)
   - Type: Worker
   - Capabilities: Go, API design, databases, microservices

3. **Frontend User Interface Artisan** (`frontend-artisan`)
   - Type: Worker
   - Capabilities: JavaScript, React, CSS, UI design

4. **Payment Security Sentinel** (`payment-sentinel`)
   - Type: Specialist
   - Capabilities: Payment processing, security, compliance

5. **Deployment Infrastructure Marshal** (`deployment-marshal`)
   - Type: Specialist
   - Capabilities: Docker, Kubernetes, AWS, DevOps

6. **Quality Test Forge** (`test-forge`)
   - Type: Specialist
   - Capabilities: Testing, automation, quality assurance

### 🧠 Layered Prompt System Preview

The chat interface includes a preview of Guild's revolutionary layered prompt system:

```
📋 Platform Layer:  Safety guidelines, Guild ethics
🏰 Guild Layer:     Project-specific goals and coding standards
👷 Role Layer:      Agent-specific role definitions
📚 Domain Layer:    Project type specializations
👤 Session Layer:   User preferences and session context
💬 Turn Layer:      Ephemeral instructions for current interaction
```

### 🎯 What Makes This Special

1. **Visual Interface**: See your agents in a beautiful terminal UI
2. **Medieval Theming**: Memorable and cohesive experience
3. **Real-time Ready**: Architecture prepared for gRPC streaming
4. **Layered Prompts**: Preview of dynamic prompt management
5. **Agent Personas**: Each agent has distinct capabilities and roles

### ⚠️ Current Limitations

1. **Build Errors**: The project has build failures that prevent full functionality
2. **Single Agent**: While multiple agents are displayed, actual multi-agent orchestration is not working
3. **gRPC Issues**: The gRPC integration has interface mismatches preventing compilation
4. **Limited Commands**: Many slash commands are not fully implemented

### 🔮 Required for Full Functionality

To realize the chat interface's full potential, the following must be completed:

- Fix build errors in pkg/grpc (Campaign/Objectives interface mismatches)
- Complete multi-agent orchestration implementation
- Enable tool execution through gRPC
- Implement remaining slash commands

## 📋 Summary

The Guild Chat interface demonstrates strong potential with:

- ✅ Beautiful terminal UI with markdown rendering
- ✅ Medieval theming throughout
- ✅ Basic chat functionality with LLM providers
- ❌ Multi-agent orchestration (not working)
- ❌ Full command system (partially implemented)
- ❌ gRPC integration (build errors)

While impressive visually, the chat interface currently functions as a single-agent chat client rather than the envisioned multi-agent orchestration system.
