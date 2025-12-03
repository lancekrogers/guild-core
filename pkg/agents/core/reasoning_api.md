# Reasoning System API Reference

## Table of Contents

1. [Core Types](#core-types)
2. [Main APIs](#main-apis)
3. [Streaming APIs](#streaming-apis)
4. [Storage APIs](#storage-apis)
5. [Pattern Learning APIs](#pattern-learning-apis)
6. [Token Management APIs](#token-management-apis)
7. [UI Integration APIs](#ui-integration-apis)
8. [Error Handling](#error-handling)
9. [Examples](#examples)

## Core Types

### ThinkingType

```go
type ThinkingType string

const (
    ThinkingTypeAnalysis        ThinkingType = "analysis"
    ThinkingTypePlanning        ThinkingType = "planning"
    ThinkingTypeHypothesis      ThinkingType = "hypothesis"
    ThinkingTypeVerification    ThinkingType = "verification"
    ThinkingTypeDecisionMaking  ThinkingType = "decision_making"
    ThinkingTypeErrorRecovery   ThinkingType = "error_recovery"
    ThinkingTypeOptimization    ThinkingType = "optimization"
)
```

### ThinkingBlock

```go
type ThinkingBlock struct {
    ID              string                    // Unique identifier
    Type            ThinkingType              // Type of thinking
    Content         string                    // Raw content
    StructuredData  *StructuredThinking       // Parsed structure
    Confidence      float64                   // 0-1 confidence score
    DecisionPoints  []DecisionPoint           // Key decisions made
    ToolContext     *ToolContext              // Tool usage context
    ErrorContext    *ErrorAnalysis            // Error handling info
    Timestamp       time.Time                 // Creation time
    Duration        time.Duration             // Processing time
    TokenCount      int                       // Token usage
    ParentID        *string                   // Parent block ID
    ChildIDs        []string                  // Child block IDs
    Tags            []string                  // Metadata tags
    Metadata        map[string]interface{}    // Additional data
}
```

### ReasoningChainEnhanced

```go
type ReasoningChainEnhanced struct {
    ID              string                    // Chain ID
    AgentID         string                    // Owning agent
    SessionID       string                    // Session context
    TaskID          string                    // Associated task
    Blocks          []*ThinkingBlock          // Thinking blocks
    Summary         string                    // Generated summary
    FinalConfidence float64                   // Overall confidence
    Strategy        ReasoningStrategy         // Applied strategy
    Quality         QualityMetrics            // Quality scores
    Performance     PerformanceMetrics        // Performance data
    Insights        []Insight                 // Extracted insights
    Patterns        []PatternMatch            // Matched patterns
    StartTime       time.Time                 // Start timestamp
    EndTime         time.Time                 // End timestamp
    TotalTokens     int                       // Total tokens used
    TotalCost       float64                   // Estimated cost
    Feedback        *ReasoningFeedback        // Human/system feedback
    Improvements    []Improvement             // Suggested improvements
    Context         map[string]interface{}    // Additional context
    Tags            []string                  // Metadata tags
}
```

## Main APIs

### ReasoningSystem

The main entry point for the reasoning system.

```go
// Create a new reasoning system
func NewReasoningSystem(ctx context.Context, config ReasoningSystemConfig) (*ReasoningSystem, error)

// Configuration
type ReasoningSystemConfig struct {
    ExtractorConfig ReasoningConfig
    StorageConfig   ReasoningStorageConfig
    EnableAnalytics bool
    DatabasePath    string // Empty for default .guild/memory.db
}

// Methods
func (rs *ReasoningSystem) EnhanceAgent(agent Agent) error
func (rs *ReasoningSystem) GetInsights(ctx context.Context, agentID string) ([]string, error)
func (rs *ReasoningSystem) StartMaintenance(ctx context.Context)
func (rs *ReasoningSystem) Close(ctx context.Context) error
```

### ReasoningExtractor

Extracts reasoning from LLM responses.

```go
// Create a new extractor
func NewReasoningExtractor(config ReasoningConfig) (*ReasoningExtractor, error)

// Configuration
type ReasoningConfig struct {
    EnableCaching      bool
    CacheMaxSize       int
    CacheTTL           time.Duration
    MaxReasoningLength int
    MinConfidence      float64
    MaxConfidence      float64
    StrictValidation   bool
}

// Methods
func (re *ReasoningExtractor) Extract(ctx context.Context, response string) (*ReasoningChain, error)
func (re *ReasoningExtractor) ExtractWithMetadata(ctx context.Context, response string, metadata map[string]interface{}) (*ReasoningChain, error)
func (re *ReasoningExtractor) ValidateChain(chain *ReasoningChain) error
func (re *ReasoningExtractor) GetStats() map[string]interface{}
```

### ThinkingBlockParser

Parses thinking blocks from text.

```go
// Create a new parser
func NewThinkingBlockParser() *ThinkingBlockParser

// Methods
func (p *ThinkingBlockParser) ParseThinkingBlocks(ctx context.Context, response string) ([]*ThinkingBlock, error)
func (p *ThinkingBlockParser) ValidateBlock(block *ThinkingBlock) error
```

### ReasoningChainBuilder

Builds reasoning chains with analysis.

```go
// Create a new builder
func NewReasoningChainBuilder(agentID, sessionID, taskID string) *ReasoningChainBuilder

// Methods
func (b *ReasoningChainBuilder) AddBlock(block *ThinkingBlock) error
func (b *ReasoningChainBuilder) SetStrategy(name, description string)
func (b *ReasoningChainBuilder) AdaptStrategy(reason, toStrategy string)
func (b *ReasoningChainBuilder) AddContext(key string, value interface{})
func (b *ReasoningChainBuilder) AddTag(tag string)
func (b *ReasoningChainBuilder) SetCost(cost float64)
func (b *ReasoningChainBuilder) Build(ctx context.Context) (*ReasoningChainEnhanced, error)
```

## Streaming APIs

### ReasoningStreamer

Real-time reasoning streaming.

```go
// Create a new streamer
func NewReasoningStreamer(parser *ThinkingBlockParser, chainBuilder *ReasoningChainBuilder) *ReasoningStreamer

// Configuration
type StreamConfig struct {
    BufferSize       int
    FlushInterval    time.Duration
    MaxBlockSize     int
    EnableMetrics    bool
    InterruptTimeout time.Duration
}

// Methods
func (rs *ReasoningStreamer) Stream(ctx context.Context, reader io.Reader) error
func (rs *ReasoningStreamer) Interrupt()
func (rs *ReasoningStreamer) EventChannel() <-chan StreamEvent
func (rs *ReasoningStreamer) ErrorChannel() <-chan error
func (rs *ReasoningStreamer) GetChain(ctx context.Context) (*ReasoningChainEnhanced, error)
```

### StreamEvent

Events emitted during streaming.

```go
type StreamEvent struct {
    Type      StreamEventType
    Timestamp time.Time
    Data      interface{}
    Metadata  map[string]interface{}
}

type StreamEventType string
const (
    StreamEventThinkingStart    StreamEventType = "thinking_start"
    StreamEventThinkingUpdate   StreamEventType = "thinking_update"
    StreamEventThinkingComplete StreamEventType = "thinking_complete"
    StreamEventContentChunk     StreamEventType = "content_chunk"
    StreamEventToolCall         StreamEventType = "tool_call"
    StreamEventDecisionPoint    StreamEventType = "decision_point"
    StreamEventConfidenceUpdate StreamEventType = "confidence_update"
    StreamEventError            StreamEventType = "error"
    StreamEventInterrupted      StreamEventType = "interrupted"
)
```

## Storage APIs

### ReasoningStorage

Interface for reasoning persistence.

```go
type ReasoningStorage interface {
    // Chain operations
    Store(ctx context.Context, chain *ReasoningChainEnhanced) error
    Get(ctx context.Context, chainID string) (*ReasoningChainEnhanced, error)
    Query(ctx context.Context, query *ReasoningQuery) ([]*ReasoningChainEnhanced, error)
    Delete(ctx context.Context, before time.Time) (int64, error)
    
    // Pattern operations
    StorePattern(ctx context.Context, pattern *LearnedPattern) error
    GetPattern(ctx context.Context, patternID string) (*LearnedPattern, error)
    QueryPatterns(ctx context.Context, query *PatternQuery) ([]*LearnedPattern, error)
    UpdatePattern(ctx context.Context, pattern *LearnedPattern) error
    
    // Analytics
    GetStats(ctx context.Context, agentID string, start, end time.Time) (*ReasoningStats, error)
    GetInsights(ctx context.Context, agentID string, limit int) ([]*StoredInsight, error)
    
    // Lifecycle
    Close(ctx context.Context) error
}
```

### ReasoningQuery

Query parameters for reasoning chains.

```go
type ReasoningQuery struct {
    AgentID       string
    SessionID     string
    TaskID        string
    MinConfidence float64
    MaxConfidence float64
    StartTime     time.Time
    EndTime       time.Time
    HasFeedback   *bool
    Tags          []string
    OrderBy       string // "timestamp", "confidence", "tokens"
    Ascending     bool
    Limit         int
    Offset        int
}
```

## Pattern Learning APIs

### PatternLearner

ML-based pattern learning system.

```go
// Create a new learner
func NewPatternLearner(repository PatternRepository, llmClient LLMClient) *PatternLearner

// Configuration
type PatternLearningConfig struct {
    MinOccurrences       int
    MinConfidence        float64
    MaxPatternsPerBatch  int
    LearningRate         float64
    DecayFactor          float64
    CrossDomainThreshold float64
}

// Methods
func (pl *PatternLearner) LearnPatterns(ctx context.Context, chains []*ReasoningChainEnhanced, config PatternLearningConfig) ([]*LearnedPattern, error)
func (pl *PatternLearner) ApplyPattern(ctx context.Context, pattern *LearnedPattern, context string) (*PatternApplication, error)
func (pl *PatternLearner) EvaluatePattern(ctx context.Context, pattern *LearnedPattern, outcomes []*PatternOutcome) (*PatternEvaluation, error)
func (pl *PatternLearner) RefinePattern(ctx context.Context, pattern *LearnedPattern, feedback *PatternFeedback) error
func (pl *PatternLearner) ApplySuggestions(ctx context.Context, currentContext string) ([]*PatternSuggestion, error)
```

### LearnedPattern

Discovered reasoning pattern.

```go
type LearnedPattern struct {
    ID                string
    Name              string
    Description       string
    Signature         []ThinkingType
    TriggerConditions []string
    ExpectedOutcomes  []string
    SuccessMetrics    map[string]float64
    Applications      []PatternApplication
    Performance       PatternPerformance
    LearningMetadata  LearningMetadata
    CreatedAt         time.Time
    LastUsed          time.Time
    Version           int
}
```

## Token Management APIs

### TokenManager

Manages token usage and safety.

```go
// Create a new manager
func NewTokenManager(config TokenConfig) *TokenManager

// Configuration
type TokenConfig struct {
    DefaultLimit     int
    SafetyMargin     float64   // Default 0.15 (85% soft limit)
    EmergencyMargin  float64   // Default 0.02 (98% emergency)
    CompactionConfig CompactionConfig
}

// Methods
func (tm *TokenManager) CreateWindow(ctx context.Context, windowID string, limit int) error
func (tm *TokenManager) AddMessage(ctx context.Context, windowID string, message ContextMessage) error
func (tm *TokenManager) GetWindow(windowID string) (*ContextWindow, error)
func (tm *TokenManager) CheckSafety(windowID string) (*TokenSafety, error)
func (tm *TokenManager) CompactWindow(ctx context.Context, windowID string) error
func (tm *TokenManager) PredictUsage(windowID string, plannedTokens int) (*TokenPrediction, error)
```

### TokenCounter

Provider-specific token counting.

```go
type TokenCounter interface {
    CountTokens(text string) (int, error)
    CountMessages(messages []ContextMessage) (int, error)
    EstimateTokens(text string) int
}

// Implementations
func NewOpenAITokenCounter(model string) *OpenAITokenCounter
func NewAnthropicTokenCounter(model string) *AnthropicTokenCounter
func NewOllamaTokenCounter(model string) *OllamaTokenCounter
```

## UI Integration APIs

### ReasoningDisplay

UI component for reasoning display.

```go
// Create a new display
func NewReasoningDisplay(theme Theme) *ReasoningDisplay

// Methods (tea.Model interface)
func (rd *ReasoningDisplay) Init() tea.Cmd
func (rd *ReasoningDisplay) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (rd *ReasoningDisplay) View() string

// Additional methods
func (rd *ReasoningDisplay) SetStreamer(streamer *core.ReasoningStreamer)
func (rd *ReasoningDisplay) HandleInterrupt()
func (rd *ReasoningDisplay) GetCurrentBlocks() []*core.ThinkingBlock
```

### ReasoningIntegration

Integrates reasoning with chat interface.

```go
// Create integration
func NewReasoningIntegration(display *ReasoningDisplay, chatModel tea.Model, config ReasoningIntegrationConfig) *ReasoningIntegration

// Configuration
type ReasoningIntegrationConfig struct {
    Layout          LayoutType
    UpdateInterval  time.Duration
    MaxBlocksShown  int
    ShowConfidence  bool
    ShowTimestamps  bool
    EnableShortcuts bool
}

// Layout options
type LayoutType string
const (
    LayoutBottom  LayoutType = "bottom"
    LayoutRight   LayoutType = "right"
    LayoutOverlay LayoutType = "overlay"
    LayoutSplit   LayoutType = "split"
)
```

## Error Handling

All APIs use the gerror package for consistent error handling:

```go
// Check for specific error codes
if err != nil {
    if gerror.Is(err, gerror.ErrCodeResourceLimit) {
        // Handle token limit exceeded
    }
}

// Extract error details
if err != nil {
    gerr := gerror.AsGError(err)
    component := gerr.Component()
    details := gerr.Details()
}

// Common error codes
gerror.ErrCodeValidation    // Invalid input
gerror.ErrCodeResourceLimit // Token/resource limits
gerror.ErrCodeNotFound      // Entity not found
gerror.ErrCodeInternal      // Internal error
gerror.ErrCodeCanceled      // Operation canceled
```

## Examples

### Basic Reasoning Extraction

```go
// Setup
extractor, err := core.NewReasoningExtractor(core.DefaultReasoningConfig())
if err != nil {
    return err
}

// Extract reasoning
response := "<thinking>I need to solve this step by step...</thinking>The answer is 42."
chain, err := extractor.Extract(ctx, response)
if err != nil {
    return err
}

fmt.Printf("Found %d thinking blocks\n", len(chain.Blocks))
fmt.Printf("Confidence: %.2f\n", chain.Confidence)
```

### Streaming with UI

```go
// Create components
parser := core.NewThinkingBlockParser()
builder := core.NewReasoningChainBuilder(agentID, sessionID, taskID)
streamer := core.NewReasoningStreamer(parser, builder)

// Create UI
display := ui.NewReasoningDisplay(theme)
display.SetStreamer(streamer)

// Stream response
go func() {
    if err := streamer.Stream(ctx, responseReader); err != nil {
        log.Printf("Stream error: %v", err)
    }
}()

// Handle events
for event := range streamer.EventChannel() {
    switch event.Type {
    case core.StreamEventThinkingStart:
        fmt.Println("Thinking started...")
    case core.StreamEventThinkingComplete:
        fmt.Println("Thinking complete!")
    }
}
```

### Pattern Learning

```go
// Setup pattern learning
learner := core.NewPatternLearner(repository, llmClient)

// Configure learning
config := core.PatternLearningConfig{
    MinOccurrences: 3,
    MinConfidence:  0.7,
    LearningRate:   0.1,
    DecayFactor:    0.95,
}

// Learn from recent chains
chains, _ := storage.Query(ctx, &core.ReasoningQuery{
    StartTime: time.Now().Add(-24 * time.Hour),
    Limit:     100,
})

patterns, err := learner.LearnPatterns(ctx, chains, config)
if err != nil {
    return err
}

fmt.Printf("Discovered %d patterns\n", len(patterns))

// Apply patterns
suggestions, _ := learner.ApplySuggestions(ctx, currentContext)
for _, suggestion := range suggestions {
    fmt.Printf("Pattern: %s (confidence: %.2f)\n", 
        suggestion.Pattern.Name, suggestion.Confidence)
}
```

### Token Management

```go
// Create token manager
manager := core.NewTokenManager(core.TokenConfig{
    DefaultLimit: 8000,
    SafetyMargin: 0.15,
})

// Create window
manager.CreateWindow(ctx, "main", 8000)

// Add messages
for _, msg := range conversation {
    if err := manager.AddMessage(ctx, "main", msg); err != nil {
        if gerror.Is(err, gerror.ErrCodeResourceLimit) {
            // Compact and retry
            manager.CompactWindow(ctx, "main")
            manager.AddMessage(ctx, "main", msg)
        }
    }
}

// Check safety
safety, _ := manager.CheckSafety("main")
if safety.PercentUsed > 0.85 {
    fmt.Println("Warning: Approaching token limit")
}
```

### Storage Queries

```go
// Query high-confidence chains
query := &core.ReasoningQuery{
    AgentID:       "agent-123",
    MinConfidence: 0.8,
    StartTime:     time.Now().Add(-7 * 24 * time.Hour),
    OrderBy:       "confidence",
    Ascending:     false,
    Limit:         50,
}

chains, err := storage.Query(ctx, query)
if err != nil {
    return err
}

// Get analytics
stats, _ := storage.GetStats(ctx, "agent-123", time.Time{}, time.Now())
fmt.Printf("Average confidence: %.2f\n", stats.AvgConfidence)
fmt.Printf("Success rate: %.2f%%\n", stats.SuccessRate * 100)
```

This comprehensive API documentation provides developers with everything they need to integrate and use the reasoning system effectively.
