# Guild Project Structure

## Overview

Guild uses a project-local directory structure similar to Git. When you initialize a Guild project, it creates a `.guild/` directory that contains all project-specific configuration, data, and state.

## The .guild Directory

The `.guild/` directory is the heart of a Guild project. It stores all project-local data and should be excluded from version control by adding it to `.gitignore`.

### Directory Structure

```
.guild/
├── config.yaml        # Project-specific configuration
├── corpus/           # Knowledge base and documentation
│   ├── documents/    # Source documents
│   ├── chunks/       # Processed text chunks
│   └── metadata/     # Document metadata
├── embeddings/       # Vector embeddings for RAG
│   ├── openai/       # OpenAI embeddings
│   ├── ollama/       # Ollama embeddings
│   └── anthropic/    # Anthropic embeddings
├── agents/           # Agent configurations
│   ├── templates/    # Agent templates
│   └── instances/    # Active agent instances
├── objectives/       # Project objectives
│   ├── active/       # Currently active objectives
│   ├── completed/    # Completed objectives
│   └── templates/    # Objective templates
├── memory/           # BoltDB storage
│   ├── guild.db      # Main database
│   └── backups/      # Database backups
├── cache/            # Temporary files and caches
│   ├── llm/          # LLM response cache
│   └── tools/        # Tool output cache
└── logs/             # Project logs
    ├── agents/       # Agent activity logs
    └── system/       # System logs
```

## File Descriptions

### config.yaml
Project-specific configuration that overrides global settings:
- Provider configurations
- Model preferences
- Tool settings
- Cost budgets
- Feature flags

### corpus/
The knowledge base for your project:
- **documents/**: Original source documents (markdown, text, code)
- **chunks/**: Processed chunks for efficient retrieval
- **metadata/**: Document relationships and metadata

### embeddings/
Vector embeddings organized by provider:
- Supports multiple embedding providers
- Automatic fallback to different providers
- Cached to avoid recomputation

### agents/
Agent configurations and state:
- **templates/**: Reusable agent configurations
- **instances/**: Running agent instances with their state

### objectives/
Project goals and tasks:
- **active/**: Currently being worked on
- **completed/**: Historical record of completed work
- **templates/**: Reusable objective patterns

### memory/
BoltDB database containing:
- Task states
- Agent conversations
- Tool execution history
- Cost tracking data

### cache/
Temporary data that can be safely deleted:
- LLM responses (for retry/resume)
- Tool outputs
- Intermediate processing results

## Working with Project Structure

### Initialization

```bash
# Initialize in current directory
guild init

# Initialize with a specific path
guild init /path/to/project

# Initialize with template
guild init --template webapp
```

### Project Detection

Guild automatically detects project boundaries by looking for `.guild/` directories, similar to how Git works:

```bash
# From any subdirectory, Guild finds the project root
cd /my/project/src/components
guild status  # Works from any subdirectory
```

### Configuration Hierarchy

Guild uses a hierarchical configuration system:

1. **Global Config**: `~/.guild/config.yaml`
2. **Project Config**: `.guild/config.yaml`
3. **Environment Variables**: `GUILD_*`
4. **Command-line flags**: `--flag value`

Each level overrides the previous, allowing fine-grained control.

## Best Practices

### Version Control

Add to `.gitignore`:
```gitignore
# Guild project directory
.guild/
```

However, you may want to track certain templates:
```gitignore
# Ignore all of .guild except templates
.guild/*
!.guild/agents/templates/
!.guild/objectives/templates/
```

### Backup Strategy

Important data to backup:
- `.guild/memory/guild.db` - Your project's memory
- `.guild/corpus/documents/` - Source documents
- `.guild/config.yaml` - Project configuration

### Security Considerations

The `.guild/` directory may contain:
- API keys (in config.yaml)
- Sensitive document content
- Conversation history

Always ensure proper file permissions and never commit `.guild/` to public repositories.

## Migration and Portability

### Exporting a Project

```bash
# Export project data (excludes cache and logs)
guild export --output project-backup.tar.gz
```

### Importing a Project

```bash
# Import project data
guild import project-backup.tar.gz
```

### Cleaning Up

```bash
# Remove cache and temporary files
guild clean

# Remove all project data (careful!)
guild clean --all
```

## Troubleshooting

### Common Issues

1. **"Not in a guild project" error**
   - Ensure you're in a directory with `.guild/` or a parent with it
   - Run `guild init` to create a new project

2. **Permission errors**
   - Check that `.guild/` has proper write permissions
   - Ensure your user owns the directory

3. **Disk space issues**
   - Run `guild clean` to remove caches
   - Check `.guild/logs/` size
   - Consider pruning old embeddings

### Diagnostic Commands

```bash
# Check project status
guild status

# Verify project integrity
guild doctor

# Show disk usage
guild info --disk-usage
```
