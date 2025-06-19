# Suggestion System Integration Guide

This document explains how the suggestion system is integrated into the chat application and how to verify it's working correctly.

## Architecture Overview

The suggestion system consists of several key components:

1. **SuggestionAwareAgentFactory** (`pkg/agent/suggestion_factory.go`)
   - Creates agents with built-in suggestion capabilities
   - Configures suggestion providers (commands, tools, templates, follow-ups)
   - Manages the suggestion manager instance

2. **ChatSuggestionHandler** (`pkg/agent/chat_integration.go`)
   - Bridges between chat interface and agent suggestions
   - Handles suggestion requests and responses
   - Provides configuration and filtering capabilities

3. **Enhanced Agents** (`pkg/agent/enhanced_agent.go`)
   - Agents that implement the EnhancedGuildArtisan interface
   - Have integrated suggestion managers
   - Can provide context-aware suggestions

4. **Chat App Integration** (`internal/chat/v2/app.go`)
   - Initializes suggestion-aware agents in `initializeSuggestionSystem()`
   - Connects agents to chat service
   - Integrates with completion engine

## Integration Flow

1. **App Initialization**:
   ```go
   app.initializeComponents()
     -> app.initializeSuggestionSystem()
        -> Creates SuggestionAwareAgentFactory
        -> Creates enhanced agent with suggestions
        -> Creates ChatSuggestionHandler
   ```

2. **Service Integration**:
   ```go
   app.initializeServices()
     -> If enhanced agent exists:
        -> services.NewChatServiceWithSuggestions()
     -> Else:
        -> services.NewChatService()
   ```

3. **Completion Engine Integration**:
   ```go
   app.completionEngine.SetEnhancedAgent(agent, handler)
     -> Links suggestion manager to completion engine
     -> Enables agent-based suggestions in completions
   ```

## Verification Steps

### 1. Build Verification
```bash
cd guild-core
make build
```
Should compile without errors.

### 2. Run Tests
```bash
cd guild-core
go test ./internal/chat/v2 -run TestSuggestionIntegration -v
```

### 3. Manual Testing

Start the chat interface:
```bash
./guild chat --campaign "test"
```

#### Test Suggestion Triggers:

1. **Command Suggestions**:
   - Type `/` and wait ~300ms
   - Should see command suggestions appear

2. **Agent Mentions**:
   - Type `@` and wait ~300ms
   - Should see available agents

3. **Tool Suggestions**:
   - Type partial tool names
   - Should see matching tools

4. **Context-Aware Suggestions**:
   - After executing a command, should see follow-up suggestions
   - Suggestions should be relevant to current context

### 4. Debug Output

To verify the suggestion system is active:

1. Check initialization logs:
   ```
   Warning: Failed to get LLM client for suggestions: ...
   ```
   This is expected if no LLM provider is configured but shows the system tried to initialize.

2. Check for enhanced agent creation:
   - The app should have `enhancedAgent` field populated
   - The `chatHandler` should be non-nil

3. Check completion engine:
   - `completionEngine.suggestionManager` should be set
   - `completionEngine.chatHandler` should be set

## Configuration

The suggestion system can be configured through:

1. **Agent Factory Configuration**:
   ```go
   factory.ConfigureSuggestionProviders(
       templateManager,
       lspManager,
       customProviders...
   )
   ```

2. **Chat Service Configuration**:
   - `enableSuggestions`: Enable/disable suggestions
   - `suggestionMode`: Control when suggestions appear
   - `tokenBudget`: Limit suggestion token usage

3. **Completion Engine Settings**:
   - Debounce delay (300ms default)
   - Max results per provider
   - Confidence thresholds

## Troubleshooting

### Suggestions Not Appearing

1. **Check Dependencies**:
   - Ensure registry is initialized
   - Verify LLM client is available
   - Check memory manager is configured

2. **Check Integration**:
   - Verify `initializeSuggestionSystem()` runs without errors
   - Ensure enhanced agent is created
   - Confirm chat service uses suggestion variant

3. **Check Providers**:
   - Verify suggestion providers are registered
   - Check provider configurations
   - Ensure providers return suggestions

### Performance Issues

1. **Adjust Debounce**:
   - Increase debounce delay for slower systems
   - Default is 300ms, try 500ms or higher

2. **Limit Suggestions**:
   - Reduce max suggestions per provider
   - Filter by confidence threshold
   - Disable expensive providers (LSP, AI-based)

3. **Token Optimization**:
   - Monitor token usage in chat service
   - Adjust token budget limits
   - Use caching for repeated queries

## Missing Pieces

Currently, some components have placeholder implementations:

1. **Cost Manager**: Simple implementation, no actual cost tracking
2. **Commission Manager**: Minimal implementation for interface compliance
3. **Tool Registry Adapter**: Type mismatch requires copying tools
4. **Provider Metadata**: Not exposed through suggestion manager interface

These can be enhanced as the system matures.