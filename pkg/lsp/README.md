# LSP Package - Framework Infrastructure

This package provides Language Server Protocol (LSP) infrastructure for the Guild Framework. It manages language server lifecycles, connections, and protocol communication.

## Architecture Overview

The LSP package is **framework infrastructure**, not a tool. It provides:

- **Server Lifecycle Management**: Starting/stopping language servers
- **Connection Management**: Maintaining persistent connections
- **Protocol Implementation**: JSON-RPC communication with language servers
- **State Management**: Tracking open files and server state
- **Multi-Language Support**: Automatic language detection and server selection

## Framework vs Tools

### This Package (Infrastructure)

```go
// Long-running service with state
type Manager struct {
    servers map[string]*Server  // Persistent connections
    config  *Config            // Configuration
    mu      sync.RWMutex       // Thread safety
}
```

### Tools (User Actions)

```go
// Stateless wrappers that use the infrastructure
type CompletionTool struct {
    manager *lsp.Manager  // Uses the service
}
```

## Using LSP as a Framework

The LSP package is designed to be used by external projects to build their own tools:

### Basic Usage

```go
import "github.com/guild-ventures/guild-core/pkg/lsp"

// Create manager
manager, err := lsp.NewManager(configPath)

// Use for code intelligence
completions, err := manager.GetCompletion(ctx, file, line, col)
definition, err := manager.GetDefinition(ctx, file, line, col)
references, err := manager.GetReferences(ctx, file, line, col, true)
hover, err := manager.GetHover(ctx, file, line, col)
```

### Example: Custom Completion Tool

```go
// Your custom tool with caching and AI enhancement
type SmartCompletionTool struct {
    manager *lsp.Manager
    cache   *CompletionCache
    ai      *AIEnhancer
}

func (t *SmartCompletionTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    // Check cache first
    if cached := t.cache.Get(input); cached != nil {
        return cached, nil
    }
    
    // Get LSP completions
    completions, err := t.manager.GetCompletion(ctx, file, line, col)
    if err != nil {
        return nil, err
    }
    
    // Enhance with AI
    enhanced := t.ai.RankByContext(completions, t.getContext(file))
    filtered := t.ai.FilterIrrelevant(enhanced)
    
    // Cache and return
    result := t.formatResult(filtered)
    t.cache.Store(input, result)
    
    return result, nil
}
```

### Example: Security Analysis Tool

```go
// Domain-specific tool for security analysis
type SecurityScanTool struct {
    manager *lsp.Manager
    scanner *VulnerabilityScanner
}

func (t *SecurityScanTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    // Use LSP to understand code structure
    symbols, err := t.manager.GetDocumentSymbols(ctx, file)
    if err != nil {
        return nil, err
    }
    
    // Find function definitions
    functions := extractFunctions(symbols)
    
    // Security analysis
    vulnerabilities := []Vulnerability{}
    for _, fn := range functions {
        // Check for SQL injection patterns
        if vuln := t.scanner.CheckSQLInjection(fn); vuln != nil {
            vulnerabilities = append(vulnerabilities, vuln)
        }
        
        // Check for unsafe operations
        if vuln := t.scanner.CheckUnsafeOps(fn); vuln != nil {
            vulnerabilities = append(vulnerabilities, vuln)
        }
    }
    
    return t.formatSecurityReport(vulnerabilities), nil
}
```

### Example: Batch Operations Tool

