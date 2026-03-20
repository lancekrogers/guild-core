# LSP Tools Implementation Guide

This guide explains how to complete the implementation of the LSP tools once the LSP manager is extended with the necessary methods.

## Prerequisites

The following needs to be added to `pkg/lsp/`:

### 1. Manager Methods

Add these methods to `pkg/lsp/manager.go`:

- `GetDocumentSymbols(ctx, filePath) ([]DocumentSymbol, error)`
- `GetWorkspaceSymbols(ctx, query) ([]SymbolInformation, error)`
- `GetCodeActions(ctx, filePath, startLine, startChar, endLine, endChar) ([]CodeAction, error)`
- `RenameSymbol(ctx, filePath, line, character, newName) (*WorkspaceEdit, error)`
- `FormatDocument(ctx, filePath, options) ([]TextEdit, error)`

### 2. Protocol Types

Add missing types to `pkg/lsp/protocol.go`:

- `WorkspaceSymbolParams`
- `SymbolInformation`
- `CodeActionContext`
- `CodeAction`
- `WorkspaceEdit`
- `RenameParams`
- `DocumentFormattingParams`
- `FormattingOptions`

### 3. Public Methods

Either:

- Make `Client.request()` public by renaming to `Request()`
- OR add a public wrapper method to Server/Client

## Updating the Tools

Once the LSP manager is extended, update each tool:

### Document Symbols Tool

```go
// In tools/lsp/document_symbols.go

func (t *DocumentSymbolsTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    // Parse input
    var params DocumentParams
    if err := json.Unmarshal([]byte(input), &params); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
            WithComponent("lsp.document_symbols_tool").
            WithOperation("execute")
    }

    // Get symbols from LSP
    symbols, err := t.manager.GetDocumentSymbols(ctx, params.File)
    if err != nil {
        return nil, err
    }

    // Convert to our format
    result := convertDocumentSymbols(symbols)

    // Marshal result
    output, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
            WithComponent("lsp.document_symbols_tool").
            WithOperation("execute")
    }

    // Create metadata
    metadata := map[string]string{
        "file":         params.File,
        "symbol_count": fmt.Sprintf("%d", countSymbols(result)),
    }

    // Extract summary data
    extraData := map[string]interface{}{
        "has_classes":   hasSymbolKind(result, lsp.SymbolKindClass),
        "has_functions": hasSymbolKind(result, lsp.SymbolKindFunction, lsp.SymbolKindMethod),
        "has_variables": hasSymbolKind(result, lsp.SymbolKindVariable, lsp.SymbolKindConstant),
    }

    return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}

// Helper to convert LSP symbols to our format
func convertDocumentSymbols(symbols []lsp.DocumentSymbol) []*SymbolInfo {
    var result []*SymbolInfo
    for _, symbol := range symbols {
        info := &SymbolInfo{
            Name:   symbol.Name,
            Kind:   symbolKindToString(symbol.Kind),
            Detail: symbol.Detail,
            Range: &Range{
                StartLine:   symbol.Range.Start.Line,
                StartColumn: symbol.Range.Start.Character,
                EndLine:     symbol.Range.End.Line,
                EndColumn:   symbol.Range.End.Character,
            },
        }
        
        if symbol.SelectionRange != (lsp.Range{}) {
            info.SelectionRange = &Range{
                StartLine:   symbol.SelectionRange.Start.Line,
                StartColumn: symbol.SelectionRange.Start.Character,
                EndLine:     symbol.SelectionRange.End.Line,
                EndColumn:   symbol.SelectionRange.End.Character,
            }
        }
        
        // Recursively convert children
        if len(symbol.Children) > 0 {
            info.Children = convertDocumentSymbols(symbol.Children)
        }
        
        result = append(result, info)
    }
    return result
}
```

### Workspace Symbols Tool

