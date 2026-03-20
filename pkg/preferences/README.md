# Guild Preferences Package

A hierarchical preference management system for the Guild framework that supports system, user, campaign, guild, and agent-level configurations with inheritance.

## Features

- **Hierarchical Preference System**: System → User → Campaign → Guild → Agent inheritance
- **Type-Safe Preferences**: Built-in validation for known preference types
- **In-Memory Caching**: High-performance cache with TTL and automatic cleanup
- **Bulk Operations**: Set and retrieve multiple preferences efficiently
- **Import/Export**: JSON-based preference export and import
- **Default Values**: Comprehensive defaults for all system preferences
- **Custom Validation**: Extensible validation framework for preferences

## Architecture

The preferences system follows Guild's standard architecture patterns:

```
Service Layer (pkg/preferences/)
    ↓
Repository Layer (pkg/storage/preferences_repository.go)
    ↓
Database Layer (SQLite)
```

## Usage

### Basic Operations

```go
// Initialize service
repo := storageRegistry.GetPreferencesRepository()
service := preferences.NewService(repo)

// Set system preference
err := service.SetSystemPreference(ctx, "ui.theme", "dark")

// Get user preference
theme, err := service.GetUserPreference(ctx, userID, "ui.theme")

// Set campaign preference
err := service.SetCampaignPreference(ctx, campaignID, "guild.maxAgents", 10)

// Set guild preference
err := service.SetGuildPreference(ctx, guildID, "agent.timeout", 3600)

// Set agent preference
err := service.SetAgentPreference(ctx, agentID, "agent.verbose", true)
```

### Preference Resolution

The preference system resolves values through the inheritance hierarchy:

```go
// Resolve preference through full hierarchy
value, err := service.ResolvePreference(ctx, "agent.timeout",
    &agentID,    // Most specific
    &guildID,
    &campaignID,
    &userID)     // Least specific (system is always included)

// The system will return the most specific value found
```

### Bulk Operations

```go
// Set multiple preferences at once
prefs := map[string]interface{}{
    "ui.theme": "light",
    "ui.fontSize": 16,
    "ui.autoSave": true,
}
err := service.SetPreferences(ctx, "user", &userID, prefs)

// Get multiple preferences
keys := []string{"ui.theme", "ui.fontSize", "ui.autoSave"}
values, err := service.GetPreferences(ctx, "user", &userID, keys)
```

### Import/Export

```go
// Export preferences to JSON
data, err := service.ExportPreferences(ctx, "campaign", &campaignID)

// Import preferences from JSON
err = service.ImportPreferences(ctx, "campaign", &campaignID, data)
```

## Preference Categories

### UI Preferences

- `ui.theme`: Theme selection (light/dark/auto)
- `ui.language`: UI language (ISO 639-1 code)
- `ui.fontSize`: Font size (8-32)
- `ui.showLineNumbers`: Show line numbers in editors
- `ui.wordWrap`: Enable word wrapping
- `ui.autoSave`: Enable auto-save
- `ui.autoSaveInterval`: Auto-save interval in seconds

### Agent Preferences

- `agent.maxConcurrent`: Maximum concurrent agents
- `agent.timeout`: Agent timeout in seconds
- `agent.retryAttempts`: Number of retry attempts
- `agent.retryDelay`: Delay between retries
- `agent.verbose`: Enable verbose logging
- `agent.autoAssign`: Auto-assign tasks to agents

### Guild Preferences

- `guild.maxAgents`: Maximum agents per guild
- `guild.coordinationMode`: Coordination mode (collaborative/hierarchical/autonomous)
- `guild.loadBalancing`: Load balancing strategy (round-robin/least-loaded/random/weighted)
- `guild.healthCheckInterval`: Health check interval in seconds

### Memory Preferences

- `memory.maxWorkingSize`: Maximum working memory items
- `memory.promotionThreshold`: Importance threshold for promotion (0-1)
- `memory.retentionDays`: Days to retain memories
- `memory.compressionEnabled`: Enable memory compression
- `memory.vectorDimensions`: Vector embedding dimensions

