# Reasoning System Implementation Guide

## Overview

This guide provides step-by-step instructions for implementing and extending the Guild Framework Reasoning System. Whether you're adding new features, customizing behavior, or troubleshooting issues, this guide will help you understand the implementation details.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Implementation Patterns](#implementation-patterns)
3. [Adding New Thinking Types](#adding-new-thinking-types)
4. [Creating Custom Compaction Strategies](#creating-custom-compaction-strategies)
5. [Implementing New Token Counters](#implementing-new-token-counters)
6. [Building Custom UI Layouts](#building-custom-ui-layouts)
7. [Performance Optimization](#performance-optimization)
8. [Testing Strategies](#testing-strategies)
9. [Debugging Guide](#debugging-guide)
10. [Best Practices](#best-practices)

## Getting Started

### Prerequisites

- Go 1.21+
- SQLite3
- Understanding of concurrent Go programming
- Familiarity with the Guild Framework

### Basic Setup

```go
// 1. Import required packages
import (
    "context"
    "github.com/lancekrogers/guild/pkg/agents/core"
    "github.com/lancekrogers/guild/pkg/config"
    "github.com/lancekrogers/guild/pkg/observability"
)

// 2. Initialize observability
ctx := context.Background()
ctx = observability.WithComponent(ctx, "reasoning_system")
logger := observability.GetLogger(ctx)

// 3. Create reasoning system
systemConfig := core.DefaultReasoningSystemConfig()
reasoningSystem, err := core.NewReasoningSystem(ctx, systemConfig)
if err != nil {
    logger.WithError(err).ErrorContext(ctx, "Failed to create reasoning system")
    return err
}
defer reasoningSystem.Close(ctx)

// 4. Start maintenance tasks
reasoningSystem.StartMaintenance(ctx)
```

## Implementation Patterns

### 1. Context-First Pattern

Always pass context as the first parameter:

```go
func ProcessReasoning(ctx context.Context, input string) (*core.ReasoningChain, error) {
    // Check context early
    if err := ctx.Err(); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeCanceled, "context cancelled")
    }
    
    // Use context for timeouts
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Pass context through
    return extractor.Extract(ctx, input)
}
```

### 2. Error Wrapping Pattern

Use gerror for consistent error handling:

```go
func ValidateThinkingBlock(block *core.ThinkingBlock) error {
    if block == nil {
        return gerror.New(gerror.ErrCodeValidation, "thinking block is nil", nil).
            WithComponent("validator").
            WithOperation("ValidateThinkingBlock")
    }
    
    if block.Confidence < 0 || block.Confidence > 1 {
        return gerror.Newf(gerror.ErrCodeValidation, 
            "invalid confidence: %f", block.Confidence).
            WithComponent("validator").
            WithDetails("block_id", block.ID)
    }
    
    return nil
}
```

### 3. Concurrent-Safe Pattern

Use sync primitives for thread safety:

```go
type SafePatternRepository struct {
    mu       sync.RWMutex
    patterns map[string]*core.LearnedPattern
    index    sync.Map // For fast lookups
}

func (r *SafePatternRepository) Get(patternID string) (*core.LearnedPattern, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    pattern, exists := r.patterns[patternID]
    if !exists {
        return nil, gerror.Newf(gerror.ErrCodeNotFound, 
            "pattern not found: %s", patternID)
    }
    
    return pattern, nil
}

func (r *SafePatternRepository) Store(pattern *core.LearnedPattern) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    r.patterns[pattern.ID] = pattern
    r.index.Store(pattern.Name, pattern.ID)
    
    return nil
}
```

## Adding New Thinking Types

### Step 1: Define the Type

```go
// In thinking_block.go
const (
    // ... existing types
    ThinkingTypeCreative ThinkingType = "creative"
)
```

### Step 2: Add Structure Extractor

```go
// In structure_extractor.go
func (se *StructureExtractor) extractCreative(content string) (*StructuredThinking, error) {
    structured := &StructuredThinking{
        Type: ThinkingTypeCreative,
    }
    
    // Extract creative elements
    ideas := se.extractIdeas(content)
    metaphors := se.extractMetaphors(content)
    connections := se.extractConnections(content)
    
    structured.Metadata = map[string]interface{}{
        "ideas":       ideas,
        "metaphors":   metaphors,
        "connections": connections,
        "originality": se.assessOriginality(content),
    }
    
    return structured, nil
}
```

### Step 3: Update Parser

```go
// In thinking_block.go
func (p *ThinkingBlockParser) detectType(content string) ThinkingType {
    // ... existing detection logic
    
    // Creative thinking indicators
    creativeKeywords := []string{
        "imagine", "what if", "creative", "novel",
        "innovative", "brainstorm", "idea",
    }
    
    if containsKeywords(content, creativeKeywords) {
        return ThinkingTypeCreative
    }
    
    return ThinkingTypeAnalysis // default
}
```

### Step 4: Add Quality Metrics

```go
// In quality_scorer.go
func (qs *QualityScorer) scoreCreativeThinking(block *core.ThinkingBlock) float64 {
    score := 0.0
    weights := map[string]float64{
        "originality":  0.3,
        "feasibility":  0.2,
        "connections":  0.2,
        "elaboration":  0.2,
        "flexibility":  0.1,
    }
    
    // Score each dimension
    if metadata, ok := block.Metadata["originality"].(float64); ok {
        score += metadata * weights["originality"]
    }
    
    // ... score other dimensions
    
    return score
}
```

## Creating Custom Compaction Strategies

### Step 1: Implement the Interface

```go
type CompactionStrategy interface {
    Compact(ctx context.Context, window *ContextWindow) (*ContextWindow, error)
    Priority() int
    Name() string
}
```

### Step 2: Create Your Strategy

```go
type SemanticCompactionStrategy struct {
    embedder     Embedder
    similarity   float64
    minMessages  int
}

func (s *SemanticCompactionStrategy) Compact(
    ctx context.Context, 
    window *core.ContextWindow,
) (*core.ContextWindow, error) {
    if len(window.Messages) < s.minMessages {
        return window, nil
    }
    
    // Generate embeddings
    embeddings, err := s.generateEmbeddings(ctx, window.Messages)
    if err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeInternal, 
            "failed to generate embeddings")
    }
    
    // Cluster similar messages
    clusters := s.clusterMessages(embeddings, s.similarity)
    
    // Keep representative from each cluster
    compacted := &core.ContextWindow{
        ID:           window.ID,
        MaxTokens:    window.MaxTokens,
        Messages:     make([]core.ContextMessage, 0),
        CurrentUsage: 0,
    }
    
    for _, cluster := range clusters {
        representative := s.selectRepresentative(cluster)
        compacted.Messages = append(compacted.Messages, representative)
    }
    
    return compacted, nil
}

func (s *SemanticCompactionStrategy) Priority() int {
    return 3 // Higher than redundancy, lower than priority
}

func (s *SemanticCompactionStrategy) Name() string {
    return "semantic"
}
```

### Step 3: Register the Strategy

```go
// In context_compactor.go
func NewContextCompactor(summarizer MessageSummarizer) *ContextCompactor {
    compactor := &ContextCompactor{
        strategies: make(map[string]CompactionStrategy),
    }
    
    // Register default strategies
    compactor.RegisterStrategy(NewSummarizationStrategy(summarizer))
    compactor.RegisterStrategy(NewPriorityStrategy())
    compactor.RegisterStrategy(NewRedundancyStrategy())
    compactor.RegisterStrategy(NewTemporalStrategy())
    
    // Register custom strategy
    compactor.RegisterStrategy(NewSemanticCompactionStrategy(embedder))
    
    return compactor
}
```

## Implementing New Token Counters

### Step 1: Implement TokenCounter Interface

```go
type GeminiTokenCounter struct {
    model      string
    tokenizer  Tokenizer
    cache      sync.Map
    cacheTTL   time.Duration
}

func NewGeminiTokenCounter(model string) *GeminiTokenCounter {
    return &GeminiTokenCounter{
        model:     model,
        tokenizer: newGeminiTokenizer(model),
        cacheTTL:  5 * time.Minute,
    }
}

func (tc *GeminiTokenCounter) CountTokens(text string) (int, error) {
    // Check cache
    if cached, ok := tc.cache.Load(text); ok {
        if entry, ok := cached.(*tokenCacheEntry); ok {
            if time.Since(entry.timestamp) < tc.cacheTTL {
                return entry.count, nil
            }
        }
    }
    
    // Count tokens
    tokens, err := tc.tokenizer.Tokenize(text)
    if err != nil {
        return 0, gerror.Wrap(err, gerror.ErrCodeInternal, 
            "tokenization failed")
    }
    
    count := len(tokens)
    
    // Cache result
    tc.cache.Store(text, &tokenCacheEntry{
        count:     count,
        timestamp: time.Now(),
    })
    
    return count, nil
}

func (tc *GeminiTokenCounter) CountMessages(
    messages []core.ContextMessage,
) (int, error) {
    total := 0
    
    for _, msg := range messages {
        // Count content
        contentTokens, err := tc.CountTokens(msg.Content)
        if err != nil {
            return 0, err
        }
        
        // Add role tokens
        roleTokens := tc.getRoleTokens(msg.Role)
        
        // Add formatting overhead
        overhead := 4 // Gemini-specific
        
        total += contentTokens + roleTokens + overhead
    }
    
    return total, nil
}

func (tc *GeminiTokenCounter) EstimateTokens(text string) int {
    // Quick estimation without full tokenization
    words := len(strings.Fields(text))
    chars := len(text)
    
    // Gemini-specific heuristic
    return int(float64(words)*1.2 + float64(chars)*0.05)
}
```

### Step 2: Register in Token Manager

```go
// In token_management.go
func NewTokenManager(config TokenConfig) *TokenManager {
    tm := &TokenManager{
        counters:  make(map[string]TokenCounter),
        compactor: NewContextCompactor(config.Summarizer),
        analyzer:  NewTokenAnalyzer(),
        predictor: NewTokenPredictor(),
    }
    
    // Register counters
    tm.counters["openai"] = NewOpenAITokenCounter("gpt-4")
    tm.counters["anthropic"] = NewAnthropicTokenCounter("claude-3")
    tm.counters["ollama"] = NewOllamaTokenCounter("llama2")
    tm.counters["gemini"] = NewGeminiTokenCounter("gemini-pro")
    
    return tm
}
```

## Building Custom UI Layouts

### Step 1: Define Layout Type

```go
// In reasoning_integration.go
const (
    // ... existing layouts
    LayoutFloating LayoutType = "floating"
)
```

### Step 2: Implement Layout Logic

```go
func (ri *ReasoningIntegration) renderFloating() string {
    var b strings.Builder
    
    // Render main chat in background
    chatView := ri.chatModel.View()
    b.WriteString(chatView)
    
    // Calculate floating window position
    floatWidth := ri.width / 3
    floatHeight := ri.height / 3
    floatX := ri.width - floatWidth - 2
    floatY := 2
    
    // Create floating window
    floatingStyle := lipgloss.NewStyle().
        Width(floatWidth).
        Height(floatHeight).
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("99")).
        Padding(1).
        Background(lipgloss.Color("235"))
    
    // Render reasoning in floating window
    reasoningView := ri.display.View()
    floatingWindow := floatingStyle.Render(reasoningView)
    
    // Overlay floating window
    lines := strings.Split(b.String(), "\n")
    floatingLines := strings.Split(floatingWindow, "\n")
    
    for i := 0; i < len(floatingLines) && floatY+i < len(lines); i++ {
        if floatY+i >= 0 && floatY+i < len(lines) {
            line := lines[floatY+i]
            // Replace portion of line with floating content
            lines[floatY+i] = overlayLine(line, floatingLines[i], floatX)
        }
    }
    
    return strings.Join(lines, "\n")
}
```

### Step 3: Add Layout to Switch

```go
func (ri *ReasoningIntegration) View() string {
    switch ri.config.Layout {
    case LayoutBottom:
        return ri.renderSplitHorizontal()
    case LayoutRight:
        return ri.renderSplitVertical()
    case LayoutOverlay:
        return ri.renderOverlay()
    case LayoutSplit:
        return ri.renderSplit()
    case LayoutFloating:
        return ri.renderFloating()
    default:
        return ri.renderSplitHorizontal()
    }
}
```

## Performance Optimization

### 1. Token Counting Optimization

```go
// Use batch processing
func (tc *TokenCounter) CountMessagesBatch(
    messageGroups [][]ContextMessage,
) ([]int, error) {
    results := make([]int, len(messageGroups))
    errors := make([]error, len(messageGroups))
    
    var wg sync.WaitGroup
    for i, messages := range messageGroups {
        wg.Add(1)
        go func(idx int, msgs []ContextMessage) {
            defer wg.Done()
            count, err := tc.CountMessages(msgs)
            results[idx] = count
            errors[idx] = err
        }(i, messages)
    }
    
    wg.Wait()
    
    // Check for errors
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }
    
    return results, nil
}
```

### 2. Pattern Matching Optimization

```go
// Use trie for pattern matching
type PatternTrie struct {
    root *trieNode
    mu   sync.RWMutex
}

type trieNode struct {
    children map[ThinkingType]*trieNode
    patterns []*LearnedPattern
}

func (pt *PatternTrie) Insert(pattern *LearnedPattern) {
    pt.mu.Lock()
    defer pt.mu.Unlock()
    
    node := pt.root
    for _, thinkingType := range pattern.Signature {
        if node.children[thinkingType] == nil {
            node.children[thinkingType] = &trieNode{
                children: make(map[ThinkingType]*trieNode),
            }
        }
        node = node.children[thinkingType]
    }
    
    node.patterns = append(node.patterns, pattern)
}

func (pt *PatternTrie) Search(sequence []ThinkingType) []*LearnedPattern {
    pt.mu.RLock()
    defer pt.mu.RUnlock()
    
    var results []*LearnedPattern
    node := pt.root
    
    for _, thinkingType := range sequence {
        if node.children[thinkingType] != nil {
            node = node.children[thinkingType]
            results = append(results, node.patterns...)
        } else {
            break
        }
    }
    
    return results
}
```

### 3. Storage Query Optimization

```go
// Use prepared statements
type SQLiteStorage struct {
    db    *sql.DB
    stmts map[string]*sql.Stmt
}

func (s *SQLiteStorage) prepareStatements() error {
    queries := map[string]string{
        "get_chain": `
            SELECT data FROM reasoning_chains 
            WHERE id = $1
        `,
        "query_chains": `
            SELECT data FROM reasoning_chains 
            WHERE agent_id = $1 
            AND confidence >= $2 
            AND created_at BETWEEN $3 AND $4
            ORDER BY confidence DESC 
            LIMIT $5 OFFSET $6
        `,
    }
    
    for name, query := range queries {
        stmt, err := s.db.Prepare(query)
        if err != nil {
            return gerror.Wrap(err, gerror.ErrCodeStorage, 
                "failed to prepare statement").
                WithDetails("statement", name)
        }
        s.stmts[name] = stmt
    }
    
    return nil
}
```

## Testing Strategies

### 1. Unit Testing Patterns

```go
func TestThinkingBlockParser(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []*core.ThinkingBlock
        wantErr  bool
    }{
        {
            name: "single analysis block",
            input: `<thinking type="analysis">
                Analyzing the problem...
            </thinking>`,
            expected: []*core.ThinkingBlock{
                {
                    Type:    core.ThinkingTypeAnalysis,
                    Content: "Analyzing the problem...",
                },
            },
        },
        {
            name: "nested thinking blocks",
            input: `<thinking>
                Main thought
                <thinking type="hypothesis">
                    Nested hypothesis
                </thinking>
            </thinking>`,
            expected: []*core.ThinkingBlock{
                {
                    Type:    core.ThinkingTypeAnalysis,
                    Content: "Main thought",
                },
                {
                    Type:     core.ThinkingTypeHypothesis,
                    Content:  "Nested hypothesis",
                    ParentID: stringPtr("block_1"),
                },
            },
        },
    }
    
    parser := core.NewThinkingBlockParser()
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            blocks, err := parser.ParseThinkingBlocks(ctx, tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseThinkingBlocks() error = %v, wantErr %v", 
                    err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(blocks, tt.expected) {
                t.Errorf("ParseThinkingBlocks() = %v, want %v", 
                    blocks, tt.expected)
            }
        })
    }
}
```

### 2. Integration Testing

```go
func TestReasoningSystemIntegration(t *testing.T) {
    ctx := context.Background()
    
    // Setup test database
    tempDB := setupTestDB(t)
    defer tempDB.Close()
    
    // Create system
    config := core.ReasoningSystemConfig{
        DatabasePath: tempDB.Path(),
        StorageConfig: core.ReasoningStorageConfig{
            StorageBackend: "sqlite",
            RetentionDays:  1,
        },
    }
    
    system, err := core.NewReasoningSystem(ctx, config)
    require.NoError(t, err)
    defer system.Close(ctx)
    
    // Test reasoning extraction and storage
    response := generateTestResponse()
    
    extractor := system.Extractor
    chain, err := extractor.Extract(ctx, response)
    require.NoError(t, err)
    
    // Store chain
    err = system.Storage.Store(ctx, chain)
    require.NoError(t, err)
    
    // Query chain
    retrieved, err := system.Storage.Get(ctx, chain.ID)
    require.NoError(t, err)
    require.Equal(t, chain.ID, retrieved.ID)
    
    // Test pattern learning
    patterns, err := system.Analyzer.IdentifyPatterns(ctx, []*chain)
    require.NoError(t, err)
    require.NotEmpty(t, patterns)
}
```

### 3. Benchmarking

```go
func BenchmarkThinkingBlockParsing(b *testing.B) {
    parser := core.NewThinkingBlockParser()
    ctx := context.Background()
    
    // Generate test data
    testCases := []struct {
        name  string
        input string
    }{
        {"small", generateThinkingBlocks(1)},
        {"medium", generateThinkingBlocks(10)},
        {"large", generateThinkingBlocks(50)},
    }
    
    for _, tc := range testCases {
        b.Run(tc.name, func(b *testing.B) {
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _, err := parser.ParseThinkingBlocks(ctx, tc.input)
                if err != nil {
                    b.Fatal(err)
                }
            }
        })
    }
}

func BenchmarkTokenCounting(b *testing.B) {
    counters := map[string]core.TokenCounter{
        "openai":    core.NewOpenAITokenCounter("gpt-4"),
        "anthropic": core.NewAnthropicTokenCounter("claude-3"),
        "ollama":    core.NewOllamaTokenCounter("llama2"),
    }
    
    text := generateLongText(1000) // 1000 words
    
    for name, counter := range counters {
        b.Run(name, func(b *testing.B) {
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _, err := counter.CountTokens(text)
                if err != nil {
                    b.Fatal(err)
                }
            }
        })
    }
}
```

## Debugging Guide

### 1. Enable Debug Logging

```go
// Set log level
os.Setenv("GUILD_LOG_LEVEL", "debug")

// Add debug logs
logger := observability.GetLogger(ctx)
logger.DebugContext(ctx, "Processing thinking block",
    "block_id", block.ID,
    "type", block.Type,
    "confidence", block.Confidence,
    "token_count", block.TokenCount)
```

### 2. Reasoning Chain Inspection

```go
// Create debug helper
func DebugReasoningChain(chain *core.ReasoningChainEnhanced) {
    fmt.Printf("=== Reasoning Chain Debug ===\n")
    fmt.Printf("ID: %s\n", chain.ID)
    fmt.Printf("Agent: %s\n", chain.AgentID)
    fmt.Printf("Duration: %s\n", chain.EndTime.Sub(chain.StartTime))
    fmt.Printf("Total Tokens: %d\n", chain.TotalTokens)
    fmt.Printf("Final Confidence: %.2f\n", chain.FinalConfidence)
    
    fmt.Printf("\n=== Thinking Blocks ===\n")
    for i, block := range chain.Blocks {
        fmt.Printf("\n[Block %d]\n", i+1)
        fmt.Printf("Type: %s\n", block.Type)
        fmt.Printf("Confidence: %.2f\n", block.Confidence)
        fmt.Printf("Tokens: %d\n", block.TokenCount)
        fmt.Printf("Duration: %s\n", block.Duration)
        
        if len(block.DecisionPoints) > 0 {
            fmt.Printf("Decisions:\n")
            for _, dp := range block.DecisionPoints {
                fmt.Printf("  - %s (%.0f%%)\n", 
                    dp.Decision, dp.Confidence*100)
            }
        }
    }
    
    fmt.Printf("\n=== Quality Metrics ===\n")
    fmt.Printf("Coherence: %.2f\n", chain.Quality.Coherence)
    fmt.Printf("Completeness: %.2f\n", chain.Quality.Completeness)
    fmt.Printf("Depth: %.2f\n", chain.Quality.Depth)
    fmt.Printf("Overall: %.2f\n", chain.Quality.Overall)
    
    fmt.Printf("\n=== Insights ===\n")
    for _, insight := range chain.Insights {
        fmt.Printf("- [%s] %s (%.0f%%)\n", 
            insight.Type, insight.Description, 
            insight.Confidence*100)
    }
}
```

### 3. Pattern Learning Debug

```go
// Trace pattern learning
func (pl *PatternLearner) debugLearnPatterns(
    ctx context.Context, 
    chains []*core.ReasoningChainEnhanced,
) {
    logger := observability.GetLogger(ctx)
    
    logger.DebugContext(ctx, "Starting pattern learning",
        "chain_count", len(chains),
        "min_occurrences", pl.config.MinOccurrences)
    
    // Extract features
    features := pl.featureExtractor.ExtractBatch(chains)
    logger.DebugContext(ctx, "Extracted features",
        "feature_count", len(features))
    
    // Analyze sequences
    sequences := pl.analyzer.FindSequences(chains)
    logger.DebugContext(ctx, "Found sequences",
        "sequence_count", len(sequences))
    
    for _, seq := range sequences {
        logger.DebugContext(ctx, "Sequence candidate",
            "pattern", seq.Pattern,
            "occurrences", seq.Count,
            "confidence", seq.Confidence)
    }
}
```

## Best Practices

### 1. Error Handling

```go
// Always wrap errors with context
if err != nil {
    return gerror.Wrap(err, gerror.ErrCodeInternal, "operation failed").
        WithComponent("reasoning_system").
        WithOperation("ProcessReasoning").
        WithDetails("input_length", len(input))
}

// Use specific error codes
var (
    ErrTokenLimitExceeded = gerror.New(gerror.ErrCodeResourceLimit, 
        "token limit exceeded", nil)
    ErrInvalidConfidence = gerror.New(gerror.ErrCodeValidation, 
        "confidence must be between 0 and 1", nil)
)
```

### 2. Resource Management

```go
// Always clean up resources
func ProcessWithReasoning(ctx context.Context) error {
    // Acquire resources
    manager := pool.GetTokenManager()
    defer pool.ReturnTokenManager(manager)
    
    window, err := manager.CreateWindow(ctx, "temp", 8000)
    if err != nil {
        return err
    }
    defer manager.CloseWindow("temp")
    
    // Use resources
    // ...
    
    return nil
}
```

### 3. Concurrency Safety

```go
// Use channels for communication
type PatternLearner struct {
    tasks    chan learningTask
    results  chan learningResult
    shutdown chan struct{}
}

func (pl *PatternLearner) Start(workers int) {
    for i := 0; i < workers; i++ {
        go pl.worker()
    }
}

func (pl *PatternLearner) worker() {
    for {
        select {
        case task := <-pl.tasks:
            result := pl.processTask(task)
            pl.results <- result
        case <-pl.shutdown:
            return
        }
    }
}
```

### 4. Configuration Management

```go
// Use structured configuration
type Config struct {
    Reasoning ReasoningConfig `mapstructure:"reasoning"`
    Storage   StorageConfig   `mapstructure:"storage"`
    UI        UIConfig        `mapstructure:"ui"`
}

func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    viper.SetEnvPrefix("GUILD")
    viper.AutomaticEnv()
    
    // Set defaults
    viper.SetDefault("reasoning.enable_caching", true)
    viper.SetDefault("reasoning.cache_ttl", "5m")
    viper.SetDefault("storage.retention_days", 90)
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeConfig, 
            "failed to read config")
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeConfig, 
            "failed to unmarshal config")
    }
    
    return &config, nil
}
```

This implementation guide provides comprehensive instructions for working with and extending the reasoning system, ensuring developers can effectively build upon this foundation.
