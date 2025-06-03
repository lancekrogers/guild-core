# Understanding Guild Projects

## Project-Local Philosophy

Guild follows a project-local approach similar to Git. Each project has its own isolated configuration, data, and state stored in a `.guild/` directory. This design provides several benefits:

- **Isolation**: Multiple projects can coexist without interference
- **Portability**: Projects can be moved or shared (excluding `.guild/`)
- **Flexibility**: Each project can have different configurations
- **Security**: Sensitive data stays within project boundaries

## Key Concepts

### Project Root
The directory containing `.guild/` is considered the project root. All Guild commands work relative to this root, regardless of your current directory within the project.

### Automatic Detection
Guild automatically searches for `.guild/` in the current directory and all parent directories, similar to how Git finds `.git/`.

### Data Locality
All project data stays within `.guild/`:
- Knowledge corpus for RAG
- Agent conversation history
- Task states and progress
- Embeddings and vector data
- Cost tracking information

## Comparison with Similar Tools

| Feature | Guild | Git | Docker |
|---------|-------|-----|---------|
| Local directory | `.guild/` | `.git/` | `Dockerfile` |
| Auto-detection | ✓ | ✓ | ✗ |
| Hierarchical config | ✓ | ✓ | ✗ |
| Isolated environments | ✓ | ✓ | ✓ |
| Version control friendly | ✓ | N/A | ✓ |

## Practical Examples

### Starting a New Project

```bash
# Create a new web app project with Guild
mkdir my-webapp
cd my-webapp
guild init

# This creates:
# my-webapp/
# └── .guild/
#     ├── config.yaml
#     ├── corpus/
#     ├── embeddings/
#     └── ...
```

### Working from Subdirectories

```bash
# Guild works from any subdirectory
cd my-webapp/src/components
guild status              # ✓ Works
guild agent list          # ✓ Works
guild objective create    # ✓ Works
```

### Multiple Projects

```bash
# Each project is isolated
~/projects/webapp/.guild/      # Web app agents and config
~/projects/api/.guild/         # API agents and config
~/projects/ml-model/.guild/    # ML agents and config

# Switch between projects naturally
cd ~/projects/webapp && guild status
cd ~/projects/api && guild status
```

## Configuration Cascade

Guild looks for configuration in this order:

1. **Command-line flags** (highest priority)
   ```bash
   guild run --model gpt-4
   ```

2. **Environment variables**
   ```bash
   export GUILD_MODEL=gpt-4
   ```

3. **Project config** (`.guild/config.yaml`)
   ```yaml
   model: gpt-4
   provider: openai
   ```

4. **Global config** (`~/.guild/config.yaml`)
   ```yaml
   model: gpt-3.5-turbo
   provider: openai
   ```

5. **Defaults** (lowest priority)

## Data Management

### What Goes in .guild/

**Stored in .guild/:**
- Processed documents and embeddings
- Agent state and memory
- Task history and progress
- Cache and temporary files
- Project-specific configuration

**Stored in project root:**
- `guild.yaml` - Project definition (version controlled)
- `objectives/` - Objective definitions (version controlled)
- Source code and documents

### Backup Considerations

Essential to backup:
- `.guild/memory/` - Project memory and state
- `.guild/corpus/documents/` - Original documents
- `.guild/config.yaml` - Project configuration

Can be regenerated:
- `.guild/embeddings/` - Can be recreated from corpus
- `.guild/cache/` - Temporary data
- `.guild/logs/` - Can be rotated

## Common Patterns

### Development Workflow

```bash
# Morning routine
cd ~/work/project
guild status                    # Check project state
guild objective list --active   # See what needs doing
guild run                       # Start agents working

# During development
guild corpus add docs/          # Add new documentation
guild agent logs -f             # Monitor agent activity

# End of day
guild objective complete obj-123
guild status --summary
```

### Team Collaboration

Share these files (via Git):
- `guild.yaml` - Project definition
- `objectives/*.md` - Objective definitions
- `docs/` - Documentation for corpus

Don't share (add to .gitignore):
- `.guild/` - Local state and data

### CI/CD Integration

```yaml
# .github/workflows/guild.yml
- name: Initialize Guild
  run: guild init
  
- name: Add documentation to corpus
  run: guild corpus add docs/
  
- name: Run validation
  run: guild validate
```

## Tips and Best Practices

1. **Initialize early**: Run `guild init` when starting a new project
2. **Document objectives**: Keep objective files in version control
3. **Regular cleanup**: Use `guild clean` to remove old caches
4. **Monitor costs**: Check `guild status --costs` regularly
5. **Backup memory**: Periodically backup `.guild/memory/guild.db`

## Next Steps

- [Configuration Guide](./configuration.md) - Detailed configuration options
- [Creating Objectives](./objectives.md) - Writing effective objectives
- [Agent Management](./agents.md) - Working with agents