# Guild gRPC API Reference

## Overview

The Guild daemon exposes gRPC services for real-time chat, session management, and event streaming. All services use Protocol Buffers v3 for message serialization and support both streaming and unary RPC patterns.

## Service Definitions

### ChatService

Interactive bidirectional communication with Guild agents.

```protobuf
service ChatService {
  // Interactive bidirectional chat with agents
  rpc Chat(stream ChatRequest) returns (stream ChatResponse);

  // Get chat history for a session
  rpc GetChatHistory(GetChatHistoryRequest) returns (GetChatHistoryResponse);

  // Create a new chat session
  rpc CreateChatSession(CreateChatSessionRequest) returns (ChatSession);

  // End a chat session
  rpc EndChatSession(EndChatSessionRequest) returns (EndChatSessionResponse);

  // List active chat sessions
  rpc ListChatSessions(ListChatSessionsRequest) returns (ListChatSessionsResponse);
}
```

## Message Types

### Core Chat Messages

#### ChatRequest

Main request message for the bidirectional chat stream.

```protobuf
message ChatRequest {
  oneof request {
    ChatMessage message = 1;
    ChatControl control = 2;
    ToolApproval tool_approval = 3;
  }
}
```

#### ChatResponse

Response message containing various types of chat events.

```protobuf
message ChatResponse {
  oneof response {
    ChatMessage message = 1;
    AgentThinking thinking = 2;
    ToolExecution tool_execution = 3;
    ChatEvent event = 4;
    ChatError error = 5;
  }
}
```

#### ChatMessage

Core message structure for user and agent communication.

```protobuf
message ChatMessage {
  string session_id = 1;
  string sender_id = 2; // agent_id or "user"
  string sender_name = 3;
  string content = 4;
  MessageType type = 5;
  int64 timestamp = 6;
  map<string, string> metadata = 7;

  enum MessageType {
    USER_MESSAGE = 0;
    AGENT_RESPONSE = 1;
    SYSTEM_MESSAGE = 2;
    TOOL_REQUEST = 3;
    TOOL_RESULT = 4;
    AGENT_FRAGMENT = 5; // Streaming response fragment
  }
}
```

**Fields:**

- `session_id`: Unique identifier for the chat session
- `sender_id`: ID of the message sender (agent ID or "user")
- `sender_name`: Display name of the sender
- `content`: Message text content
- `type`: Message classification (see MessageType enum)
- `timestamp`: Unix timestamp in milliseconds
- `metadata`: Additional key-value pairs for message context

### Agent Communication

#### AgentThinking

Indicates agent processing state during response generation.

```protobuf
message AgentThinking {
  string agent_id = 1;
  string agent_name = 2;
  string session_id = 3;
  ThinkingState state = 4;
  string description = 5;
  int64 timestamp = 6;

  enum ThinkingState {
    ANALYZING = 0;
    PLANNING = 1;
    RESEARCHING = 2;
    EXECUTING = 3;
    REVIEWING = 4;
    FINALIZING = 5;
  }
}
```

**Usage:**
Sent during agent processing to provide real-time feedback on agent activity. Clients can display thinking indicators based on the state.

#### ToolExecution

Status and progress information for tool execution.

```protobuf
message ToolExecution {
  string tool_id = 1;
  string tool_name = 2;
  string agent_id = 3;
  string session_id = 4;
  map<string, string> parameters = 5;
  ExecutionStatus status = 6;
  double progress = 7; // 0.0 to 1.0
  string result = 8;
  string error = 9;
  int64 started_at = 10;
  int64 updated_at = 11;
  double estimated_cost = 12;
  repeated string required_permissions = 13;
  bool requires_approval = 14;

  enum ExecutionStatus {
    PENDING = 0;
    AWAITING_APPROVAL = 1;
    EXECUTING = 2;
    COMPLETED = 3;
    FAILED = 4;
    CANCELLED = 5;
  }
}
```

### Session Management

#### ChatSession

Complete session metadata and status.

```protobuf
message ChatSession {
  string id = 1;
  string name = 2;
  repeated string agent_ids = 3;
  string campaign_id = 4;
  SessionStatus status = 5;
  int64 created_at = 6;
  int64 last_activity = 7;
  map<string, string> metadata = 8;
  SessionContext context = 9;

  enum SessionStatus {
    ACTIVE = 0;
    PAUSED = 1;
    ENDED = 2;
    ERROR = 3;
  }
}
```