```go
// In tools/lsp/workspace_symbols.go

func (t *WorkspaceSymbolsTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    // Parse input
    var params WorkspaceSymbolParams
    if err := json.Unmarshal([]byte(input), &params); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
            WithComponent("lsp.workspace_symbols_tool").
            WithOperation("execute")
    }

    // Search workspace
    symbols, err := t.manager.GetWorkspaceSymbols(ctx, params.Query)
    if err != nil {
        return nil, err
    }

    // Convert to our format
    result := convertWorkspaceSymbols(symbols)

    // Marshal result
    output, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
            WithComponent("lsp.workspace_symbols_tool").
            WithOperation("execute")
    }

    metadata := map[string]string{
        "query":        params.Query,
        "result_count": fmt.Sprintf("%d", len(result)),
    }

    // Group results by file
    fileCount := make(map[string]int)
    for _, symbol := range result {
        fileCount[symbol.Location.File]++
    }

    extraData := map[string]interface{}{
        "files_matched": len(fileCount),
        "file_counts":   fileCount,
    }

    return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}

func convertWorkspaceSymbols(symbols []lsp.SymbolInformation) []*WorkspaceSymbolInfo {
    var result []*WorkspaceSymbolInfo
    for _, symbol := range symbols {
        info := &WorkspaceSymbolInfo{
            Name:          symbol.Name,
            Kind:          symbolKindToString(symbol.Kind),
            ContainerName: symbol.ContainerName,
            Location: struct {
                File  string `json:"file"`
                Range *Range `json:"range"`
            }{
                File: uriToFilePath(symbol.Location.URI),
                Range: &Range{
                    StartLine:   symbol.Location.Range.Start.Line,
                    StartColumn: symbol.Location.Range.Start.Character,
                    EndLine:     symbol.Location.Range.End.Line,
                    EndColumn:   symbol.Location.Range.End.Character,
                },
            },
        }
        result = append(result, info)
    }
    return result
}
```

### Code Actions Tool

```go
// In tools/lsp/code_actions.go

func (t *CodeActionsTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    // Parse input
    var params CodeActionParams
    if err := json.Unmarshal([]byte(input), &params); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
            WithComponent("lsp.code_actions_tool").
            WithOperation("execute")
    }

    // Get code actions
    actions, err := t.manager.GetCodeActions(ctx, params.File, 
        params.Line, params.Column, 
        params.Line, params.Column) // Single position for now
    if err != nil {
        return nil, err
    }

    // Convert to our format
    result := convertCodeActions(actions)

    // Marshal result
    output, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
            WithComponent("lsp.code_actions_tool").
            WithOperation("execute")
    }

    metadata := map[string]string{
        "file":         params.File,
        "position":     fmt.Sprintf("%d:%d", params.Line, params.Column),
        "action_count": fmt.Sprintf("%d", len(result)),
    }

    // Extract action kinds
    kinds := make(map[string]int)
    for _, action := range result {
        if action.Kind != "" {
            kinds[action.Kind]++
        }
    }

    extraData := map[string]interface{}{
        "action_kinds": kinds,
        "has_quickfix": hasActionKind(actions, "quickfix"),
        "has_refactor": hasActionKind(actions, "refactor"),
    }

    return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}

func convertCodeActions(actions []lsp.CodeAction) []*CodeAction {
    var result []*CodeAction
    for _, action := range actions {
        ca := &CodeAction{
            Title:       action.Title,
            Kind:        action.Kind,
            IsPreferred: action.IsPreferred,
        }

        // Convert diagnostics
        for _, diag := range action.Diagnostics {
            ca.Diagnostics = append(ca.Diagnostics, diag.Message)
        }

        // Convert edit if present
        if action.Edit != nil {
            ca.Edit = convertWorkspaceEdit(action.Edit)
        }

        // Convert command if present
        if action.Command != nil {
            ca.Command = map[string]interface{}{
                "command": action.Command.Command,
                "title":   action.Command.Title,
            }
        }

        result = append(result, ca)
    }
    return result
}
```

### Rename Symbol Tool

