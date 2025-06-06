# Prompts Package Structure

This package provides two prompt management systems for the Guild framework:

## Directory Structure

```
internal/prompts/
├── interface.go          # Common interfaces for all prompt systems
├── factory.go           # Factory for creating prompt managers
├── README.md            # This file
├── layered/             # Advanced 6-layer prompt system
│   ├── manager.go       # Main layered prompt manager
│   ├── assembler.go     # Assembles prompts from layers
│   ├── registry.go      # Registry for layered prompts
│   ├── types.go         # Type definitions
│   ├── interfaces.go    # Layered-specific interfaces
│   ├── context/         # Context formatting utilities
│   └── commission/      # Commission-specific prompts (renamed from objective)
├── standard/            # Standard template-based system
│   ├── manager.go       # Template manager
│   ├── loader.go        # Template loader
│   ├── templates/       # Markdown prompt templates
│   │   ├── agent/       # Agent-specific prompts
│   │   ├── manager/     # Manager prompts
│   │   └── commission/  # Commission prompts (renamed from objective)
│   └── evaluation/      # Prompt evaluation framework
└── adapters/            # Adapters between systems
    └── layered_adapter.go

```

## Prompt Systems

### 1. Standard System (`standard/`)
- Template-based prompt rendering
- Markdown templates with metadata
- Simple variable substitution
- Good for static prompts

### 2. Layered System (`layered/`) - KEY INNOVATION
The layered prompt system is one of Guild's key innovations, providing a sophisticated 6-layer hierarchy:

1. **Platform Layer** - Core Guild platform rules (safety, terms of service)
2. **Guild Layer** - Project-wide goals and style guidelines
3. **Role Layer** - Agent role definitions (Guild Master, Code Artisan, etc.)
4. **Domain Layer** - Project type specializations (web-app, cli-tool, etc.)
5. **Session Layer** - User preferences and session context
6. **Turn Layer** - Ephemeral instructions for single interactions

This allows for:
- Dynamic prompt composition
- Context-aware behavior modification
- Runtime prompt customization
- Efficient token usage through layer priorities

## Usage

```go
// Create a standard prompt manager
stdManager, err := prompts.NewStandardManager()

// Create a layered prompt manager
layeredManager, err := prompts.NewLayeredManager()

// Use the factory with configuration
manager, err := prompts.NewManager(ctx, prompts.ManagerConfig{
    Type: prompts.TypeLayered,
    LayeredConfig: &prompts.LayeredManagerConfig{
        DefaultPlatformPrompt: "...",
        DefaultGuildPrompt: "...",
    },
})
```

## Migration Status

- ✅ Merged `prompts` and `prompts_pkg` into unified structure
- ✅ Renamed to clarify `layered` vs `standard` systems
- ✅ Created adapters for compatibility
- ⚠️ Need to update imports throughout codebase
- ⚠️ Need to complete objective → commission rename in templates

## TODO

1. Update all imports from old paths to new structure
2. Complete objective → commission terminology update
3. Implement proper template rendering in layered adapter
4. Add tests for the new structure
5. Update documentation