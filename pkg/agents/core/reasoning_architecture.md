# Reasoning System Architecture

## Overview

The Guild Framework Reasoning System transforms raw LLM thinking into structured, analyzable, and learnable data. This document describes the architecture, design decisions, and implementation details of the production-grade reasoning system.

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Core Components](#core-components)
3. [Architecture Decision Records](#architecture-decision-records)
4. [Data Flow](#data-flow)
5. [Integration Guide](#integration-guide)
6. [Performance Considerations](#performance-considerations)
7. [Future Enhancements](#future-enhancements)

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Chat Interface                              │
│  ┌─────────────┐  ┌──────────────────┐  ┌─────────────────────┐   │
│  │   Chat UI   │  │ Reasoning Display │  │ Integration Layer  │   │
│  └──────┬──────┘  └────────┬─────────┘  └──────────┬──────────┘   │
└─────────┼──────────────────┼────────────────────────┼──────────────┘
          │                  │                        │
┌─────────▼──────────────────▼────────────────────────▼──────────────┐
│                      Reasoning System Core                          │
│  ┌─────────────┐  ┌──────────────────┐  ┌─────────────────────┐   │
│  │  Extractor  │  │ Thinking Parser  │  │  Chain Builder     │   │
│  └──────┬──────┘  └────────┬─────────┘  └──────────┬──────────┘   │
│         │                  │                        │              │
│  ┌──────▼──────┐  ┌────────▼─────────┐  ┌──────────▼──────────┐   │
│  │   Storage   │  │ Token Management │  │ Pattern Learning   │   │
│  └──────┬──────┘  └────────┬─────────┘  └──────────┬──────────┘   │
└─────────┼──────────────────┼────────────────────────┼──────────────┘
          │                  │                        │
┌─────────▼──────────────────▼────────────────────────▼──────────────┐
│                        Infrastructure                               │
│  ┌─────────────┐  ┌──────────────────┐  ┌─────────────────────┐   │
│  │   SQLite    │  │  Observability   │  │    Metrics         │   │
│  └─────────────┘  └──────────────────┘  └─────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. ThinkingBlock Structure (`thinking_block.go`)

The fundamental unit of reasoning, capturing not just what was thought, but how and why:

```go
type ThinkingBlock struct {
    ID              string
    Type            ThinkingType
    Content         string
    StructuredData  *StructuredThinking
    Confidence      float64
    DecisionPoints  []DecisionPoint
    ToolContext     *ToolContext
    ErrorContext    *ErrorAnalysis
    Timestamp       time.Time
    Duration        time.Duration
    TokenCount      int
    ParentID        *string
    ChildIDs        []string
    Tags            []string
    Metadata        map[string]interface{}
}
```

**Key Features:**
- 7 distinct thinking types (Analysis, Planning, Decision Making, etc.)
- Rich metadata for learning and analysis
- Hierarchical relationships between thoughts
- Tool integration context

### 2. Reasoning Chain Enhanced (`reasoning_chain_enhanced.go`)

Aggregates thinking blocks into complete reasoning flows:

```go
type ReasoningChainEnhanced struct {
    Blocks          []*ThinkingBlock
    Strategy        ReasoningStrategy
    Quality         QualityMetrics
    Performance     PerformanceMetrics
    Insights        []Insight
    Patterns        []PatternMatch
    // ... additional fields
}
```

**Key Features:**
- Strategy tracking with adaptations
- Multi-dimensional quality scoring
- Performance metrics for optimization
- Automatic insight extraction

### 3. Real-time Streaming (`reasoning_stream.go`)

Enables live reasoning display with interruption support:

```go
type ReasoningStreamer struct {
    parser        *ThinkingBlockParser
    chainBuilder  *ReasoningChainBuilder
    eventChan     chan StreamEvent
    errorChan     chan error
    interruptChan chan struct{}
    // ... state management
}
```

**Key Features:**
- Event-based streaming architecture
- Backpressure handling
- Graceful interruption (ESC key)
- Custom chunk splitting for optimal display

### 4. Token Management (`token_management.go`)

Prevents context overflow with intelligent management:

```go
type TokenManager struct {
    windows   sync.Map // map[string]*ContextWindow
    counters  map[string]TokenCounter
    compactor *ContextCompactor
    analyzer  *TokenAnalyzer
    predictor *TokenPredictor
}
```

**Key Features:**
- Provider-specific token counting
- Safety margins (85% soft limit, 95% hard limit)
- Automatic compaction triggers
- Token usage prediction

### 5. Pattern Learning (`pattern_learning.go`)

ML-based pattern discovery and application:

```go
type PatternLearner struct {
    repository       PatternRepository
    analyzer         *PatternAnalyzer
    applicator       *PatternApplicator
    evaluator        *PatternEvaluator
    featureExtractor *FeatureExtractor
}
```

**Key Features:**
- Automatic pattern discovery
- Reinforcement learning with decay
- Cross-domain pattern transfer
- Feature extraction for similarity

### 6. Compaction Strategies (`compaction_strategies.go`)

Multiple strategies for context compression:

- **Summarization**: LLM-based intelligent summarization
- **Priority-based**: Preserves high-value messages
- **Redundancy**: Removes duplicate information
- **Temporal**: Decays old messages
- **Hybrid**: Combines multiple strategies

### 7. UI Components (`reasoning_display.go`, `reasoning_integration.go`)

Production-grade UI integration:

```go
type ReasoningDisplay struct {
    viewport    viewport.Model
    spinner     spinner.Model
    streamer    *core.ReasoningStreamer
    eventChan   <-chan core.StreamEvent
    blocks      []*core.ThinkingBlock
    // ... display state
}
```

**Key Features:**
- Multiple layout options (bottom, right, overlay, split)
- Rich styling with lipgloss
- Real-time updates with spinner
- Interruption handling

## Architecture Decision Records

### ADR-001: Structured Thinking Blocks

**Status**: Accepted

**Context**: Raw text reasoning is difficult to analyze and learn from.

**Decision**: Implement structured ThinkingBlock with typed content and metadata.

**Consequences**:
- ✅ Enables sophisticated analysis and pattern learning
- ✅ Provides rich context for debugging and improvement
- ❌ Increases complexity and storage requirements

### ADR-002: Event-Based Streaming

**Status**: Accepted

**Context**: Need real-time reasoning display without blocking operations.

**Decision**: Use channel-based event streaming with dedicated goroutines.

**Consequences**:
- ✅ Non-blocking UI updates
- ✅ Clean separation of concerns
- ✅ Supports interruption
- ❌ Requires careful channel management

### ADR-003: Provider-Agnostic Token Counting

**Status**: Accepted

**Context**: Different LLM providers count tokens differently.

**Decision**: Implement provider-specific token counters with common interface.

**Consequences**:
- ✅ Accurate token counts per provider
- ✅ Prevents unexpected truncation
- ❌ Requires maintaining multiple implementations

### ADR-004: SQLite for Persistence

**Status**: Accepted

**Context**: Need durable storage for reasoning chains and patterns.

**Decision**: Use SQLite with structured schema and migrations.

**Consequences**:
- ✅ Zero-dependency persistence
- ✅ ACID compliance
- ✅ Easy backup and portability
- ❌ Limited concurrent write performance

### ADR-005: Pattern Learning with Decay

**Status**: Accepted

**Context**: Patterns should evolve over time based on effectiveness.

**Decision**: Implement reinforcement learning with temporal decay.

**Consequences**:
- ✅ Adapts to changing requirements
- ✅ Prevents stale patterns
- ❌ Requires tuning decay parameters

## Data Flow

### 1. Reasoning Extraction Flow

```
LLM Response → ReasoningExtractor → ThinkingBlockParser
                                          ↓
                                    ThinkingBlocks
                                          ↓
                                  ReasoningChainBuilder
                                          ↓
                                 ReasoningChainEnhanced
                                          ↓
                                    Storage & Analysis
```

### 2. Real-time Streaming Flow

```
LLM Stream → ReasoningStreamer → StreamEvents
                                      ↓
                              StreamProcessor
                                      ↓
                              ReasoningDisplay
                                      ↓
                                 UI Updates
```

### 3. Pattern Learning Flow

```
Stored Chains → PatternAnalyzer → Pattern Candidates
                                        ↓
                                PatternEvaluator
                                        ↓
                                Validated Patterns
                                        ↓
                                PatternRepository
```

## Integration Guide

### Basic Integration

```go
// 1. Create reasoning system
config := core.DefaultReasoningSystemConfig()
system, err := core.NewReasoningSystem(ctx, config)
if err != nil {
    return err
}
defer system.Close(ctx)

// 2. Enhance agent
agent := createAgent() // Your agent creation
if err := system.EnhanceAgent(agent); err != nil {
    return err
}

// 3. Use enhanced agent
response, err := agent.ExecuteWithReasoning(ctx, "Your task here")
```

### UI Integration

```go
// 1. Create reasoning display
display := ui.NewReasoningDisplay(theme)

// 2. Create integration
config := ui.ReasoningIntegrationConfig{
    Layout:          ui.LayoutSplit,
    UpdateInterval:  100 * time.Millisecond,
    MaxBlocksShown:  10,
}
integration := ui.NewReasoningIntegration(display, chatModel, config)

// 3. Use in tea.Program
p := tea.NewProgram(integration)
```

### Custom Pattern Learning

```go
// 1. Create pattern learner
learner := core.NewPatternLearner(repository, nil)

// 2. Learn from chains
patterns, err := learner.LearnPatterns(ctx, recentChains, config)

// 3. Apply patterns
suggestions, err := learner.ApplySuggestions(ctx, currentContext)
```

## Performance Considerations

### Token Management
- **Soft limit** at 85% triggers compaction
- **Hard limit** at 95% forces immediate action
- **Emergency limit** at 98% prevents overflow
- Token prediction helps avoid limits

### Memory Management
- Concurrent-safe operations with sync.Map
- Lazy loading of pattern data
- Configurable retention policies
- Background cleanup tasks

### Streaming Performance
- Buffered channels (default 1024)
- Custom splitting for optimal chunks
- Flush intervals (100ms default)
- Backpressure handling

### Storage Optimization
- JSON compression for large blocks
- Indexed queries on common fields
- Batch operations for efficiency
- Configurable retention (90 days default)

## Future Enhancements

### Short-term (Next Sprint)
1. **Distributed Reasoning**: Multi-agent reasoning coordination
2. **Advanced Visualizations**: Reasoning graph visualization
3. **Export Formats**: Export chains as markdown, JSON, or PDF
4. **Pattern Marketplace**: Share and import reasoning patterns

### Medium-term (Next Quarter)
1. **Reasoning Debugger**: Step-through reasoning with breakpoints
2. **A/B Testing**: Compare reasoning strategies
3. **Custom Thinking Types**: Domain-specific reasoning types
4. **Reasoning Metrics Dashboard**: Real-time analytics

### Long-term (Next Year)
1. **Reasoning Compiler**: Compile patterns into optimized prompts
2. **Federated Learning**: Learn from distributed deployments
3. **Reasoning Certification**: Validate reasoning quality
4. **AI-Human Collaboration**: Mixed reasoning chains

## Conclusion

The Guild Framework Reasoning System represents a significant advancement in making AI reasoning transparent, analyzable, and improvable. By treating reasoning as structured data rather than opaque text, we enable a new class of AI applications that can learn, adapt, and explain their thinking process.

The system is designed for production use with careful attention to performance, reliability, and extensibility. Whether you're building a simple chatbot or a complex multi-agent system, the reasoning framework provides the tools you need to understand and improve your AI's thinking process.