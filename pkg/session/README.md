# Session Management Package

This package implements comprehensive session management for Guild Framework, fulfilling the requirements for Product Vision Sprint 6, Agent 2 task.

## Overview

The session management system provides:

- **Session Persistence**: SQLite-based storage with encryption, compression, and auto-save
- **Conversation Resume**: Full UI state restoration with agent reconnection and task continuation
- **Export/Import**: Support for JSON, Markdown, HTML, and PDF formats
- **Session Analytics**: Usage tracking, productivity insights, and dashboard integration

## Architecture

### Core Components

1. **SessionManager** (`persistence.go`)
   - Handles session CRUD operations
   - Provides encryption and compression
   - Implements auto-save functionality
   - Manages session state persistence

2. **SessionResumer** (`resume.go`)
   - Restores UI state on session resume
   - Reconnects agents to previous state
   - Continues interrupted tasks
   - Provides crash recovery

3. **SessionExporter** (`export.go`)
   - Exports sessions to multiple formats
   - Supports filtering and customization
   - Handles import operations
   - Validates import data

4. **SessionAnalytics** (`analytics.go`)
   - Tracks usage patterns and metrics
   - Generates productivity insights
   - Creates analytical reports
   - Provides dashboard integration

### Integration Interfaces

The package defines comprehensive interfaces (`interfaces.go`) for integration with:

- **Orchestrator**: Agent and task management
- **UI System**: State restoration and notifications
- **Storage Layer**: Persistent data management
- **Analytics Store**: Metrics and reporting data

## Key Features

### Session Persistence

- **Encryption**: AES-256-GCM encryption for sensitive data
- **Compression**: gzip compression for large session states
- **Auto-save**: Configurable intervals with change buffering
- **Transactions**: Atomic operations for data integrity

### Resume Functionality

- **UI State**: Restore scroll position, input buffer, command history
- **Agent State**: Reconnect agents with preserved context
- **Task Continuity**: Resume paused tasks and handle failures
- **Recovery**: Automatic crash recovery with unsaved changes

### Export/Import

- **Multiple Formats**: JSON, Markdown, HTML (PDF requires external tools)
- **Filtering**: Date range, agent-specific, and custom filters
- **Metadata**: Optional inclusion of session context and metadata
- **Validation**: Comprehensive import data validation

### Analytics

- **Usage Tracking**: Agent activity, command usage, token consumption
- **Productivity Metrics**: Session duration, task completion rates
- **Insights Generation**: Automated productivity and efficiency insights
- **Trend Analysis**: Historical productivity pattern analysis

## Usage Examples

### Basic Session Management

```go
// Create session manager
store := NewSQLiteSessionStore(db)
manager := NewSessionManager(store, 
    WithEncryption(encryptionKey),
    WithAutoSaveInterval(30*time.Second))

// Create session
session := &Session{
    ID:         "session-123",
    UserID:     "user-456", 
    CampaignID: "campaign-789",
    StartTime:  time.Now(),
    State:      SessionState{Status: SessionStatusActive},
}

// Save session
err := manager.SaveSession(ctx, session)
```

### Session Resume

```go
// Create resumer
resumer := NewSessionResumer(manager, uiRestorer, orchestrator, corpus)

// Resume session
err := resumer.ResumeSession(ctx, "session-123")
```

### Export Session

```go
// Create exporter
exporter := NewSessionExporter()

// Export as Markdown
data, err := exporter.Export(session, ExportOptions{
    Format:          ExportFormatMarkdown,
    IncludeMetadata: true,
    SyntaxHighlight: true,
})
```

### Session Analytics

```go
// Create analytics
analytics := NewSessionAnalytics(analyticsStore)

// Analyze session
report, err := analytics.AnalyzeSession(ctx, session)
fmt.Printf("Productivity Score: %.2f\n", report.ProductivityScore)
```

## Configuration

The package supports comprehensive configuration through:

- **SessionConfig**: Session duration limits, cleanup intervals
- **AutoSaveConfig**: Auto-save behavior and intervals
- **EncryptionConfig**: Encryption settings and key management
- **AnalyticsConfig**: Analytics tracking preferences
- **ExportConfig**: Default export formats and options

## Testing

Comprehensive test suite includes:

- **Unit Tests**: Individual component testing
- **Integration Tests**: Cross-component functionality
- **Mock Implementations**: For testing without dependencies
- **Round-trip Tests**: Export/import data integrity

Run tests with:
```bash
go test ./pkg/session/...
```

## Error Handling

The package uses Guild's `gerror` framework for consistent error handling:

- **Context Propagation**: All operations accept context.Context
- **Error Wrapping**: Detailed error context and codes
- **Graceful Degradation**: Continue operation when possible
- **Observability**: Comprehensive logging and metrics

## Future Enhancements

- **Cloud Storage**: Support for cloud-based session storage
- **Real-time Sync**: Multi-device session synchronization
- **Advanced Analytics**: Machine learning-based insights
- **Custom Formats**: Plugin system for additional export formats

## Dependencies

- **gerror**: Guild's error handling framework
- **storage**: Guild's storage abstraction layer
- **observability**: Logging and metrics framework
- **Standard Library**: crypto, compress, encoding packages

## Contributing

When contributing to this package:

1. Follow Guild's coding standards and naming conventions
2. Use context-first patterns for all operations
3. Include comprehensive tests for new functionality
4. Update documentation for public interfaces
5. Ensure error handling follows gerror patterns

## License

Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2