# Guild Chat Interface Demo

## 🏰 Guild Chat - Your Portal to AI Agent Communication

The Guild Chat interface is now **functional** and ready for use! Here's what you can do:

### 🚀 Launch Guild Chat

```bash
# Start the chat interface
./guild chat --campaign "e-commerce-demo"

# Or with a specific session
./guild chat --campaign "e-commerce-demo" --session "my-session-123"
```

### 💬 Chat Features Implemented

#### **Agent Communication**
```
# Message specific agents
@backend-craftsman design the user authentication API
@frontend-artisan create the login component
@payment-sentinel implement secure payment processing

# Broadcast to all agents
@all what's your current status?
```

#### **Slash Commands**
```
/help                    # Show all available commands
/status                  # Show campaign and agent status
/agents                  # List all available agents
/prompt list             # Show active prompt layers
/prompt get --layer role # View specific prompt layer
/exit                    # Exit chat (or Ctrl+C)
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

### 🔮 Coming Next

While we wait for the layered prompt system completion, the chat interface is ready for:
- gRPC integration for real agent communication
- Tool execution visualization
- Real-time task status updates
- Enhanced prompt management

## 🏆 Achievement Unlocked

**Guild Chat TUI: Phase 1 Complete!** ✅

Guild now has a **functional, beautiful chat interface** that showcases:
- Multi-agent communication patterns
- Command system architecture  
- Medieval Guild theming
- Layered prompt system foundation
- Professional terminal UI

This is **exactly** what makes Guild unique - no other AI agent framework has a visual chat interface like this!