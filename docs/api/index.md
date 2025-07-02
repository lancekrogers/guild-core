# Guild Framework API Reference

## Overview

Guild provides a comprehensive API for building multi-agent AI applications. This reference covers the core packages and interfaces for extending Guild with custom artisans, tools, and integrations.

## Core Packages

### `pkg/agent`

The agent package provides interfaces and implementations for AI artisans.

#### Agent Interface

```go
package agent

import (
    "context"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Agent interface {
    // GetID returns the unique identifier for this artisan
    GetID() string
    
    // GetCapabilities returns the tools this artisan can use
    GetCapabilities() []string
    
    // ProcessMessage handles incoming messages
    ProcessMessage(ctx context.Context, msg Message) (Response, gerror.Error)
    
    // GetState returns the current artisan state
    GetState() AgentState
}

type Message struct {
    ID       string                 `json:"id"`
    Content  string                 `json:"content"`
    Metadata map[string]interface{} `json:"metadata"`
    Timestamp time.Time             `json:"timestamp"`
}

type Response struct {
    Content   string                 `json:"content"`
    Actions   []Action               `json:"actions"`
    Metadata  map[string]interface{} `json:"metadata"`
    Cost      CostInfo               `json:"cost"`
}
```

#### Creating Custom Artisans

```go
package main

import (
    "context"
    "github.com/guild-framework/guild/pkg/agent"
    "github.com/guild-framework/guild/pkg/providers"
    "github.com/guild-framework/guild/pkg/gerror"
)

type CustomArtisan struct {
    agent.BaseAgent
    specialSkill string
}

func NewCustomArtisan(id string, provider providers.Provider) (*CustomArtisan, gerror.Error) {
    base, err := agent.NewBaseAgent(id, provider)
    if err != nil {
        return nil, gerror.Wrap(err, "failed to create base agent")
    }
    
    return &CustomArtisan{
        BaseAgent:    base,
        specialSkill: "domain-expertise",
    }, nil
}

func (ca *CustomArtisan) ProcessMessage(ctx context.Context, msg agent.Message) (agent.Response, gerror.Error) {
    // Add custom logic here
    if err := ctx.Err(); err != nil {
        return agent.Response{}, gerror.New(gerror.ErrCodeCancelled, "context cancelled")
    }
    
    // Use base implementation with context
    return ca.BaseAgent.ProcessMessage(ctx, msg)
}
```

### `pkg/orchestrator`

The orchestrator package manages multi-artisan coordination.

#### Orchestrator Interface

```go
package orchestrator

import (
    "context"
    "github.com/guild-framework/guild/pkg/agent"
    "github.com/guild-framework/guild/pkg/commission"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Orchestrator interface {
    // RegisterAgent adds an artisan to the orchestrator
    RegisterAgent(ctx context.Context, agent agent.Agent) gerror.Error
    
    // RouteMessage sends a message to the appropriate artisan
    RouteMessage(ctx context.Context, msg Message) gerror.Error
    
    // GetAgentStatus returns the status of all artisans
    GetAgentStatus(ctx context.Context) (map[string]AgentStatus, gerror.Error)
    
    // ExecuteCommission runs a commission with multiple artisans
    ExecuteCommission(ctx context.Context, commission commission.Commission) gerror.Error
}

type AgentStatus struct {
    ID         string    `json:"id"`
    State      string    `json:"state"`
    ActiveTask string    `json:"active_task,omitempty"`
    LastSeen   time.Time `json:"last_seen"`
    Cost       CostInfo  `json:"cost"`
}
```

#### Event System

```go
package main

import (
    "context"
    "log"
    "github.com/guild-framework/guild/pkg/orchestrator"
    "github.com/guild-framework/guild/pkg/gerror"
)

func subscribeToEvents(ctx context.Context, orch orchestrator.Orchestrator) gerror.Error {
    // Subscribe to orchestrator events
    events, err := orch.Subscribe(ctx, orchestrator.EventFilter{
        Types: []orchestrator.EventType{
            orchestrator.EventTaskStarted,
            orchestrator.EventTaskCompleted,
            orchestrator.EventAgentMessage,
        },
    })
    if err != nil {
        return gerror.Wrap(err, "failed to subscribe to events")
    }

    go func() {
        for {
            select {
            case event := <-events:
                switch e := event.(type) {
                case orchestrator.TaskStartedEvent:
                    log.Printf("Task %s started by %s", e.TaskID, e.AgentID)
                case orchestrator.TaskCompletedEvent:
                    log.Printf("Task %s completed", e.TaskID)
                }
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return nil
}
```