```go
// In tools/lsp/rename_symbol.go

func (t *RenameSymbolTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    // Parse input
    var params RenameParams
    if err := json.Unmarshal([]byte(input), &params); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
            WithComponent("lsp.rename_symbol_tool").
            WithOperation("execute")
    }

    // Validate new name
    if params.NewName == "" {
        return nil, gerror.Newf(gerror.ErrCodeValidation, "new_name cannot be empty").
            WithComponent("lsp.rename_symbol_tool").
            WithOperation("execute")
    }

    // Perform rename
    edit, err := t.manager.RenameSymbol(ctx, params.File, params.Line, params.Column, params.NewName)
    if err != nil {
        return nil, err
    }

    // Convert to our format
    result := &RenameResult{
        Success: true,
        Changes: convertTextEdits(edit.Changes),
    }

    // Calculate summary
    result.Summary.TotalFiles = len(result.Changes)
    for _, edits := range result.Changes {
        result.Summary.TotalChanges += len(edits)
    }

    // Marshal result
    output, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
            WithComponent("lsp.rename_symbol_tool").
            WithOperation("execute")
    }

    metadata := map[string]string{
        "file":          params.File,
        "position":      fmt.Sprintf("%d:%d", params.Line, params.Column),
        "new_name":      params.NewName,
        "total_files":   fmt.Sprintf("%d", result.Summary.TotalFiles),
        "total_changes": fmt.Sprintf("%d", result.Summary.TotalChanges),
    }

    extraData := map[string]interface{}{
        "success":      result.Success,
        "files_affected": getFileList(result.Changes),
    }

    return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}
```

### Format Document Tool

```go
// In tools/lsp/format_document.go

func (t *FormatDocumentTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    // Parse input
    var params FormatParams
    if err := json.Unmarshal([]byte(input), &params); err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
            WithComponent("lsp.format_document_tool").
            WithOperation("execute")
    }

    // Set default options if not provided
    options := lsp.FormattingOptions{
        TabSize:      4,
        InsertSpaces: true,
    }
    
    if params.Options != nil {
        if tabSize, ok := params.Options["tabSize"].(float64); ok {
            options.TabSize = int(tabSize)
        }
        if insertSpaces, ok := params.Options["insertSpaces"].(bool); ok {
            options.InsertSpaces = insertSpaces
        }
    }

    // Format document
    edits, err := t.manager.FormatDocument(ctx, params.File, options)
    if err != nil {
        return nil, err
    }

    // Convert to our format
    result := &FormatResult{
        Success:   true,
        Formatted: len(edits) > 0,
        Edits:     convertTextEditList(edits),
    }

    // Calculate summary
    lines := make(map[int]bool)
    for _, edit := range result.Edits {
        for line := edit.Range.StartLine; line <= edit.Range.EndLine; line++ {
            lines[line] = true
        }
    }
    result.Summary.LinesChanged = len(lines)

    if result.Summary.LinesChanged > 0 {
        result.Summary.Message = fmt.Sprintf("Formatted %d lines", result.Summary.LinesChanged)
    } else {
        result.Summary.Message = "Document is already properly formatted"
    }

    // Marshal result
    output, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
            WithComponent("lsp.format_document_tool").
            WithOperation("execute")
    }

    metadata := map[string]string{
        "file":          params.File,
        "formatted":     fmt.Sprintf("%v", result.Formatted),
        "lines_changed": fmt.Sprintf("%d", result.Summary.LinesChanged),
        "tab_size":      fmt.Sprintf("%d", options.TabSize),
        "insert_spaces": fmt.Sprintf("%v", options.InsertSpaces),
    }

    extraData := map[string]interface{}{
        "formatted":   result.Formatted,
        "edit_count":  len(edits),
    }

    return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}
```

## Helper Functions

Add these helper functions to a shared file (e.g., `tools/lsp/helpers.go`):

