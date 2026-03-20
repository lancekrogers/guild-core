# LSP Tools TODO

## Current Status

The LSP tools have been created with placeholder implementations. To make them fully functional, the following needs to be done:

## Required LSP Manager Extensions

The `pkg/lsp/Manager` type needs to be extended with the following methods:

### 1. Document Symbols

```go
func (m *Manager) GetDocumentSymbols(ctx context.Context, filePath string) ([]DocumentSymbol, error)
```

### 2. Workspace Symbols  

```go
func (m *Manager) GetWorkspaceSymbols(ctx context.Context, query string) ([]SymbolInformation, error)
```

### 3. Code Actions

```go
func (m *Manager) GetCodeActions(ctx context.Context, filePath string, range Range) ([]CodeAction, error)
```

### 4. Rename Symbol

```go
func (m *Manager) RenameSymbol(ctx context.Context, filePath string, line, character int, newName string) (*WorkspaceEdit, error)
```

### 5. Format Document

```go
func (m *Manager) FormatDocument(ctx context.Context, filePath string, options FormattingOptions) ([]TextEdit, error)
```

## Implementation Steps

1. **Update pkg/lsp/client.go**: Make the `request` method public or add a public wrapper
2. **Update pkg/lsp/manager.go**: Add the new methods listed above
3. **Update pkg/lsp/protocol.go**: Add any missing protocol types (most are already there)
4. **Update tools/lsp/*.go**: Replace placeholder implementations with actual LSP calls
5. **Add tests**: Create comprehensive tests for each new tool

## Why These Tools Matter

- **Document Symbols**: Essential for code navigation and understanding structure
- **Workspace Symbols**: Critical for finding definitions across projects
- **Code Actions**: Enables automated refactoring and fixes
- **Rename Symbol**: Safe refactoring across entire codebase
- **Format Document**: Maintains code consistency

## Current Workaround

The tools are currently created with placeholder implementations that:

1. Validate inputs
2. Return example data showing the expected structure
3. Include metadata indicating they need LSP manager extensions

This allows the tools to be registered and tested while the LSP manager is being extended.