### `pkg/commission`

The commission package handles project planning and task breakdown.

#### Commission Structure

```go
package commission

import (
    "time"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Commission struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Tasks       []Task                 `json:"tasks"`
    Status      CommissionStatus       `json:"status"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

type Task struct {
    ID           string       `json:"id"`
    Title        string       `json:"title"`
    Description  string       `json:"description"`
    Agent        string       `json:"agent"`
    Dependencies []string     `json:"dependencies"`
    Status       TaskStatus   `json:"status"`
    Complexity   int          `json:"complexity"`
    EstimatedHours float64    `json:"estimated_hours"`
    ActualHours    float64    `json:"actual_hours"`
}

type CommissionStatus string

const (
    CommissionStatusPlanning   CommissionStatus = "planning"
    CommissionStatusInProgress CommissionStatus = "in_progress" 
    CommissionStatusCompleted  CommissionStatus = "completed"
    CommissionStatusBlocked    CommissionStatus = "blocked"
)
```

#### Commission Refinement

```go
package main

import (
    "context"
    "fmt"
    "github.com/guild-framework/guild/pkg/commission"
    "github.com/guild-framework/guild/pkg/gerror"
)

func refineCommission(ctx context.Context) gerror.Error {
    refiner := commission.NewRefiner()
    
    // Refine user requirements into tasks
    refined, err := refiner.Refine(ctx, commission.Commission{
        Name:        "REST API",
        Description: "Build a todo list API with auth",
    })
    if err != nil {
        return gerror.Wrap(err, "failed to refine commission")
    }

    // refined.Tasks contains the breakdown
    for _, task := range refined.Tasks {
        fmt.Printf("%s: %s (complexity: %d)\n", 
            task.ID, task.Title, task.Complexity)
    }
    
    return nil
}
```

### `pkg/corpus`

The corpus package provides knowledge management and RAG capabilities.

#### Corpus Interface

```go
package corpus

import (
    "context"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Corpus interface {
    // Index adds documents to the corpus
    Index(ctx context.Context, docs []Document) gerror.Error
    
    // Search queries the corpus
    Search(ctx context.Context, query string, opts SearchOptions) ([]Result, gerror.Error)
    
    // Extract learns from conversations
    Extract(ctx context.Context, messages []Message) ([]Knowledge, gerror.Error)
}

type Document struct {
    ID       string                 `json:"id"`
    Content  string                 `json:"content"`
    Metadata map[string]interface{} `json:"metadata"`
    Vector   []float64              `json:"vector,omitempty"`
}

type SearchOptions struct {
    MaxResults     int     `json:"max_results"`
    MinScore       float64 `json:"min_score"`
    IncludeContext bool    `json:"include_context"`
}
```

#### Vector Search

```go
package main

import (
    "context"
    "github.com/guild-framework/guild/pkg/corpus"
    "github.com/guild-framework/guild/pkg/gerror"
)

func searchCorpus(ctx context.Context) gerror.Error {
    corp, err := corpus.New(corpus.Config{
        VectorDB:  "chromadb",
        Dimension: 1536,
    })
    if err != nil {
        return gerror.Wrap(err, "failed to create corpus")
    }

    // Index documentation
    if err := corp.IndexDirectory(ctx, "./docs"); err != nil {
        return gerror.Wrap(err, "failed to index directory")
    }

    // Search with context
    results, err := corp.Search(ctx, "authentication patterns", 
        corpus.SearchOptions{
            MaxResults:     5,
            MinScore:       0.7,
            IncludeContext: true,
        })
    if err != nil {
        return gerror.Wrap(err, "search failed")
    }
    
    for _, result := range results {
        fmt.Printf("Score: %.2f - %s\n", result.Score, result.Content)
    }
    
    return nil
}
```

### `pkg/tools`

The tools package provides safe execution environments for artisan tools.

#### Tool Interface

```go
package tools

import (
    "context"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Tool interface {
    GetName() string
    GetDescription() string
    GetParameters() []Parameter
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, gerror.Error)
}

type Parameter struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Type        string      `json:"type"`
    Required    bool        `json:"required"`
    Default     interface{} `json:"default,omitempty"`
}
```

#### Creating Custom Tools

```go
package main

