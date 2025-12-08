# Guild Daemon Integration Guide

## Quick Start

### Starting the Daemon

```bash
# Auto-detect campaign and start daemon
guild serve

# Specific campaign
guild serve --campaign my-project

# Foreground mode for development
guild serve --foreground

# Custom socket path
guild serve --socket /tmp/my-guild.sock
```

### Connecting to the Daemon

```bash
# Start interactive chat
guild chat

# Connect to specific campaign
guild chat --campaign my-project

# Connect programmatically (see API examples below)
```

## gRPC API Integration

### Go Client Example

```go
package main

import (
    "context"
    "log"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    
    pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
)

func main() {
    // Connect to daemon
    conn, err := grpc.NewClient(
        "unix:///tmp/guild.sock",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Create chat client
    client := pb.NewChatServiceClient(conn)
    
    // Create session
    session, err := client.CreateChatSession(context.Background(), &pb.CreateChatSessionRequest{
        Name:       "my-session",
        CampaignId: "my-campaign",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Created session: %s", session.Id)
    
    // Start chat stream
    stream, err := client.Chat(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    // Send message
    err = stream.Send(&pb.ChatRequest{
        Request: &pb.ChatRequest_Message{
            Message: &pb.ChatMessage{
                SessionId:  session.Id,
                SenderId:   "user",
                SenderName: "Developer",
                Content:    "Hello, Guild!",
                Type:       pb.ChatMessage_USER_MESSAGE,
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Receive response
    resp, err := stream.Recv()
    if err != nil {
        log.Fatal(err)
    }
    
    if msg := resp.GetMessage(); msg != nil {
        log.Printf("Agent response: %s", msg.Content)
    }
}
```

### Python Client Example

```python
import grpc
import guild_pb2
import guild_pb2_grpc

def main():
    # Connect to daemon via Unix socket
    channel = grpc.insecure_channel('unix:///tmp/guild.sock')
    client = guild_pb2_grpc.ChatServiceStub(channel)
    
    # Create session
    session = client.CreateChatSession(
        guild_pb2.CreateChatSessionRequest(
            name="python-session",
            campaign_id="my-campaign"
        )
    )
    
    print(f"Created session: {session.id}")
    
    # Start chat stream
    def message_generator():
        yield guild_pb2.ChatRequest(
            message=guild_pb2.ChatMessage(
                session_id=session.id,
                sender_id="user",
                sender_name="Python Client",
                content="Hello from Python!",
                type=guild_pb2.ChatMessage.USER_MESSAGE
            )
        )
    
    # Send message and receive responses
    responses = client.Chat(message_generator())
    
    for response in responses:
        if response.HasField('message'):
            print(f"Agent: {response.message.content}")
        elif response.HasField('thinking'):
            print(f"Agent thinking: {response.thinking.description}")

if __name__ == "__main__":
    main()
```

### JavaScript/Node.js Client Example

```javascript
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');

// Load proto definitions
const packageDefinition = protoLoader.loadSync('guild/v1/chat.proto');
const guild = grpc.loadPackageDefinition(packageDefinition).guild.v1;

async function main() {
    // Connect to daemon
    const client = new guild.ChatService(
        'unix:///tmp/guild.sock',
        grpc.credentials.createInsecure()
    );
    
    // Create session
    const session = await new Promise((resolve, reject) => {
        client.createChatSession({
            name: 'js-session',
            campaign_id: 'my-campaign'
        }, (error, response) => {
            if (error) reject(error);
            else resolve(response);
        });
    });
    
    console.log(`Created session: ${session.id}`);
    
    // Start chat stream
    const stream = client.chat();
    
    // Handle responses
    stream.on('data', (response) => {
        if (response.message) {
            console.log(`Agent: ${response.message.content}`);
        } else if (response.thinking) {
            console.log(`Agent thinking: ${response.thinking.description}`);
        }
    });
    
    // Send message
    stream.write({
        message: {
            session_id: session.id,
            sender_id: 'user',
            sender_name: 'JS Client',
            content: 'Hello from JavaScript!',
            type: 'USER_MESSAGE'
        }
    });
}

main().catch(console.error);
```

## Event Streaming Integration

### Subscribing to Events

```go
// Connect to event stream
eventClient := pb.NewEventServiceClient(conn)

stream, err := eventClient.StreamEvents(context.Background(), &pb.StreamEventsRequest{
    EventTypes: []string{"task.created", "task.updated"},
    CampaignId: "my-campaign",
})
if err != nil {
    log.Fatal(err)
}

// Process events
for {
    event, err := stream.Recv()
    if err != nil {
        log.Printf("Stream error: %v", err)
        break
    }
    
    switch event.Type {
    case "task.created":
        log.Printf("New task: %s", event.Data["name"])
    case "task.updated":
        log.Printf("Task updated: %s -> %s", 
            event.Data["name"], event.Data["status"])
    }
}
```

### Publishing Events

```go
// Publish custom event
_, err := eventClient.PublishEvent(context.Background(), &pb.PublishEventRequest{
    Type: "custom.event",
    Data: map[string]string{
        "source": "my-integration",
        "action": "data_processed",
    },
})
if err != nil {
    log.Printf("Failed to publish event: %v", err)
}
```

## Session Management

### Session Lifecycle

