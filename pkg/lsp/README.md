# Guild LSP Integration

The Guild framework includes powerful Language Server Protocol (LSP) integration that provides zero-token code intelligence. This means agents can perform code completions, find definitions, locate references, and get type information without sending any file content - resulting in 97.5% token savings!

## Overview

The LSP integration allows Guild agents to:
- Get code completions at any position
- Navigate to symbol definitions
- Find all references to a symbol
- Get hover information (types, documentation)
- All without transmitting file content!

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Guild Agent   │────▶│   LSP Manager   │────▶│ Language Server │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ├── Go (gopls)
                               ├── TypeScript (typescript-language-server)
                               ├── Python (pylsp)
                               ├── Rust (rust-analyzer)
                               └── More...
```

## Supported Languages

Out of the box, Guild supports LSP for:
- **Go** - via `gopls`
- **TypeScript/JavaScript** - via `typescript-language-server`
- **Python** - via `pylsp`
- **Rust** - via `rust-analyzer`
- **Java** - via `jdtls`
- **C#** - via `omnisharp`

## Installation

### Language Server Installation

Before using LSP features, install the language servers you need:

```bash
# Go
go install golang.org/x/tools/gopls@latest

# TypeScript/JavaScript
npm install -g typescript typescript-language-server

# Python
pip install python-lsp-server

# Rust
rustup component add rust-analyzer

# Java (requires manual download)
# Download from: https://download.eclipse.org/jdtls/

# C#
# Download OmniSharp from: https://github.com/OmniSharp/omnisharp-roslyn
```

## Usage

### Using LSP Tools Directly

```go
import (
    "github.com/guild-ventures/guild-core/pkg/lsp"
    lsptools "github.com/guild-ventures/guild-core/tools/lsp"
)

// Create LSP manager
manager, err := lsp.NewManager("")
if err != nil {
    log.Fatal(err)
}
defer manager.Shutdown(context.Background())

// Create completion tool
completionTool := lsptools.NewCompletionTool(manager)

// Get completions
input := `{"file": "/path/to/main.go", "line": 10, "column": 15}`
result, err := completionTool.Execute(ctx, input)
```

### Using with Agents

The LSP integration is automatically available to agents through the LSP-aware executor:

```go
// Create LSP-aware executor
executor := executor.NewLSPAwareExecutor(baseExecutor, lspManager)

// Agent can now use LSP tools
task := Task{
    Tool: "lsp_completion",
    Input: map[string]interface{}{
        "file": "/path/to/file.go",
        "line": 20,
        "column": 10,
    },
}
```

## Configuration

LSP configuration is stored in `~/.guild/lsp/config.yaml`:

```yaml
servers:
  go:
    command: ["gopls", "serve"]
    init_options:
      usePlaceholders: true
      completeUnimported: true
    file_patterns: ["*.go"]
    root_markers: ["go.mod", "go.sum"]
    
  typescript:
    command: ["typescript-language-server", "--stdio"]
    init_options:
      preferences:
        includeCompletionsWithSnippetText: true
    file_patterns: ["*.ts", "*.tsx", "*.js", "*.jsx"]
    root_markers: ["package.json", "tsconfig.json"]
```

## Token Savings Example

Traditional approach (sending file content):
```json
{
  "tool": "analyze_code",
  "file_content": "// 2000 tokens of file content here...",
  "position": {"line": 50, "column": 10}
}
```
**Tokens used: ~2000**

LSP approach:
```json
{
  "tool": "lsp_completion",
  "file": "/path/to/file.go",
  "line": 50,
  "column": 10
}
```
**Tokens used: ~50**

**Savings: 97.5%!**

## Advanced Features

### Lifecycle Management

The LSP manager automatically:
- Starts language servers on demand
- Pools servers for reuse
- Cleans up idle servers
- Monitors server health
- Restarts failed servers

### Context Enhancement

The LSP context enhancer automatically adds rich context to agent tasks:
- Symbol types and signatures
- Related files and imports
- Project structure information
- All without sending file content!

## Troubleshooting

### Server Not Starting

1. Ensure the language server is installed and in PATH
2. Check the server command in config.yaml
3. Look for errors in the Guild logs

### No Completions/Results

1. Ensure the file has been saved (LSP works on saved files)
2. Check that the project root is correctly detected
3. Verify the language server supports the requested feature

### Performance Issues

1. Adjust idle timeout in lifecycle manager
2. Increase memory limits for language servers
3. Use server pooling for better performance

## Contributing

To add support for a new language:

1. Add default configuration in `config.go`
2. Add language detection in `DetectLanguage()`
3. Test with the language server
4. Submit a PR!

## License

The LSP integration is part of the Guild framework and follows the same license.