import (
    "context"
    "database/sql"
    "strings"
    "github.com/guild-framework/guild/pkg/tools"
    "github.com/guild-framework/guild/pkg/gerror"
)

type DatabaseTool struct {
    db *sql.DB
}

func NewDatabaseTool(db *sql.DB) *DatabaseTool {
    return &DatabaseTool{db: db}
}

func (dt *DatabaseTool) GetName() string {
    return "database_query"
}

func (dt *DatabaseTool) GetDescription() string {
    return "Execute read-only database queries"
}

func (dt *DatabaseTool) GetParameters() []tools.Parameter {
    return []tools.Parameter{
        {
            Name:        "query",
            Description: "SQL query to execute (SELECT only)",
            Type:        "string",
            Required:    true,
        },
    }
}

func (dt *DatabaseTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, gerror.Error) {
    if err := ctx.Err(); err != nil {
        return nil, gerror.New(gerror.ErrCodeCancelled, "context cancelled")
    }
    
    query, ok := params["query"].(string)
    if !ok {
        return nil, gerror.New(gerror.ErrCodeInvalidInput, "query parameter required")
    }
    
    // Validate query (read-only)
    if !dt.isReadOnly(query) {
        return nil, gerror.New(gerror.ErrCodeForbidden, "only SELECT queries allowed")
    }
    
    rows, err := dt.db.QueryContext(ctx, query)
    if err != nil {
        return nil, gerror.Wrap(gerror.New(gerror.ErrCodeDatabase, "query failed"), err.Error())
    }
    defer rows.Close()
    
    // Return results
    return dt.rowsToJSON(rows)
}

func (dt *DatabaseTool) isReadOnly(query string) bool {
    normalized := strings.ToUpper(strings.TrimSpace(query))
    return strings.HasPrefix(normalized, "SELECT")
}
```

### `pkg/session`

The session package handles conversation persistence.

#### Session Management

```go
package session

