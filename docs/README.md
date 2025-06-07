# Guild Framework Documentation

This directory contains comprehensive documentation for the Guild Framework.

## Documentation Structure

- `getting-started/` - Quick start guides and tutorials
- `architecture/` - System design and architecture documentation
  - `task-execution.md` - Task execution system with phases and prompts
- `api/` - API reference documentation
  - `executor.md` - Task executor API reference
- `features/` - Feature-specific documentation
  - `workspace-isolation.md` - Git worktree workspace isolation
- `examples/` - Example code and use cases
- `deployment/` - Deployment and configuration guides

## Viewing Documentation

### Online
Once published, documentation will be available at:
- https://pkg.go.dev/github.com/guild-ventures/guild-core (API docs)
- https://guild-ventures.github.io/guild-core (User guides)

### Local Development

1. **View API Documentation locally with pkgsite:**
   ```bash
   # Install pkgsite
   go install golang.org/x/pkgsite/cmd/pkgsite@latest

   # Run from guild-core directory
   pkgsite -http=:8080

   # Open http://localhost:8080/github.com/guild-ventures/guild-core
   ```

2. **Generate static documentation site:**
   ```bash
   # Using Hugo (recommended for Go projects)
   hugo server -D

   # Or using MkDocs
   mkdocs serve
   ```

## Writing Documentation

### API Documentation
- Write godoc comments for all exported types, functions, and packages
- First sentence should be a clear summary
- Use examples in `*_test.go` files with `Example` prefix

### User Documentation
- Write in Markdown format
- Include code examples
- Follow the structure in this directory

## Documentation Tools

- **pkgsite**: Modern replacement for godoc, used by pkg.go.dev
- **Hugo**: Static site generator popular in Go community
- **MkDocs**: Alternative with good search and theming options