#### SessionContext

Contextual information about the session environment.

```protobuf
message SessionContext {
  string project_path = 1;
  string current_commission = 2;
  repeated string active_tasks = 3;
  map<string, string> environment = 4;
  repeated string available_tools = 5;
}
```

### Control Messages

#### ChatControl

Session lifecycle and flow control.

```protobuf
message ChatControl {
  ControlAction action = 1;
  string session_id = 2;
  map<string, string> parameters = 3;

  enum ControlAction {
    START_SESSION = 0;
    END_SESSION = 1;
    PAUSE_SESSION = 2;
    RESUME_SESSION = 3;
    INTERRUPT_AGENT = 4;
    REQUEST_STATUS = 5;
  }
}
```

#### ToolApproval

User approval/rejection for tool execution requests.

```protobuf
message ToolApproval {
  string tool_execution_id = 1;
  string session_id = 2;
  bool approved = 3;
  string reason = 4;
  map<string, string> modified_parameters = 5;
}
```

### Events and Errors

#### ChatEvent

System events during chat sessions.

```protobuf
message ChatEvent {
  string session_id = 1;
  EventType type = 2;
  string description = 3;
  map<string, string> data = 4;
  int64 timestamp = 5;

  enum EventType {
    SESSION_STARTED = 0;
    SESSION_ENDED = 1;
    AGENT_JOINED = 2;
    AGENT_LEFT = 3;
    TOOL_DISCOVERED = 4;
    MEMORY_ACCESSED = 5;
    CONTEXT_UPDATED = 6;
    ERROR_OCCURRED = 7;
    COMMISSION_CREATED = 8;
    TASK_ASSIGNED = 9;
  }
}
```

#### ChatError

Error information with context and recovery suggestions.

```protobuf
message ChatError {
  string session_id = 1;
  ErrorCode code = 2;
  string message = 3;
  map<string, string> details = 4;
  int64 timestamp = 5;

  enum ErrorCode {
    UNKNOWN = 0;
    INVALID_SESSION = 1;
    AGENT_UNAVAILABLE = 2;
    TOOL_EXECUTION_FAILED = 3;
    PERMISSION_DENIED = 4;
    RATE_LIMITED = 5;
    CONTEXT_TOO_LARGE = 6;
    INVALID_REQUEST = 7;
  }
}
```

## Request/Response Patterns

### Session Management RPCs

#### CreateChatSession

Creates a new chat session with specified agents and context.

```protobuf
message CreateChatSessionRequest {
  string name = 1;
  repeated string agent_ids = 2;
  string campaign_id = 3;
  map<string, string> metadata = 4;
  SessionContext context = 5;
}
```

**Response:** `ChatSession`

#### EndChatSession

Gracefully terminates a chat session.

```protobuf
message EndChatSessionRequest {
  string session_id = 1;
  string reason = 2;
}

message EndChatSessionResponse {
  bool success = 1;
  string message = 2;
  ChatSessionSummary summary = 3;
}
```

#### ListChatSessions

Retrieves active and optionally ended sessions.

```protobuf
message ListChatSessionsRequest {
  bool include_ended = 1;
  string campaign_id = 2;
  int32 limit = 3;
}

message ListChatSessionsResponse {
  repeated ChatSession sessions = 1;
  int32 total_count = 2;
}
```

#### GetChatHistory

Retrieves message history for a session with pagination.

```protobuf
message GetChatHistoryRequest {
  string session_id = 1;
  int64 since_timestamp = 2;
  int32 limit = 3;
  bool include_system_messages = 4;
}

message GetChatHistoryResponse {
  repeated ChatMessage messages = 1;
  repeated ChatEvent events = 2;
  int32 total_count = 3;
  bool has_more = 4;
}
```

### Summary Information

#### ChatSessionSummary

Aggregate statistics about a completed session.

```protobuf
message ChatSessionSummary {
  int32 total_messages = 1;
  int32 tools_executed = 2;
  double total_cost = 3;
  int32 tasks_created = 4;
  repeated string agents_involved = 5;
  string outcome = 6;
}
```

## Usage Examples

### Basic Chat Flow

