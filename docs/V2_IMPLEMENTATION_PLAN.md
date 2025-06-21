# Guild Chat V2 Implementation Plan

**Date**: June 21, 2025  
**Status**: V2 foundation exists, needs key integrations to replace V1  
**Scope**: Complete V2 to allow V1 removal (18,455 lines → modular V2)

## Current V2 Status

### ✅ Working Components
- **App Structure**: `app.go` - Complete Bubble Tea model ✅
- **Layout System**: `layout/` - Flexible pane management ✅
- **Service Layer**: `services/` - ChatService, DaemonService, ProviderService ✅
- **Basic Panes**: `panes/input.go`, `panes/output.go`, `panes/status.go` ✅
- **Session Management**: SQLite integration working ✅
- **Command Framework**: `commands.go` - CommandProcessor structure ✅
- **Build System**: V2 compiles successfully ✅
- **CLI Integration**: Loads via `GUILD_CHAT_V2=true` ✅

### ✅ COMPLETED Critical Components
1. **Agent Communication**: ✅ **IMPLEMENTED** - Full agent routing and @mention handling
2. **gRPC Integration**: ✅ **IMPLEMENTED** - Connected to guild daemon with real gRPC calls
3. **Command Implementation**: ✅ **IMPLEMENTED** - All V1 commands ported with full functionality
4. **Rich Content**: ⚠️ Command framework supports rich output, rendering integration needed

## Critical Integration Points

### 1. Agent Communication (HIGHEST PRIORITY)

**V1 Implementation** (spread across multiple files):
- `agent_indicators.go` - Real-time agent status
- `message_handler.go` - Agent message routing
- Chat integration with gRPC streaming

**V2 Needed**:
```go
// Create: internal/chat/v2/agents/router.go
type AgentRouter struct {
    guildClient pb.GuildClient
    agents map[string]*AgentInfo
}

func (ar *AgentRouter) RouteMessage(input string) (*AgentTarget, error)
func (ar *AgentRouter) ParseMention(input string) (string, string, error) // @agent message
func (ar *AgentRouter) BroadcastToAll(message string) tea.Cmd
```

### 2. Command System Integration

**V1 Implementation**: `chat_commands.go` - Complete slash command system

**V2 Status**: Framework exists, needs command implementations
```go
// Needed in commands.go:
func (cp *CommandProcessor) registerCommands() {
    cp.handlers["/help"] = &HelpCommand{}
    cp.handlers["/status"] = &StatusCommand{}
    cp.handlers["/agents"] = &AgentsCommand{}
    cp.handlers["/prompt"] = &PromptCommand{}
    // ... all V1 commands
}
```

### 3. gRPC Streaming Integration

**V1 Implementation**: Integrated throughout multiple files

**V2 Needed**: Connect existing services to actual gRPC calls
```go
// Enhance: internal/chat/v2/services/chat.go
func (cs *ChatService) SendMessageToAgent(agentID, message string) tea.Cmd
func (cs *ChatService) StartAgentStream(agentID string) tea.Cmd
func (cs *ChatService) HandleStreamResponse(msg pb.StreamMessage) tea.Cmd
```

## Implementation Strategy

### Phase 1: Agent Communication (Day 1 - 4 hours)

#### 1.1 Create Agent Router
```bash
# Create new file: internal/chat/v2/agents.go
```
**Implementation**:
- Extract agent routing logic from V1 `message_handler.go`
- Implement @mention parsing
- Connect to ChatService for gRPC calls

#### 1.2 Integrate with App.Update()
**Location**: `app.go` handleSubmit() method
**Changes**:
- Add agent routing call
- Connect to gRPC services
- Handle agent responses

### Phase 2: Command Implementation ✅ COMPLETED

#### 2.1 Implement Core Commands ✅ COMPLETED
**Location**: `commands.go` - 1,500+ lines of comprehensive command handlers
**Extracted from V1**: All `chat_commands.go` functionality replicated
- ✅ `/help` - Comprehensive help with all V1 commands
- ✅ `/status` - Detailed system and agent status display  
- ✅ `/agents` - Agent list with status icons and capabilities
- ✅ `/prompt` - Full 6-layer prompt management (list/get/set/delete)
- ✅ `/tools` - Complete tool management (list/search/info/status)
- ✅ `/test` - Rich content testing (markdown/code/mixed)
- ✅ `/export` - Session export with multiple formats
- ✅ `/guild` - Guild management and switching
- ✅ All V1 command aliases and variations

#### 2.2 Connect Command Processor ✅ COMPLETED  
**Location**: `app.go` handleSubmit() method with AgentRouter integration
**Implemented**:
- ✅ Route slash commands to CommandProcessor
- ✅ Handle @mentions for agent communication
- ✅ Broadcast messages to all agents
- ✅ Command responses via PaneUpdateMsg

### Phase 3: Rich Content (Day 2 - 3 hours)

#### 3.1 Message Rendering
**Extract from V1**: `content_formatter.go`, `markdown_renderer.go`
**Create**: `internal/chat/v2/content.go`
- Markdown parsing and rendering
- Code syntax highlighting
- Agent message styling

#### 3.2 Integrate with OutputPane
**Location**: `panes/output.go`
**Changes**:
- Add rich content rendering
- Support message formatting
- Handle code blocks and markdown

### Phase 4: Testing & Polish (Day 2 - 2 hours)

#### 4.1 Integration Testing
- Test V2 against existing daemon
- Verify feature parity with V1
- Test agent communication end-to-end

