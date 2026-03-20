# SuggestionService Integration Guide

## Overview

The `SuggestionService` provides intelligent, context-aware suggestions for the chat interface. It integrates with the existing `ChatService` to enhance user interactions with:

- Pre-execution suggestions
- Follow-up suggestions
- Token-optimized context building
- Performance-optimized caching
- Concurrent-safe operations

## Key Features

### 1. Token Optimization

- Configurable token budgets
- Automatic context truncation
- Token usage tracking
- Cost-aware operations

### 2. Smart Caching

- TTL-based cache management
- Cache hit/miss tracking
- Periodic cleanup
- Thread-safe operations

### 3. Context Building

- Conversation history integration
- File context support
- User preference handling
- Follow-up suggestion generation

## Integration Example

```go
// Create an enhanced agent with suggestion capabilities
enhancedAgent := agent.NewSuggestionAwareWorkerAgent(...)

// Create the integrated service
chatWithSuggestions, err := services.NewChatWithSuggestions(
    ctx,
    chatService,
    enhancedAgent,
)

// Send a message with suggestions
cmd := chatWithSuggestions.SendMessageWithSuggestions("agent-id", "How do I implement authentication?")

// The service will:
// 1. Get pre-execution suggestions (e.g., "Consider using JWT", "OAuth2 might be suitable")
// 2. Send the message to the agent
// 3. Generate follow-up suggestions based on the response
```

## Configuration

```go
// Configure the suggestion service
service.SetTokenBudget(4096)           // Max tokens for context
service.SetCacheTTL(5 * time.Minute)   // Cache duration
service.SetConfig(agent.ChatSuggestionConfig{
    EnableSuggestions:    true,
    DefaultMaxResults:    5,
    DefaultMinConfidence: 0.5,
    EnabledTypes: []suggestions.SuggestionType{
        suggestions.SuggestionTypeCommand,
        suggestions.SuggestionTypeFollowUp,
        suggestions.SuggestionTypeTemplate,
    },
})
```

## Suggestion Modes

The integrated service supports different suggestion modes:

- **`pre`**: Only provide suggestions before execution
- **`post`**: Only provide follow-up suggestions after execution
- **`both`**: Provide suggestions both before and after (default)
- **`none`**: Disable suggestions

```go
err := chatWithSuggestions.SetSuggestionMode("both")
```

## Performance Considerations

1. **Caching**: Suggestions are cached to reduce API calls and improve response times
2. **Token Limits**: Context is automatically optimized to stay within token budgets
3. **Concurrent Access**: All operations are thread-safe for concurrent usage
4. **Async Operations**: Suggestions are fetched asynchronously using Tea commands

## Statistics and Monitoring

```go
stats := service.GetStats()
// Returns:
// - total_requests: Total suggestion requests
// - cache_hits/misses: Cache performance
// - cache_hit_rate: Percentage of cache hits
// - token_used: Total tokens consumed
// - avg_latency: Average response time
```

## Testing

The service includes comprehensive tests covering:

- Basic functionality
- Error handling
- Cache management
- Concurrent access
- Token optimization
- Performance benchmarks

Run tests with:

```bash
go test -v ./internal/chat/v2/services -run "Test.*Suggestion"
```

## Future Enhancements

1. **Provider Fallback**: Use multiple suggestion providers with fallback
2. **Learning**: Track accepted/rejected suggestions to improve quality
3. **Custom Providers**: Plugin architecture for custom suggestion sources
4. **Streaming**: Real-time suggestion updates during long operations
