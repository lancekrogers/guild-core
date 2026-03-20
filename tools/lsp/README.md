# LSP Tools

This directory contains Language Server Protocol (LSP) tools that provide advanced code intelligence features to AI agents.

## Available Tools

### Document Symbols Tool

Lists all symbols (functions, classes, variables) in a document.

- **Use case**: Understanding code structure, navigation
- **Input**: File path
- **Output**: Hierarchical symbol tree

### Workspace Symbols Tool

Search for symbols across the entire workspace.

- **Use case**: Finding definitions across multiple files
- **Input**: Symbol query
- **Output**: List of matching symbols with locations

### Code Actions Tool

Get available code refactorings and fixes at a position.

- **Use case**: Automated refactoring, fixing issues
- **Input**: File path, line, column
- **Output**: Available actions (extract method, add imports, etc.)

### Rename Symbol Tool

Rename a symbol across all its usages.

- **Use case**: Safe refactoring
- **Input**: File path, line, column, new name
- **Output**: List of files and changes

### Format Document Tool

Format code according to language standards.

- **Use case**: Code cleanup, standardization
- **Input**: File path
- **Output**: Formatted content or edits

## Implementation Notes

These tools extend the existing LSP functionality in `pkg/lsp/` by adding new operations not yet exposed through the manager. They follow the same pattern as existing LSP tools but are placed here to keep the framework organized.

## Usage

All tools follow the standard Guild tool interface and can be registered with the tool registry:

```go
import "github.com/guild-ventures/guild-core/tools/lsp"

// Register tools
registry.Register(lsp.NewDocumentSymbolsTool(lspManager))
registry.Register(lsp.NewWorkspaceSymbolsTool(lspManager))
// ... etc
```
