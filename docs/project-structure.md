# Guild Project Structure

## Overview

Guild uses a project-local directory structure similar to Git. When you initialize a Guild project, it creates a `.guild/` directory that contains all project-specific configuration, data, and state.

## The .guild Directory

The `.guild/` directory is the heart of a Guild project. It stores all project-local data and should typically be excluded from version control by adding it to `.gitignore`.

### Current Directory Structure

```
.guild/
├── guild.yaml         # Main guild configuration
├── memory.db         # SQLite database for state and memory
├── corpus/           # Knowledge base and documentation
│   └── docs/         # Indexed documents
├── objectives/       # Commission documents (formerly objectives)
│   └── refined/      # Refined commission outputs
├── kanban/           # File-based task board state
│   └── <commission_id>/
│       ├── review/   # Tasks requiring review
│       └── blocked/  # Blocked task details
├── archives/         # Agent memory and context
├── campaigns/        # Campaign definitions
└── prompts/          # Custom prompt templates
```

## File Descriptions

### guild.yaml

Main configuration file containing:
- Guild name and description
- Agent configurations
- Provider settings
- Model preferences

Example structure:
```yaml
name: "My Development Guild"
description: "A team for building web applications"

agents:
  - id: "backend-dev"
    name: "Backend Developer"
    model: "claude-3-sonnet-20240229"
    provider: "anthropic"
    # ... other agent config

providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
  openai:
    api_key: ${OPENAI_API_KEY}
```

### memory.db

SQLite database containing:
- Prompt chains and conversation history
- Task states and relationships
- Campaign state
- Agent session data

This replaced the previous BoltDB implementation for better relational data support.

### corpus/

The knowledge base for your project:
- **docs/**: Indexed project documentation
- Used by RAG (Retrieval-Augmented Generation) system
- Supports markdown and text files

### objectives/ (Commissions)

Project goals and refined outputs:
- Commission documents define work to be done
- **refined/**: AI-refined and structured versions
- Note: "objectives" directory name remains for compatibility

### kanban/

File-based task tracking system:
- Organized by commission ID
- **review/**: Tasks awaiting human review
- **blocked/**: Tasks with blockers
- Integrates with SQLite for state persistence

### archives/

Historical agent data:
- Conversation logs
- Context snapshots
- Agent memory traces

### campaigns/

Campaign workflow definitions:
- Campaign configurations
- State machines for campaign flow
- Integration with orchestrator

### prompts/

Custom prompt templates:
- Layer-specific prompts
- Project-specific instructions
- Agent role definitions

## Working with Project Structure

### Initialization

```bash
# Initialize in current directory
./bin/guild init

# Initialize with a specific path
./bin/guild init /path/to/project

# Note: Template initialization is not yet implemented
```

### Project Detection

Guild automatically detects project boundaries by looking for `.guild/` directories:

```bash
# From any subdirectory, Guild finds the project root
cd /my/project/src/components
guild chat  # Works from any subdirectory
```

### Configuration Hierarchy

Guild uses a hierarchical configuration system:

1. **Environment Variables**: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.
2. **Project Config**: `.guild/guild.yaml`
3. **Command-line flags**: `--flag value` (limited support)

## Current Limitations

Several planned features are not yet implemented:

- ❌ Global configuration (`~/.guild/config.yaml`)
- ❌ Multiple embedding providers
- ❌ Automatic caching system
- ❌ Export/import functionality
- ❌ `guild clean` command
- ❌ `guild doctor` diagnostic command
- ❌ Template-based initialization

## Best Practices

### Version Control

Add to `.gitignore`:

```gitignore
# Guild project directory
.guild/

# Or selectively ignore
.guild/memory.db
.guild/archives/
.guild/kanban/
```

You may want to track:
- `.guild/guild.yaml` - Project configuration
- `.guild/prompts/` - Custom prompts
- `.guild/campaigns/` - Campaign definitions

### Security Considerations

The `.guild/` directory may contain:
- API keys (in guild.yaml via environment variables)
- Conversation history in memory.db
- Document content in corpus

Always ensure:
- Use environment variables for API keys
- Proper file permissions on `.guild/`
- Never commit sensitive data

## Troubleshooting

### Common Issues

1. **"Not in a guild project" error**
   - Ensure you're in a directory with `.guild/` or a parent with it
   - Run `guild init` to create a new project

2. **Database errors**
   - Check that SQLite is installed
   - Ensure `.guild/memory.db` has write permissions
   - Database migrations run automatically on init

3. **Missing commands**
   - Many commands shown in other docs are not implemented
   - Check `guild --help` for available commands

### Available Diagnostic Commands

```bash
# Check if in a guild project (limited functionality)
guild info

# View corpus statistics
guild corpus scan --dry-run
```

## Migration from Older Versions

If you have an older Guild project structure:

1. The framework has migrated from BoltDB to SQLite
2. "Objectives" are now called "Commissions"
3. Many planned directories (embeddings/, cache/, etc.) were never implemented

Currently, there is no automated migration tool. You would need to:
1. Back up your old `.guild/` directory
2. Run `guild init` to create new structure
3. Manually copy relevant files

---

**Note**: This document reflects the current implementation. Many features described in other documentation are planned but not yet implemented.