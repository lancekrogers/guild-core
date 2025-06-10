# Guild LSP Integration

The Guild framework includes powerful Language Server Protocol (LSP) integration that provides zero-token code intelligence. This means agents can perform code completions, find definitions, locate references, and get type information without sending any file content - resulting in 97.5% token savings!

## Key Features

- **Universal Language Support**: Works with ANY LSP-compliant language server
- **Zero File Content**: All operations use only file path and position
- **97.5% Token Savings**: Dramatically reduces API costs
- **Extensible Configuration**: Add any language server via simple YAML config
- **Automatic Management**: Servers start/stop/restart automatically
- **Agent Integration**: Seamlessly integrated with Guild's agent system

## How It Works

The LSP integration allows Guild to use the same language servers that power VS Code, Neovim, and other modern editors. If a language has an LSP server, Guild can use it!

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Guild Agent   │────▶│   LSP Manager   │────▶│ Language Server │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ├── ANY LSP Server You Install
                              ├── gopls, rust-analyzer, pylsp...
                              ├── clangd, typescript-language-server...
                              └── Your custom language server!
```

## Configuration

LSP servers are configured in `~/.guild/lsp/config.yaml`. The system starts with an empty configuration - you add only what you need!

### Adding a Language Server

To add support for any language, create or edit `~/.guild/lsp/config.yaml`:

```yaml
servers:
  # Example: Go with gopls
  go:
    language: go
    command: ["gopls", "serve"]
    file_patterns: ["*.go"]
    root_markers: ["go.mod", "go.sum"]
    init_options:
      usePlaceholders: true
      completeUnimported: true

  # Example: Python with pyright
  python:
    language: python
    command: ["pyright-langserver", "--stdio"]
    file_patterns: ["*.py", "*.pyi"]
    root_markers: ["pyproject.toml", "setup.py", "requirements.txt"]

  # Example: Your custom language
  my-language:
    language: mylang
    command: ["/path/to/my-language-server", "--lsp"]
    file_patterns: ["*.mylang", "*.ml"]
    root_markers: ["project.mylang"]
    environment:
      MYLANG_HOME: "/opt/mylang"
```

### Configuration Options

- **language**: Language identifier (any string you choose)
- **command**: Command to start the LSP server
- **file_patterns**: Glob patterns for files this server handles
- **root_markers**: Files that indicate project root directory
- **init_options**: Server-specific initialization options
- **environment**: Environment variables for the server process

## Installing Language Servers

You can install language servers using your preferred package manager, similar to setting up your editor:

```bash
# Examples of installing popular language servers:

# Go
go install golang.org/x/tools/gopls@latest

# TypeScript/JavaScript  
npm install -g typescript typescript-language-server

# Python (multiple options)
pip install python-lsp-server[all]  # or
pip install pyright                  # or
pip install jedi-language-server

# Rust
rustup component add rust-analyzer

# C/C++
# Ubuntu/Debian: apt install clangd
# macOS: brew install llvm
# Or download from: https://clangd.llvm.org/

# Ruby
gem install solargraph

# PHP
npm install -g intelephense

# ... and many more!
```

For a comprehensive list of available language servers, see:
- https://microsoft.github.io/language-server-protocol/implementors/servers/
- https://langserver.org/

## Usage Examples

### Direct Usage

```go
import (
    "github.com/guild-ventures/guild-core/pkg/lsp"
    lsptools "github.com/guild-ventures/guild-core/pkg/lsp/tools"
)

// Create LSP manager (loads config from ~/.guild/lsp/config.yaml)
manager, err := lsp.NewManager("")
defer manager.Shutdown(context.Background())

// Use any configured language server
completionTool := lsptools.NewCompletionTool(manager)

// Works with any file type you've configured!
result, _ := completionTool.Execute(ctx, `{
    "file": "/path/to/code.anything",
    "line": 10,
    "column": 15
}`)
```

### Agent Integration

Agents automatically get LSP tools for all configured languages:

```go
// The agent executor automatically detects code tasks
// and uses LSP tools when available
task := Task{
    Description: "Add error handling to the parse function",
    Tool: "lsp_hover",  // Works for ANY configured language
    Input: map[string]interface{}{
        "file": "/project/parser.rs",  // Or .py, .ts, .java, etc.
        "line": 42,
        "column": 10,
    },
}
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

LSP approach (works with ANY language):
```json
{
  "tool": "lsp_completion",
  "file": "/path/to/any/supported/file.ext",
  "line": 50,
  "column": 10
}
```
**Tokens used: ~50**

**Savings: 97.5%!**

## Available LSP Tools

All tools work with ANY configured language server:

1. **lsp_completion**: Get code completions at a position
2. **lsp_definition**: Navigate to symbol definition
3. **lsp_references**: Find all references to a symbol
4. **lsp_hover**: Get type info and documentation

## Advanced Features

### Multi-Language Projects

Guild automatically detects the correct language server based on file extensions:

```yaml
servers:
  # Frontend
  typescript:
    file_patterns: ["*.ts", "*.tsx", "*.js", "*.jsx"]
    
  # Backend
  go:
    file_patterns: ["*.go"]
    
  # Scripts
  python:
    file_patterns: ["*.py"]
    
  # Documentation
  markdown:
    file_patterns: ["*.md", "*.mdx"]
```

### Custom Language Support

Have a domain-specific language? Just add its LSP server:

```yaml
servers:
  company-dsl:
    language: company-dsl
    command: ["/opt/company/bin/dsl-language-server"]
    file_patterns: ["*.dsl", "*.rules"]
    root_markers: ["project.dsl", ".dsl-config"]
    init_options:
      dialect: "v2"
      strict: true
```

## Troubleshooting

### Language Not Detected

1. Check that file patterns match in config.yaml
2. Ensure the language server is configured
3. Verify file extension is included in patterns

### Server Not Starting

1. Verify the language server is installed: `which <server-command>`
2. Check the command in config.yaml matches installation
3. Look for errors in Guild logs
4. Test the server manually: `<server-command> --help`

### Adding New Languages

1. Find an LSP server for your language
2. Install it using the recommended method
3. Add configuration to ~/.guild/lsp/config.yaml
4. Test with a sample file

## See Also

- `config_example.yaml` - Example configuration for many languages
- Language Server Protocol: https://microsoft.github.io/language-server-protocol/
- Available servers: https://langserver.org/