```go
package lsp

import (
    "strings"
    "github.com/guild-ventures/guild-core/pkg/lsp"
)

// symbolKindToString converts LSP SymbolKind to string
func symbolKindToString(kind lsp.SymbolKind) string {
    names := map[lsp.SymbolKind]string{
        lsp.SymbolKindFile:          "File",
        lsp.SymbolKindModule:        "Module",
        lsp.SymbolKindNamespace:     "Namespace",
        lsp.SymbolKindPackage:       "Package",
        lsp.SymbolKindClass:         "Class",
        lsp.SymbolKindMethod:        "Method",
        lsp.SymbolKindProperty:      "Property",
        lsp.SymbolKindField:         "Field",
        lsp.SymbolKindConstructor:   "Constructor",
        lsp.SymbolKindEnum:          "Enum",
        lsp.SymbolKindInterface:     "Interface",
        lsp.SymbolKindFunction:      "Function",
        lsp.SymbolKindVariable:      "Variable",
        lsp.SymbolKindConstant:      "Constant",
        lsp.SymbolKindString:        "String",
        lsp.SymbolKindNumber:        "Number",
        lsp.SymbolKindBoolean:       "Boolean",
        lsp.SymbolKindArray:         "Array",
        lsp.SymbolKindObject:        "Object",
        lsp.SymbolKindKey:           "Key",
        lsp.SymbolKindNull:          "Null",
        lsp.SymbolKindEnumMember:    "EnumMember",
        lsp.SymbolKindStruct:        "Struct",
        lsp.SymbolKindEvent:         "Event",
        lsp.SymbolKindOperator:      "Operator",
        lsp.SymbolKindTypeParameter: "TypeParameter",
    }
    
    if name, ok := names[kind]; ok {
        return name
    }
    return fmt.Sprintf("Unknown(%d)", kind)
}

// uriToFilePath converts a file URI to a file path
func uriToFilePath(uri string) string {
    if strings.HasPrefix(uri, "file://") {
        return uri[7:]
    }
    return uri
}

// filePathToURI converts a file path to a URI
func filePathToURI(path string) string {
    return "file://" + path
}
```

## Testing

Create comprehensive tests for each tool:

```go
// tools/lsp/document_symbols_test.go
func TestDocumentSymbolsTool(t *testing.T) {
    // Mock LSP manager
    mockManager := &MockLSPManager{
        symbols: []lsp.DocumentSymbol{
            {
                Name: "TestFunction",
                Kind: lsp.SymbolKindFunction,
                Range: lsp.Range{
                    Start: lsp.Position{Line: 10, Character: 0},
                    End:   lsp.Position{Line: 15, Character: 1},
                },
            },
        },
    }
    
    tool := NewDocumentSymbolsTool(mockManager)
    
    input := `{"file": "/test/main.go"}`
    result, err := tool.Execute(context.Background(), input)
    
    assert.NoError(t, err)
    assert.Contains(t, result.Output, "TestFunction")
    assert.Equal(t, "1", result.Metadata["symbol_count"])
}
```

## Integration

Once all tools are complete, update the tool registry:

```go
// In pkg/registry/lsp_tools.go or similar

import (
    lsptools "github.com/guild-ventures/guild-core/tools/lsp"
)

func RegisterLSPTools(r Registry, lspManager *lsp.Manager) error {
    // Register new tools
    tools := []tools.Tool{
        lsptools.NewDocumentSymbolsTool(lspManager),
        lsptools.NewWorkspaceSymbolsTool(lspManager),
        lsptools.NewCodeActionsTool(lspManager),
        lsptools.NewRenameSymbolTool(lspManager),
        lsptools.NewFormatDocumentTool(lspManager),
    }
    
    for _, tool := range tools {
        if err := r.Register(tool); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Summary

The implementation is straightforward once the LSP manager is extended:

1. **Manager Extensions**: Add the 5 new methods to LSP Manager
2. **Protocol Types**: Add missing LSP protocol types
3. **Update Tools**: Replace placeholder implementations
4. **Add Helpers**: Create shared helper functions
5. **Test**: Write comprehensive tests
6. **Register**: Add tools to the registry

The architecture is already in place - it just needs the LSP manager methods to be implemented!