import (
    "context"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Manager interface {
    // Save persists a session
    Save(ctx context.Context, session *Session) gerror.Error
    
    // Load retrieves a session
    Load(ctx context.Context, sessionID string) (*Session, gerror.Error)
    
    // Export generates session exports
    Export(ctx context.Context, sessionID string, opts ExportOptions) ([]byte, gerror.Error)
}

type Session struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Messages  []Message `json:"messages"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type ExportOptions struct {
    Format          ExportFormat `json:"format"`
    IncludeMetadata bool         `json:"include_metadata"`
}

type ExportFormat string

const (
    FormatMarkdown ExportFormat = "markdown"
    FormatJSON     ExportFormat = "json"
    FormatHTML     ExportFormat = "html"
)
```

#### Example Usage

```go
package main

import (
    "context"
    "github.com/guild-framework/guild/pkg/session"
    "github.com/guild-framework/guild/pkg/gerror"
)

func manageSession(ctx context.Context) gerror.Error {
    manager, err := session.NewManager(session.Config{
        Storage:  "sqlite",
        AutoSave: true,
    })
    if err != nil {
        return gerror.Wrap(err, "failed to create session manager")
    }

    // Save session
    sess := &session.Session{
        ID:   "session-123",
        Name: "My Project",
    }
    
    if err := manager.Save(ctx, sess); err != nil {
        return gerror.Wrap(err, "failed to save session")
    }

    // Load session
    loaded, err := manager.Load(ctx, "session-123")
    if err != nil {
        return gerror.Wrap(err, "failed to load session")
    }

    // Export session
    data, err := manager.Export(ctx, loaded.ID, session.ExportOptions{
        Format:          session.FormatMarkdown,
        IncludeMetadata: true,
    })
    if err != nil {
        return gerror.Wrap(err, "failed to export session")
    }
    
    return nil
}
```

## Extension Points

### Provider Integration

Implement the Provider interface to add new LLM providers:

```go
package providers

import (
    "context"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Provider interface {
    GetName() string
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, gerror.Error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, gerror.Error)
}

type CompletionRequest struct {
    Messages    []Message `json:"messages"`
    Model       string    `json:"model"`
    Temperature float64   `json:"temperature"`
    MaxTokens   int       `json:"max_tokens"`
}

type CompletionResponse struct {
    Content string   `json:"content"`
    Cost    CostInfo `json:"cost"`
}
```

### Storage Backends

Implement storage interfaces for custom backends:

```go
package storage

import (
    "context"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Storage interface {
    Store(ctx context.Context, key string, value interface{}) gerror.Error
    Retrieve(ctx context.Context, key string, dest interface{}) gerror.Error
    Delete(ctx context.Context, key string) gerror.Error
    List(ctx context.Context, prefix string) ([]string, gerror.Error)
}
```

### Authentication

Add custom authentication methods:

```go
package auth

import (
    "context"
    "github.com/guild-framework/guild/pkg/gerror"
)

type Authenticator interface {
    Authenticate(ctx context.Context, credentials Credentials) (User, gerror.Error)
    Authorize(ctx context.Context, user User, resource string, action string) gerror.Error
}

type User struct {
    ID    string   `json:"id"`
    Name  string   `json:"name"`
    Roles []string `json:"roles"`
}
```

## gRPC API

Guild exposes a gRPC API for remote integration:

### Proto Definition

```protobuf
syntax = "proto3";

package guild.v1;

service GuildService {
    rpc CreateCommission(CreateCommissionRequest) returns (Commission);
    rpc GetCommissionStatus(GetStatusRequest) returns (CommissionStatus);
    rpc StreamEvents(StreamEventsRequest) returns (stream Event);
    rpc SendMessage(SendMessageRequest) returns (MessageResponse);
}

message Commission {
    string id = 1;
    string name = 2;
    string description = 3;
    repeated Task tasks = 4;
}

message Event {
    string type = 1;
    string commission_id = 2;
    string data = 3;
    int64 timestamp = 4;
}
```

### Client Example

```go
package main

import (
    "context"
    "io"
    "log"
    "google.golang.org/grpc"
    "github.com/guild-framework/guild/pkg/api"
    "github.com/guild-framework/guild/pkg/gerror"
)

func grpcClient(ctx context.Context) gerror.Error {
    conn, err := grpc.DialContext(ctx, "localhost:50051", grpc.WithInsecure())
    if err != nil {
        return gerror.Wrap(gerror.New(gerror.ErrCodeConnection, "connection failed"), err.Error())
    }
    defer conn.Close()
    
    client := api.NewGuildServiceClient(conn)

    // Create commission
    commission, err := client.CreateCommission(ctx, &api.CreateCommissionRequest{
        Name:        "My Project",
        Description: "Build something amazing",
    })
    if err != nil {
        return gerror.Wrap(gerror.New(gerror.ErrCodeAPI, "create commission failed"), err.Error())
    }

    // Stream events
    stream, err := client.StreamEvents(ctx, &api.StreamEventsRequest{
        CommissionId: commission.Id,
    })
    if err != nil {
        return gerror.Wrap(gerror.New(gerror.ErrCodeAPI, "stream events failed"), err.Error())
    }

    for {
        event, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return gerror.Wrap(gerror.New(gerror.ErrCodeAPI, "stream recv failed"), err.Error())
        }
        
        log.Printf("Event: %v", event)
    }
    
    return nil
}
```

## Error Handling

Guild uses structured errors with the `gerror` package:

```go
package main

import (
    "github.com/guild-framework/guild/pkg/gerror"
)

func exampleErrorHandling() {
    err := gerror.New(gerror.ErrCodeNotFound, "artisan not found").
        WithComponent("orchestrator").
        WithOperation("route_message").
        WithDetails("agent_id", agentID)

    // Error contains full context for debugging
    // Use with observability system for structured logging
}
```

## Performance Considerations

### Caching

Use the built-in cache for expensive operations:

```go
package main

import (
    "context"
    "time"
    "github.com/guild-framework/guild/pkg/cache"
    "github.com/guild-framework/guild/pkg/gerror"
)

func cachedOperation(ctx context.Context) gerror.Error {
    cache, err := cache.New(cache.Config{
        MaxSize: 100 * 1024 * 1024, // 100MB
        TTL:     5 * time.Minute,
    })
    if err != nil {
        return gerror.Wrap(err, "failed to create cache")
    }

    result, err := cache.GetOrCompute(ctx, "cache-key", func(ctx context.Context) (interface{}, gerror.Error) {
        // Expensive operation
        if err := ctx.Err(); err != nil {
            return nil, gerror.New(gerror.ErrCodeCancelled, "context cancelled")
        }
        return computeResult(), nil
    })
    if err != nil {
        return gerror.Wrap(err, "cache operation failed")
    }
    
    return nil
}
```

### Connection Pooling

Reuse connections for external services:

```go
package main

import (
    "context"
    "time"
    "github.com/guild-framework/guild/pkg/pool"
    "github.com/guild-framework/guild/pkg/gerror"
)

func pooledConnections(ctx context.Context) gerror.Error {
    pool, err := pool.New(pool.Config{
        MaxConnections: 10,
        IdleTimeout:    30 * time.Second,
    })
    if err != nil {
        return gerror.Wrap(err, "failed to create pool")
    }

    conn, err := pool.Get(ctx)
    if err != nil {
        return gerror.Wrap(err, "failed to get connection")
    }
    defer pool.Put(conn)
    
    // Use connection
    return nil
}
```

## Testing

### Unit Testing

```go
package agent_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/guild-framework/guild/pkg/agent"
    "github.com/guild-framework/guild/pkg/mocks"
)

func TestAgent(t *testing.T) {
    ctx := context.Background()
    
    // Use mock provider
    provider := mocks.NewMockProvider()
    agent, err := NewCustomAgent("test", provider)
    assert.NoError(t, err)
    
    // Test message processing
    response, err := agent.ProcessMessage(ctx, agent.Message{
        Content: "test message",
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, response.Content)
}

func TestContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    provider := mocks.NewMockProvider()
    agent, err := NewCustomAgent("test", provider)
    assert.NoError(t, err)
    
    // Should handle context cancellation gracefully
    _, err = agent.ProcessMessage(ctx, agent.Message{
        Content: "test message",
    })
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cancelled")
}
```

### Integration Testing

```go
package integration_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/guild-framework/guild/pkg/orchestrator"
    "github.com/guild-framework/guild/pkg/commission"
)

func TestCommissionExecution(t *testing.T) {
    ctx := context.Background()
    
    // Use test orchestrator
    orch, err := orchestrator.NewTestOrchestrator()
    assert.NoError(t, err)
    
    // Execute commission
    err = orch.ExecuteCommission(ctx, commission.Commission{
        Name:  "Test Project",
        Tasks: []commission.Task{{Title: "Test Task"}},
    })
    
    assert.NoError(t, err)
}
```

## Observability Integration

Guild integrates with standard observability tools:

```go
package main

import (
    "context"
    "github.com/guild-framework/guild/pkg/observability"
    "go.opentelemetry.io/otel/trace"
)

func instrumentedFunction(ctx context.Context) error {
    // Add tracing
    ctx, span := observability.StartSpan(ctx, "custom-operation")
    defer span.End()
    
    // Add context to all operations
    if err := ctx.Err(); err != nil {
        span.RecordError(err)
        return err
    }
    
    return nil
}
```

This API reference provides comprehensive coverage of Guild's extensibility points while maintaining staff-level engineering standards with proper context propagation, error handling using gerror, and observability integration.