### Session Preferences

- `session.autoRestore`: Auto-restore sessions on restart
- `session.checkpointInterval`: Checkpoint interval in seconds
- `session.maxHistory`: Maximum message history
- `session.compressionEnabled`: Enable session compression
- `session.encryptionEnabled`: Enable session encryption

### Provider Preferences

- `provider.default`: Default LLM provider
- `provider.maxRetries`: Maximum retry attempts
- `provider.timeout`: Provider timeout in seconds
- `provider.temperature`: Generation temperature (0-2)
- `provider.maxTokens`: Maximum tokens per request

### Development Preferences

- `dev.debug`: Enable debug mode
- `dev.logLevel`: Log level (debug/info/warn/error)
- `dev.profiling`: Enable profiling
- `dev.metricsEnabled`: Enable metrics collection
- `dev.tracingEnabled`: Enable distributed tracing

## Validation

The system provides automatic validation for known preferences:

```go
// Theme validation - must be "light", "dark", or "auto"
err := service.SetSystemPreference(ctx, "ui.theme", "invalid")
// Returns error: invalid theme

// Font size validation - must be between 8 and 32
err := service.SetSystemPreference(ctx, "ui.fontSize", 50)
// Returns error: fontSize must be between 8 and 32

// Temperature validation - must be between 0 and 2
err := service.SetSystemPreference(ctx, "provider.temperature", 3.0)
// Returns error: temperature must be between 0 and 2
```

## Custom Preferences

You can store custom preferences not defined in the defaults:

```go
// Set custom preference
err := service.SetUserPreference(ctx, userID, "custom.myPref", "custom-value")

// Custom preferences bypass validation but maintain type consistency
```

## Caching

The preference service includes an efficient in-memory cache:

- Default TTL: 5 minutes
- Automatic cleanup: Every 10 minutes
- Cache invalidation on updates
- Thread-safe operations

## Database Schema

Preferences are stored in two tables:

### preferences

- `id`: Unique identifier
- `scope`: Preference scope (system/user/campaign/guild/agent)
- `scope_id`: ID of the scoped entity (NULL for system)
- `key`: Preference key
- `value`: JSON-encoded value
- `version`: Optimistic locking version
- `metadata`: Additional metadata
- `created_at`: Creation timestamp
- `updated_at`: Update timestamp

### preference_inheritance

- `id`: Unique identifier
- `child_scope`: Child scope type
- `child_scope_id`: Child scope ID
- `parent_scope`: Parent scope type
- `parent_scope_id`: Parent scope ID
- `priority`: Resolution priority
- `created_at`: Creation timestamp

## Performance Considerations

- Preferences are cached in-memory for fast access
- Bulk operations reduce database round trips
- Indexes on scope, scope_id, and key for efficient queries
- JSON storage allows flexible value types
- Optimistic locking prevents concurrent update conflicts

## Best Practices

1. **Use Appropriate Scopes**: Set preferences at the most appropriate level
2. **Leverage Defaults**: Rely on system defaults when possible
3. **Validate Custom Preferences**: Add validation for custom preferences
4. **Batch Operations**: Use bulk operations for multiple preferences
5. **Cache Awareness**: Be aware of cache TTL when expecting immediate updates

## Integration with Guild

The preferences system integrates seamlessly with other Guild components:

```go
// In agent initialization
timeout, _ := prefService.ResolvePreference(ctx, "agent.timeout",
    &agent.ID, &guild.ID, &campaign.ID, &user.ID)

// In UI components
theme, _ := prefService.GetUserPreference(ctx, userID, "ui.theme")

// In session management
autoRestore, _ := prefService.GetSystemPreference(ctx, "session.autoRestore")
```

## Testing

The package includes comprehensive tests covering:

- CRUD operations for all scope levels
- Preference inheritance resolution
- Validation rules
- Bulk operations
- Import/export functionality
- Cache effectiveness
- Concurrent access

Run tests with:

```bash
go test ./pkg/preferences/...
```