```go
// List active sessions
sessions, err := client.ListChatSessions(context.Background(), &pb.ListChatSessionsRequest{
    CampaignId: "my-campaign",
})
if err != nil {
    log.Fatal(err)
}

for _, session := range sessions.Sessions {
    log.Printf("Session: %s (%s)", session.Name, session.Status)
}

// Get chat history
history, err := client.GetChatHistory(context.Background(), &pb.GetChatHistoryRequest{
    SessionId: sessionId,
    Limit:     50,
})
if err != nil {
    log.Fatal(err)
}

for _, msg := range history.Messages {
    log.Printf("[%s] %s: %s", 
        time.Unix(msg.Timestamp/1000, 0).Format("15:04:05"),
        msg.SenderName, 
        msg.Content)
}

// End session
_, err = client.EndChatSession(context.Background(), &pb.EndChatSessionRequest{
    SessionId: sessionId,
    Reason:    "Integration complete",
})
```

### Session Persistence

Sessions automatically persist across daemon restarts. To restore a session:

```go
// Sessions are automatically available after daemon restart
// Just reconnect and use the same session ID

conn, _ := grpc.NewClient("unix:///tmp/guild.sock", ...)
client := pb.NewChatServiceClient(conn)

// Use existing session ID
stream, err := client.Chat(context.Background())
// Send message with existing session_id
```

## Error Handling

### Connection Management

```go
func connectWithRetry(socketPath string, maxRetries int) (*grpc.ClientConn, error) {
    var conn *grpc.ClientConn
    var err error
    
    for i := 0; i < maxRetries; i++ {
        conn, err = grpc.NewClient(
            fmt.Sprintf("unix://%s", socketPath),
            grpc.WithTransportCredentials(insecure.NewCredentials()),
            grpc.WithTimeout(5*time.Second),
        )
        
        if err == nil {
            // Test connection
            ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
            health := grpc_health_v1.NewHealthClient(conn)
            _, healthErr := health.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
            cancel()
            
            if healthErr == nil {
                return conn, nil
            }
            conn.Close()
        }
        
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    
    return nil, fmt.Errorf("failed to connect after %d retries: %w", maxRetries, err)
}
```

### Stream Recovery

```go
func maintainChatStream(client pb.ChatServiceClient, sessionId string) {
    for {
        stream, err := client.Chat(context.Background())
        if err != nil {
            log.Printf("Failed to create stream: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }
        
        // Process stream until error
        for {
            resp, err := stream.Recv()
            if err != nil {
                log.Printf("Stream error: %v", err)
                break // Reconnect
            }
            
            // Handle response
            handleResponse(resp)
        }
        
        time.Sleep(1 * time.Second) // Brief pause before reconnect
    }
}
```

## Testing Integration

### Unit Tests with Mock Daemon

```go
func TestChatIntegration(t *testing.T) {
    // Start test daemon
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Connect to daemon
    conn, err := grpc.NewClient("unix:///tmp/guild-test.sock", ...)
    require.NoError(t, err)
    defer conn.Close()
    
    client := pb.NewChatServiceClient(conn)
    
    // Test session creation
    session, err := client.CreateChatSession(ctx, &pb.CreateChatSessionRequest{
        Name: "test-session",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, session.Id)
    
    // Test message exchange
    stream, err := client.Chat(ctx)
    require.NoError(t, err)
    
    // Send test message
    err = stream.Send(&pb.ChatRequest{
        Request: &pb.ChatRequest_Message{
            Message: &pb.ChatMessage{
                SessionId: session.Id,
                Content:   "test message",
                Type:      pb.ChatMessage_USER_MESSAGE,
            },
        },
    })
    require.NoError(t, err)
    
    // Verify response
    resp, err := stream.Recv()
    require.NoError(t, err)
    assert.NotNil(t, resp.GetMessage())
}
```

## Configuration

### Campaign-Specific Settings

```yaml
# guild.yaml
daemon:
  socket_path: "/tmp/guild-myproject.sock"
  log_level: "info"
  max_concurrent_sessions: 50
  message_buffer_size: 1000
  
  # Event streaming configuration
  events:
    buffer_size: 10000
    batch_size: 100
    flush_interval: "1s"
    
  # Database settings
  database:
    path: ".guild/memory.db"
    wal_mode: true
    busy_timeout: "30s"
```

### Environment Overrides

```bash
# Override socket path
export GUILD_DAEMON_SOCKET="/tmp/custom-guild.sock"

# Set log level
export GUILD_DAEMON_LOG_LEVEL="debug"

# Database location
export GUILD_DATABASE_PATH="/custom/path/memory.db"
```

## Troubleshooting

### Common Issues

1. **Socket Permission Denied**

   ```bash
   # Check socket permissions
   ls -la /tmp/guild*.sock
   
   # Ensure daemon running as same user
   ps aux | grep guild
   ```

2. **Connection Refused**

   ```bash
   # Check daemon status
   guild status
   
   # Start daemon if not running
   guild serve --foreground
   ```

3. **Stream Disconnections**

   ```bash
   # Check daemon logs
   tail -f ~/.guild/daemon.log
   
   # Verify socket connectivity
   nc -U /tmp/guild.sock
   ```

### Debug Tools

```bash
# Enable gRPC logging
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

# Database inspection
sqlite3 .guild/memory.db ".schema"
sqlite3 .guild/memory.db "SELECT * FROM sessions;"

# Process monitoring
lsof | grep guild
netstat -an | grep guild
```

## Best Practices

### Performance

- Use connection pooling for multiple concurrent requests
- Implement client-side message batching for high throughput
- Monitor memory usage in long-running streams
- Use appropriate timeout values for your use case

### Reliability

- Implement exponential backoff for reconnections
- Handle partial message delivery gracefully
- Persist client state across connection failures
- Monitor daemon health and restart if necessary

### Security

- Validate all message content before sending
- Use appropriate Unix socket permissions
- Sanitize data before database storage
- Implement rate limiting in client code
