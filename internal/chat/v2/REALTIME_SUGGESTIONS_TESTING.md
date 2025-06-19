# Real-Time Suggestions Testing Guide

This document provides instructions for testing the real-time suggestion system in the Guild Framework chat interface.

## What We've Implemented

1. **InputSuggestionManager**: A dedicated manager that handles real-time suggestions with debouncing
   - 300ms debounce delay to avoid overwhelming the system
   - Intelligent filtering based on input changes
   - Relevance scoring for better suggestion quality

2. **Enhanced InputPane Integration**:
   - Connected input changes to trigger suggestions
   - Proper handling of stale suggestions
   - Visual feedback in the completion popup

3. **Completion Flow**:
   ```
   User Types → InputPane.OnChange → Debounce (300ms) → CompletionEngine → Show Suggestions
   ```

## Testing Instructions

### 1. Build and Run
```bash
cd guild-core
make build
./guild chat --campaign "test"
```

### 2. Test Real-Time Suggestions

#### Command Suggestions
- Type `/` and wait ~300ms
- You should see command suggestions appear
- Continue typing `/hel` - suggestions should filter
- Use Tab/Shift+Tab or arrow keys to navigate

#### Agent Mentions
- Type `@` and wait ~300ms
- Agent suggestions should appear
- Type `@age` - list should filter to matching agents

#### Context-Aware Suggestions
- Type partial tool names (e.g., `file`)
- Type partial variable names from context
- Type partial commands

### 3. Performance Testing

#### Debouncing
- Type quickly: `Hello world`
- Suggestions should only appear after you stop typing for 300ms
- Delete characters quickly - suggestions should hide immediately

#### Navigation
- When suggestions appear:
  - Tab: Next suggestion
  - Shift+Tab: Previous suggestion
  - Enter: Accept current suggestion
  - Esc: Hide suggestions
  - Arrow keys: Navigate up/down

### 4. Edge Cases to Test

1. **Empty Input**:
   - Clear input with Ctrl+L
   - Suggestions should hide

2. **Rapid Changes**:
   - Type and delete quickly
   - System should remain responsive

3. **Long Suggestions**:
   - Trigger suggestions with many results
   - UI should handle scrolling/truncation

4. **Stale Suggestions**:
   - Type slowly, then quickly change input
   - Old suggestions shouldn't appear

## Expected Behavior

### Visual Indicators
- **Normal Mode**: Purple rounded border
- **Completion Mode**: Bright purple border with "⚡ Completions" indicator
- **Selected Suggestion**: Highlighted with background color

### Suggestion Display
```
⚡ Guild Suggestions
─────────────────────
⚡ 💻 /help  Show available commands
  🤖 @agent-name  Mention an agent
  🔨 file-read  Read file contents
⚡ 1 of 3 suggestions
```

### Performance Metrics
- Suggestion latency: < 50ms after debounce
- Memory usage: Minimal increase
- CPU usage: No noticeable impact

## Troubleshooting

### Suggestions Not Appearing
1. Check if completion is enabled in config
2. Verify CompletionEngine is initialized
3. Check for errors in the console

### Slow Suggestions
1. Increase debounce delay if needed
2. Check if file indexing is complete
3. Verify no blocking operations

### UI Issues
1. Ensure terminal supports required colors
2. Check terminal dimensions (min 80x24)
3. Try different terminal emulators

## Future Enhancements

1. **Contextual Ranking**: Improve relevance based on recent commands
2. **Learning**: Track accepted suggestions to improve ranking
3. **Custom Completions**: Allow plugins to add suggestions
4. **Fuzzy Matching**: Better partial match support
5. **Async Loading**: Progressive suggestion loading for large datasets

## Code Locations

- **InputSuggestionManager**: `internal/chat/v2/input_suggestions.go`
- **InputPane**: `internal/chat/v2/panes/input.go`
- **CompletionEngine**: `internal/chat/v2/completion_enhanced.go`
- **Integration**: `internal/chat/v2/app.go` (setupInputCallbacks)

## Summary

The real-time suggestion system is now fully integrated with:
- ✅ Debounced input handling
- ✅ Visual feedback in the UI
- ✅ Keyboard navigation
- ✅ Relevance scoring
- ✅ Stale suggestion prevention

Test all the scenarios above to ensure the system works smoothly!