#### 4.2 Performance Optimization
- Ensure V2 startup ≤ V1 startup time
- Memory usage optimization
- Response latency verification

## Detailed File Plan

### New Files to Create

#### `internal/chat/v2/agents.go`
```go
// Agent routing and communication logic
type AgentRouter struct { ... }
func (ar *AgentRouter) RouteMessage(input string) (*AgentTarget, error)
func (ar *AgentRouter) ParseMention(input string) (agentID, message string, error)
func (ar *AgentRouter) GetAgentStatus(agentID string) (*pb.AgentStatus, error)
```

#### `internal/chat/v2/content.go`  
```go
// Rich content rendering (extracted from V1)
type ContentRenderer struct { ... }
func (cr *ContentRenderer) RenderMarkdown(content string) string
func (cr *ContentRenderer) HighlightCode(code, lang string) string
func (cr *ContentRenderer) FormatAgentMessage(msg AgentMessage) string
```

#### `internal/chat/v2/streaming.go`
```go
// gRPC streaming integration
type StreamManager struct { ... }
func (sm *StreamManager) StartAgentStream(agentID string) tea.Cmd
func (sm *StreamManager) HandleStreamMessage(msg pb.StreamMessage) tea.Cmd
func (sm *StreamManager) CloseStream(agentID string) tea.Cmd
```

### Files to Enhance

#### `app.go` - Main Integration Points
```go
// Add to handleSubmit():
func (app *App) handleSubmit() (tea.Model, tea.Cmd) {
    input := app.inputPane.GetValue()
    
    // 1. Check for agent mentions
    if agentTarget := app.agentRouter.ParseMention(input); agentTarget != nil {
        return app, app.chatService.SendToAgent(agentTarget.ID, agentTarget.Message)
    }
    
    // 2. Check for commands
    if isCommand, cmd := app.commandProcessor.ProcessInput(input); isCommand {
        return app, cmd
    }
    
    // 3. Default to broadcast
    return app, app.chatService.BroadcastToAll(input)
}
```

#### `services/chat.go` - gRPC Integration
```go
// Add real gRPC calls:
func (cs *ChatService) SendToAgent(agentID, message string) tea.Cmd {
    return func() tea.Msg {
        // Make actual gRPC call to guild daemon
        response, err := cs.client.SendMessage(ctx, &pb.SendMessageRequest{
            AgentId: agentID,
            Message: message,
        })
        return AgentResponseMsg{AgentID: agentID, Response: response, Error: err}
    }
}
```

#### `commands.go` - Command Implementation
```go
// Implement actual command handlers:
func (cp *CommandProcessor) registerCommands() {
    cp.handlers["/help"] = &HelpCommand{config: cp.config}
    cp.handlers["/status"] = &StatusCommand{services: cp.services}
    cp.handlers["/agents"] = &AgentsCommand{chatService: cp.chatService}
    cp.handlers["/prompt"] = &PromptCommand{promptClient: cp.promptClient}
}
```

## Success Criteria

### Functional Requirements
- [ ] **Agent Communication**: @mentions work identical to V1
- [ ] **Command System**: All V1 commands ported and working
- [ ] **Rich Content**: Markdown and code highlighting working
- [ ] **gRPC Integration**: Real-time streaming functional
- [ ] **Session Persistence**: Messages saved and restored

### Non-Functional Requirements  
- [ ] **Performance**: V2 ≤ 110% of V1 memory usage
- [ ] **Startup Time**: V2 ≤ V1 startup time
- [ ] **Modularity**: No file > 500 lines
- [ ] **Testability**: Each component unit testable

### Migration Requirements
- [ ] **Feature Parity**: 100% of V1 functionality
- [ ] **CLI Compatibility**: Drop-in replacement via environment variable
- [ ] **Configuration**: Same config format and options
- [ ] **User Experience**: Identical or better UX

## Risk Mitigation

### High-Risk Items
1. **gRPC Streaming**: Complex bidirectional streaming
   - **Mitigation**: Start with simple request/response, add streaming incrementally
   
2. **Agent State Management**: Complex agent status tracking
   - **Mitigation**: Extract exact logic from V1, don't reinvent

3. **Message Rendering**: Rich content display
   - **Mitigation**: Direct port from V1 content_formatter.go

### Testing Strategy
1. **Integration Testing**: Test against live daemon from Day 1
2. **Parallel Running**: Keep V1 as fallback during development
3. **Incremental Rollout**: Test V2 with individual features enabled

## Timeline Estimate

### Day 1 (6 hours total)
- **Morning (4 hours)**: Agent communication implementation
- **Afternoon (2 hours)**: Basic command system

### Day 2 (5 hours total)  
- **Morning (3 hours)**: Rich content rendering
- **Afternoon (2 hours)**: Integration testing and bug fixes

### Day 3 (2 hours total)
- **V1 Removal**: Delete V1 files, make V2 default
- **Documentation**: Update docs and examples

**Total Effort**: ~13 hours over 3 days

## Conclusion

V2 has an excellent architectural foundation. The main work is integrating existing V1 functionality into the V2 modular structure. This is primarily extraction and adaptation work rather than new feature development.

The modular V2 architecture will provide:
- **Better maintainability**: Components under 500 lines each
- **Improved testability**: Each module independently testable  
- **Enhanced extensibility**: Clear interfaces for future features
- **Performance benefits**: More efficient resource usage

Once complete, V2 will replace 18,455 lines of scattered V1 code with a clean, modular architecture while maintaining 100% functional parity.