```go
// Different philosophy: one tool for multiple operations
type BatchLSPTool struct {
    manager *lsp.Manager
}

type BatchParams struct {
    File           string `json:"file"`
    Operations     []string `json:"operations"`
    IncludeMetrics bool `json:"include_metrics"`
}

func (t *BatchLSPTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    var params BatchParams
    json.Unmarshal([]byte(input), &params)
    
    results := map[string]interface{}{}
    metrics := map[string]time.Duration{}
    
    for _, op := range params.Operations {
        start := time.Now()
        
        switch op {
        case "symbols":
            results["symbols"] = t.manager.GetDocumentSymbols(ctx, params.File)
        case "outline":
            results["outline"] = t.buildOutline(ctx, params.File)
        case "diagnostics":
            results["diagnostics"] = t.getDiagnostics(ctx, params.File)
        case "imports":
            results["imports"] = t.analyzeImports(ctx, params.File)
        }
        
        if params.IncludeMetrics {
            metrics[op] = time.Since(start)
        }
    }
    
    return &tools.ToolResult{
        Data: results,
        Metadata: map[string]string{
            "operations": strings.Join(params.Operations, ","),
            "duration": fmt.Sprintf("%v", time.Since(start)),
        },
    }, nil
}
```

### Example: Refactoring Assistant

```go
// High-level refactoring tool using LSP
type RefactoringTool struct {
    manager  *lsp.Manager
    analyzer *CodeAnalyzer
}

func (t *RefactoringTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
    var params RefactorParams
    json.Unmarshal([]byte(input), &params)
    
    switch params.Type {
    case "extract_method":
        return t.extractMethod(ctx, params)
    case "inline_variable":
        return t.inlineVariable(ctx, params)
    case "rename_safe":
        return t.safeRename(ctx, params)
    }
}

func (t *RefactoringTool) extractMethod(ctx context.Context, params RefactorParams) (*tools.ToolResult, error) {
    // Use LSP to understand code structure
    symbols := t.manager.GetDocumentSymbols(ctx, params.File)
    
    // Analyze selected code
    selection := t.analyzer.AnalyzeSelection(params.StartLine, params.EndLine)
    
    // Find variables used
    refs := t.manager.GetReferences(ctx, params.File, selection.Variables)
    
    // Generate method signature
    signature := t.analyzer.GenerateMethodSignature(selection, refs)
    
    // Create refactoring plan
    plan := RefactorPlan{
        NewMethod: signature,
        Replacements: t.calculateReplacements(selection, refs),
        Imports: t.requiredImports(selection),
    }
    
    return t.formatPlan(plan), nil
}
```

## Extending the Manager

To add new LSP operations, extend the Manager:

```go
// In pkg/lsp/manager.go

// GetDocumentSymbols returns all symbols in a document
func (m *Manager) GetDocumentSymbols(ctx context.Context, filePath string) ([]DocumentSymbol, error) {
    server, err := m.GetServerForFile(ctx, filePath)
    if err != nil {
        return nil, err
    }
    
    if err := m.ensureFileOpened(ctx, server, filePath); err != nil {
        return nil, err
    }
    
    params := &DocumentSymbolParams{
        TextDocument: TextDocumentIdentifier{
            URI: FilePathToURI(filePath),
        },
    }
    
    var result []DocumentSymbol
    if err := server.Client.Request(ctx, "textDocument/documentSymbol", params, &result); err != nil {
        return nil, err
    }
    
    return result, nil
}
```

## Configuration

Language servers are configured in `~/.guild/lsp/config.yaml`:

```yaml
servers:
  go:
    command: ["gopls"]
    root_markers: ["go.mod"]
    file_patterns: ["*.go"]
  
  python:
    command: ["pylsp"]
    root_markers: ["setup.py", "pyproject.toml"]
    file_patterns: ["*.py"]
  
  typescript:
    command: ["typescript-language-server", "--stdio"]
    root_markers: ["package.json", "tsconfig.json"]
    file_patterns: ["*.ts", "*.tsx", "*.js", "*.jsx"]
```

## Benefits of This Architecture

1. **Flexibility**: Build tools that match your workflow
2. **Performance**: Shared language server connections
3. **Consistency**: Single source of truth for LSP operations
4. **Extensibility**: Easy to add new language servers
5. **Reliability**: Managed lifecycle and error recovery

## See Also

- `tools/lsp/` - Guild's default LSP tool implementations
- `pkg/lsp/tools/` - Core LSP tools (completion, definition, etc.)
- Language Server Protocol specification: https://microsoft.github.io/language-server-protocol/
