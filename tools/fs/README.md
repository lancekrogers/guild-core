# Filesystem Tools

This package provides filesystem tools for the Guild Framework, including file operations and pattern matching capabilities.

## Tools

### Glob Tool

The glob tool provides fast file pattern matching with support for recursive directory traversal using `**` patterns, similar to Claude Code's Glob tool.

#### Features

- **Recursive Pattern Matching**: Supports `**` patterns for deep directory traversal
- **Exclusion Patterns**: Filter out unwanted files and directories
- **Modification Time Sorting**: Results sorted by modification time (newest first)
- **Security**: Path sanitization to prevent directory traversal attacks
- **Directory Filtering**: Excludes directories by default (configurable)

#### Usage

```json
{
  "pattern": "**/*.go",
  "path": "./src",
  "exclude": ["node_modules/**", ".git/**"],
  "include_dirs": false
}
```

#### Parameters

- `pattern` (required): Glob pattern to match files (e.g., "**/*.go", "src/**/*.js", "*.md")
- `path` (optional): Directory to search in (defaults to current directory)
- `exclude` (optional): Array of exclusion patterns
- `include_dirs` (optional): Include directories in results (default: false)

#### Examples

1. **Find all Go files recursively**:
   ```json
   {"pattern": "**/*.go"}
   ```

2. **Find JavaScript files in src directory**:
   ```json
   {"pattern": "src/**/*.js", "path": "./my-project"}
   ```

3. **Find TypeScript files excluding node_modules**:
   ```json
   {"pattern": "**/*.{ts,tsx}", "exclude": ["node_modules/**", ".git/**"]}
   ```

4. **Find test files**:
   ```json
   {"pattern": "**/*_test.go", "exclude": ["**/mock_*"]}
   ```

#### Output

Returns a JSON object with:

```json
{
  "files": [
    {
      "path": "/absolute/path/to/file.go",
      "relative_path": "pkg/example/file.go",
      "size": 1024,
      "mod_time": "2023-12-01T10:00:00Z",
      "is_dir": false
    }
  ],
  "count": 1,
  "pattern": "**/*.go",
  "search_dir": "/working/directory"
}
```

### File Tool

The file tool provides basic file system operations including read, write, list, exists, and delete operations.

## Registration

Filesystem tools are automatically registered when initializing the Guild Framework registry:

```go
import "github.com/guild-ventures/guild-core/pkg/registry"

// Tools are registered automatically during registry initialization
err := registry.RegisterFSTools(toolRegistry)
```

## Security

All filesystem tools implement security measures:

- Path sanitization to prevent directory traversal
- Operations restricted to base directory
- Safe handling of symbolic links
- Input validation and error handling

## Performance

The glob tool is optimized for performance:

- Uses efficient `filepath.Walk` for recursive traversal
- Minimal memory allocation for large directory trees
- Early termination for non-matching paths
- Sorting optimization for modification times