```go
// 1. Create session
session, err := client.CreateChatSession(ctx, &pb.CreateChatSessionRequest{
    Name:       "development-session",
    CampaignId: "my-project",
})

// 2. Start chat stream
stream, err := client.Chat(ctx)

// 3. Send user message
stream.Send(&pb.ChatRequest{
    Request: &pb.ChatRequest_Message{
        Message: &pb.ChatMessage{
            SessionId:  session.Id,
            SenderId:   "user",
            SenderName: "Developer",
            Content:    "Help me debug this function",
            Type:       pb.ChatMessage_USER_MESSAGE,
            Timestamp:  time.Now().UnixMilli(),
        },
    },
})

// 4. Receive responses
for {
    resp, err := stream.Recv()
    if err != nil {
        break
    }
    
    switch v := resp.Response.(type) {
    case *pb.ChatResponse_Message:
        fmt.Printf("Agent: %s\n", v.Message.Content)
    case *pb.ChatResponse_Thinking:
        fmt.Printf("Agent is %s...\n", v.Thinking.State)
    case *pb.ChatResponse_ToolExecution:
        fmt.Printf("Tool %s: %s\n", v.ToolExecution.ToolName, v.ToolExecution.Status)
    }
}
```

### Tool Approval Workflow

```go
// Monitor for tool execution requests
for {
    resp, err := stream.Recv()
    
    if toolExec := resp.GetToolExecution(); toolExec != nil {
        if toolExec.RequiresApproval && toolExec.Status == pb.ToolExecution_AWAITING_APPROVAL {
            // Prompt user for approval
            approved := promptUserApproval(toolExec)
            
            // Send approval response
            stream.Send(&pb.ChatRequest{
                Request: &pb.ChatRequest_ToolApproval{
                    ToolApproval: &pb.ToolApproval{
                        ToolExecutionId: toolExec.ToolId,
                        SessionId:       session.Id,
                        Approved:        approved,
                        Reason:          "User decision",
                    },
                },
            })
        }
    }
}
```

### Session Recovery

```go
// List existing sessions
sessions, err := client.ListChatSessions(ctx, &pb.ListChatSessionsRequest{
    CampaignId: "my-project",
})

// Find the session to resume
var targetSession *pb.ChatSession
for _, session := range sessions.Sessions {
    if session.Status == pb.ChatSession_ACTIVE {
        targetSession = session
        break
    }
}

// Get recent history
history, err := client.GetChatHistory(ctx, &pb.GetChatHistoryRequest{
    SessionId: targetSession.Id,
    Limit:     10,
})

// Display context to user
for _, msg := range history.Messages {
    fmt.Printf("[%s] %s: %s\n", 
        time.Unix(msg.Timestamp/1000, 0).Format("15:04:05"),
        msg.SenderName, 
        msg.Content)
}

// Resume chatting with existing session
stream, err := client.Chat(ctx)
// ... continue with session.Id
```

## Error Handling

### Common Error Scenarios

1. **INVALID_SESSION**: Session ID not found or expired
   - **Action**: Create new session or list available sessions

2. **AGENT_UNAVAILABLE**: Requested agent not online
   - **Action**: Check agent status or select different agent

3. **TOOL_EXECUTION_FAILED**: Tool encountered an error
   - **Action**: Check tool parameters and retry or use alternative

4. **RATE_LIMITED**: Too many requests in time window
   - **Action**: Implement exponential backoff and retry

5. **CONTEXT_TOO_LARGE**: Message or context exceeds size limits
   - **Action**: Truncate content or summarize previous context

### Client-Side Retry Logic

```go
func sendMessageWithRetry(stream pb.ChatService_ChatClient, req *pb.ChatRequest, maxRetries int) error {
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := stream.Send(req)
        if err == nil {
            return nil
        }
        
        // Check if error is retryable
        if isRetryableError(err) {
            backoff := time.Duration(attempt+1) * time.Second
            time.Sleep(backoff)
            continue
        }
        
        return err // Non-retryable error
    }
    
    return fmt.Errorf("failed after %d attempts", maxRetries)
}
```

## Performance Considerations

### Message Size Limits

- **Maximum message content**: 1MB
- **Maximum metadata size**: 64KB
- **Maximum concurrent streams per client**: 100

### Streaming Best Practices

- Use client-side buffering for high-frequency messages
- Implement proper backpressure handling
- Close streams gracefully when done
- Monitor connection health with periodic pings

### Connection Management

- Use connection pooling for multiple concurrent sessions
- Implement exponential backoff for reconnections
- Set appropriate timeouts (recommended: 30s for requests, 5m for streams)
- Handle context cancellation properly
