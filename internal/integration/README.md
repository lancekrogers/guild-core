# Integration Layer

This directory contains the integration layer that connects all Guild components into a cohesive system.

## Directory Structure

```
internal/integration/
├── bridges/      # Bridges between components (e.g., event-logger, ui-event)
├── services/     # Service lifecycle management and registry
├── tests/        # Integration test framework and suites
├── bootstrap/    # Application startup and shutdown sequences
└── config/       # Unified configuration management
```

## Integration Approach

All integration code follows these principles:

1. **Enhance, don't replace** - Build on existing components
2. **Context-aware** - Proper context propagation throughout
3. **Error handling** - Consistent use of gerror
4. **Testable** - Each integration point has comprehensive tests
5. **Observable** - Integration metrics and logging

## Key Interfaces

- `Service` - Lifecycle management for all components
- `EventBridge` - Component communication bridge
- `ServiceRegistry` - Central service coordination
- `Bootstrap` - Application startup/shutdown orchestration
