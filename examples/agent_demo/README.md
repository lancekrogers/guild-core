# Guild Agent System Demo

This demo showcases the integrated Guild agent system with context-aware operations and registry-based component discovery.

## What This Demo Shows

1. **Context System Integration**
   - Request tracing and debugging information
   - Component discovery through context
   - Cost tracking and resource management
   - Session management

2. **Registry-Based Architecture**
   - Dynamic component registration and discovery
   - Configuration-driven setup
   - Provider, agent, and tool management

3. **Context-Aware Agents**
   - Agents that discover providers through context
   - Automatic provider selection based on task type
   - Rich debugging and tracing information
   - Cost-aware operations

4. **Agent Types**
   - **Worker Agent**: General-purpose task execution
   - **Coding Agent**: Specialized for software development tasks
   - **Manager Agent**: Task coordination and planning

## Running the Demo

```bash
# From the guild-core directory
cd examples/agent_demo
go run main.go
```

## Expected Output

The demo will show:

1. **System Initialization**
   - Guild context creation with request/session IDs
   - Registry and configuration setup
   - Provider and agent registration

2. **Agent Operations**
   - Direct agent execution with context tracing
   - Different agent types handling appropriate tasks
   - Performance and cost tracking

3. **Context-Aware Routing**
   - Automatic agent selection based on task type
   - Provider routing based on agent capabilities
   - Context propagation through operation chains

4. **System Status**
   - Agent status and performance metrics
   - Cost tracking and budget management
   - Registry component inventory

## Key Features Demonstrated

### Context System
- **Request Tracing**: Every operation has unique request/span IDs
- **Component Discovery**: Agents find providers through context
- **Cost Management**: Budget tracking per session/request
- **Error Handling**: Rich error context for debugging

### Registry Integration
- **Dynamic Registration**: Components registered at runtime
- **Configuration-Driven**: YAML-based component setup
- **Type Safety**: Strongly-typed component access
- **Dependency Injection**: Context-based service location

### Agent Capabilities
- **Provider Selection**: Automatic provider choice based on task
- **Context Awareness**: Full tracing and debugging information
- **Capability Matching**: Task routing based on agent capabilities
- **Status Monitoring**: Real-time agent performance tracking

## Architecture Benefits

This integration provides:

1. **Easy Debugging**: Full request tracing with context information
2. **Flexible Component Wiring**: Registry-based dependency injection
3. **Cost Control**: Built-in budget and resource management
4. **Scalable Design**: Context-aware operations support concurrent execution
5. **Rich Monitoring**: Comprehensive system observability

## Integration Points

The demo shows how the Guild framework components integrate:

- **Context ↔ Registry**: Service discovery through context
- **Context ↔ Agents**: Request tracing and component access
- **Context ↔ Providers**: Automatic provider selection and routing
- **Registry ↔ Configuration**: YAML-driven component setup
- **Agents ↔ Providers**: Context-aware LLM operations

## Next Steps

This foundational system enables:

1. **Tool Integration**: Adding context-aware tools
2. **Memory System**: Persistent conversation and knowledge management
3. **Objective System**: Goal-oriented task orchestration
4. **UI Integration**: Terminal and web interfaces
5. **Monitoring**: Production observability and metrics

The context and registry systems provide the architectural foundation for all future Guild framework development.