# ChatService with SuggestionService Integration

This document describes the integration of the SuggestionService into the ChatService for token optimization and intelligent suggestions.

## Overview

The ChatService has been enhanced with integrated suggestion capabilities that provide:
- **Pre-execution suggestions**: Suggestions before sending messages to agents
- **Post-execution suggestions**: Follow-up suggestions after receiving responses
- **Token optimization**: Automatic message truncation to fit within token budgets
- **Caching**: Suggestion caching for improved performance

## Key Features

### 1. Token Optimization
The ChatService automatically optimizes messages to fit within configured token budgets:
```go
// Set token budget
chatService.SetTokenBudget(4096)

// Long messages are automatically truncated
cmd := chatService.SendMessage("agent", veryLongMessage)
```

### 2. Suggestion Modes
Four modes control when suggestions are generated:
- `SuggestionModeNone`: No suggestions
- `SuggestionModePre`: Only before execution
- `SuggestionModePost`: Only after execution
- `SuggestionModeBoth`: Both before and after (default)

```go
chatService.SetSuggestionMode(SuggestionModeBoth)
```

### 3. Integrated Suggestion Flow
```go
// Send message with automatic suggestion handling
cmd := chatService.SendMessageWithSuggestions(
    "developer",                    // agent ID
    "How do I implement a REST API?", // message
    "conv-123"                      // conversation ID
)
```

## Usage Examples

### Basic Integration
```go
// Create chat service with suggestions
chatService, err := NewChatServiceWithSuggestions(
    ctx, 
    grpcClient, 
    registry, 
    enhancedAgent
)

// Configure behavior
chatService.SetSuggestionMode(SuggestionModeBoth)
chatService.SetTokenBudget(4096)
```

### Manual Suggestion Handling
```go
// Get pre-execution suggestions
preCmd := chatService.GetPreExecutionSuggestions(message, conversationID)

// Get post-execution suggestions
postCmd := chatService.GetPostExecutionSuggestions(originalMessage, response)

// Process response with suggestions
processCmd := chatService.ProcessAgentResponse(agentResponse, originalMessage)
```

## Architecture

### Integration Points

1. **ChatService Changes**:
   - Added `suggestionService` field
   - Added `suggestionMode` and `enableSuggestions` configuration
   - Added `tokenBudget` and `tokenUsed` tracking
   - Enhanced `SendMessage` with token optimization
   - New `SendMessageWithSuggestions` method

2. **New Methods**:
   - `SetSuggestionService()`: Attach suggestion service
   - `SetSuggestionMode()`: Configure suggestion behavior
   - `GetPreExecutionSuggestions()`: Get suggestions before execution
   - `GetPostExecutionSuggestions()`: Get follow-up suggestions
   - `ProcessAgentResponse()`: Process response with suggestions
   - `SetTokenBudget()`: Configure token limits

3. **Message Types**:
   - `ChatMessageWithSuggestionsMsg`: Message with suggestions
   - `AgentResponseWithSuggestionsMsg`: Response with follow-up suggestions

### Token Optimization Flow

1. User sends message to ChatService
2. If suggestions enabled, message is optimized via `OptimizeContext()`
3. Token usage is tracked for analytics
4. Optimized message sent to agent
5. Response processed with optional follow-up suggestions

## Statistics and Monitoring

The integrated service provides comprehensive statistics:
```go
stats := chatService.GetStats()
// Includes:
// - suggestions_enabled: bool
// - suggestion_mode: string
// - token_budget: int
// - token_used: int
// - suggestion_* stats from SuggestionService
```

## Testing

Comprehensive tests cover:
- Basic integration setup
- All suggestion modes
- Token optimization
- Error handling
- Nil service handling
- Statistics tracking

Run tests:
```bash
go test ./internal/chat/v2/services/
```

## Future Enhancements

1. **Advanced Token Management**:
   - Per-agent token budgets
   - Dynamic budget adjustment
   - Token usage predictions

2. **Suggestion Quality**:
   - Confidence-based filtering
   - User preference learning
   - Context-aware suggestion ranking

3. **Performance Optimization**:
   - Parallel suggestion generation
   - Predictive caching
   - Background suggestion updates