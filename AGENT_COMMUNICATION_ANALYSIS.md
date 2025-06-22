# Agent Communication Analysis - Chat V2

## Current State

### 1. Architectural Mismatch
The chat V2 implementation has a fundamental disconnect between expected and actual gRPC methods:

**Expected by AgentRouter (agents.go):**
- `SendMessageToAgent(ctx, *AgentMessageRequest) (*AgentMessageResponse, error)`
- `ListAvailableAgents(ctx, *ListAgentsRequest) (*ListAgentsResponse, error)`
- `GetAgentStatus(ctx, *GetAgentStatusRequest) (*AgentStatus, error)`

**Actually Implemented in gRPC:**
- `Chat(stream ChatService_ChatServer) error` - Bidirectional streaming
- `CreateChatSession(ctx, *CreateChatSessionRequest) (*ChatSession, error)`
- `EndChatSession(ctx, *EndChatSessionRequest) (*EndChatSessionResponse, error)`

### 2. Component Issues

#### AgentRouter (`internal/ui/chat/agents/agents.go`)
- Expects methods that don't exist on the gRPC client
- Will fail at runtime when trying to send messages to agents
- `RefreshAgentList()` will fail with "method not found"

#### Commands (`internal/ui/chat/commands/commands.go`)
- `/agents` command returns hardcoded mock data
- `/status` command returns hardcoded mock data
- Agent mentions (@agent) are parsed but can't actually send messages

#### ChatService (`internal/ui/chat/services/chat.go`)
- Has mock implementations that don't connect to real agents
- `discoverAgents()` returns hardcoded list: ["developer", "writer", "researcher", "tester"]
- `sendToAgent()` returns fake responses

#### Status Pane (`internal/ui/chat/panes/status.go`)
- Can display agent status but never receives real updates
- Has methods like `SetAgentStatus()` but no real data flows to it

### 3. What IS Working
- The gRPC server has a proper `ChatService` implementation with bidirectional streaming
- The `ChatService` can manage sessions with agents
- Tool execution flow is implemented
- Registry pattern provides access to real agents

## Required Connections

### Option 1: Fix AgentRouter to Use ChatService (Recommended)
Instead of expecting non-existent methods, the AgentRouter should:
1. Create a chat session via `CreateChatSession`
2. Use the bidirectional `Chat` stream for communication
3. Handle responses through the streaming interface

### Option 2: Implement Missing gRPC Methods
Add the expected methods to the gRPC service:
1. Implement `SendMessageToAgent`, `ListAvailableAgents`, `GetAgentStatus`
2. These would be simple wrappers around the existing functionality

### Option 3: Bypass gRPC and Use Registry Directly
Since we're in the same process:
1. Access agents directly through the registry
2. Call agent.Execute() directly
3. More efficient but less flexible

## Implementation Plan

### Step 1: Update AgentRouter to Use ChatService
```go
// Instead of:
resp, err := ar.guildClient.SendMessageToAgent(ar.ctx, req)

// Use:
// 1. Create or reuse a chat session
// 2. Send message through Chat stream
// 3. Receive response through stream
```

### Step 2: Connect Real Agent Discovery
```go
// In ChatService.discoverAgents():
func (cs *ChatService) discoverAgents() ([]string, error) {
    if cs.registry != nil && cs.registry.Agents() != nil {
        return cs.registry.Agents().ListAgents(), nil
    }
    return []string{}, nil
}
```

### Step 3: Wire Up Real Commands
Update command handlers to use actual services instead of mock data.

### Step 4: Connect Status Updates
Route agent status changes through the event system to update the status pane.

## Quick Fix for Demo

For immediate functionality, implement the missing gRPC methods as thin wrappers:

```go
// In grpc/server.go
func (s *Server) SendMessageToAgent(ctx context.Context, req *pb.AgentMessageRequest) (*pb.AgentMessageResponse, error) {
    // Get agent from registry
    agent, err := s.agentReg.GetAgent(req.AgentId)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "agent not found: %v", err)
    }
    
    // Execute agent
    response, err := agent.Execute(ctx, req.Message)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "agent execution failed: %v", err)
    }
    
    return &pb.AgentMessageResponse{
        Response: response,
    }, nil
}

func (s *Server) ListAvailableAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
    agentIDs := s.agentReg.ListAgents()
    agents := make([]*pb.Agent, 0, len(agentIDs))
    
    for _, id := range agentIDs {
        agent, err := s.agentReg.GetAgent(id)
        if err != nil {
            continue
        }
        agents = append(agents, &pb.Agent{
            Id:   id,
            Name: agent.GetName(),
            Type: agent.GetType(),
            // Capabilities would need to be added to agent interface
        })
    }
    
    return &pb.ListAgentsResponse{
        Agents: agents,
    }, nil
}

func (s *Server) GetAgentStatus(ctx context.Context, req *pb.GetAgentStatusRequest) (*pb.AgentStatus, error) {
    // This would need agent status tracking to be implemented
    return &pb.AgentStatus{
        AgentId:      req.AgentId,
        State:        pb.AgentStatus_IDLE,
        LastActivity: time.Now().Unix(),
    }, nil
}
```

## Testing the Fix

1. Start the guild daemon
2. Run the chat interface
3. Test commands:
   - `/agents` - Should show real agents
   - `/status` - Should show real status
   - `@developer hello` - Should get real response
   - `@all hello` - Should broadcast to all agents

## Next Steps

1. Decide on approach (Option 1 recommended)
2. Implement the chosen solution
3. Add proper error handling
4. Test with real agents
5. Update documentation