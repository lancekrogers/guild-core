# Multi-Edit Tool Implementation

This document describes the atomic MultiEdit tool implementation for the Guild Framework.

## Overview

The MultiEdit tool (`multi_edit`) provides atomic multi-edit operations on a single file, equivalent to Claude Code's MultiEdit tool. It applies multiple find-and-replace operations in sequence with atomic success/failure semantics.

## Files Created

- `multi_edit_tool.go` - Main tool implementation
- `multi_edit_tool_test.go` - Comprehensive test suite
- `registry.go` - Tool registration functions
- `registry_test.go` - Registry integration tests

## Features

### Core Functionality

- **Atomic Operations**: Either all edits succeed or none are applied
- **Sequential Processing**: Edits are applied in the order specified
- **Replace Options**: Support for replace-first or replace-all modes
- **Validation**: Optional upfront validation of all edits
- **Backup Support**: Automatic backup creation before applying changes
- **Dry Run Mode**: Preview changes without applying them

### Input Parameters

```json
{
  "file_path": "path/to/file.ext",
  "edits": [
    {
      "old_string": "text to find",
      "new_string": "replacement text",
      "replace_all": false
    }
  ],
  "backup": true,
  "dry_run": false,
  "validate": true
}
```

### Output Information

- Applied/failed edit count and details
- Character change statistics
- Processing time metrics
- Backup file location
- Preview of changes (in dry run mode)
- Validation errors and warnings

## Safety Features

1. **Atomic File Writing**: Uses temporary files and atomic rename operations
2. **Input Validation**: Comprehensive validation of all parameters
3. **Error Handling**: Detailed error reporting with Guild error codes
4. **Backup Creation**: Optional backup before any modifications
5. **Dry Run Testing**: Preview mode to verify changes before applying

## Integration

The tool is registered with the Guild Framework through:

- `RegisterEditTools()` function in `registry.go`
- Integration with the main tool registry in `pkg/registry/code_tools.go`
- Cost-aware registration with zero cost magnitude (local file operations)

## Testing

Comprehensive test suite includes:

- Basic functionality tests (single/multiple edits)
- Replace mode tests (first occurrence vs all occurrences)
- Safety feature tests (dry run, backup, validation)
- Error condition tests (invalid input, file not found, etc.)
- Medieval-themed test naming following Guild conventions
- Benchmark tests for performance verification

## Usage Examples

### Simple Single Edit

```json
{
  "file_path": "main.go",
  "edits": [
    {"old_string": "oldFunc", "new_string": "newFunc", "replace_all": true}
  ]
}
```

### Multiple Complex Edits with Safety

```json
{
  "file_path": "config.json",
  "edits": [
    {"old_string": "\"debug\": false", "new_string": "\"debug\": true"},
    {"old_string": "\"port\": 8080", "new_string": "\"port\": 3000"}
  ],
  "backup": true,
  "validate": true
}
```

### Dry Run Preview

```json
{
  "file_path": "script.py",
  "edits": [
    {"old_string": "import old_module", "new_string": "import new_module"},
    {"old_string": "old_function()", "new_string": "new_function()"}
  ],
  "dry_run": true
}
```

## Architecture Notes

- Follows Guild Framework tool interface patterns
- Uses Guild error handling with proper error codes
- Implements medieval naming conventions for tests
- Provides detailed metadata and extra data in results
- Supports both the tools registry and pkg/tools registry interfaces

## Performance

- Optimized for sequential edit processing
- Minimal memory overhead with efficient string operations
- Atomic file operations prevent corruption
- Sub-millisecond processing for typical edit operations
