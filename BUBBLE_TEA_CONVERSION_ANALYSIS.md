# Bubble Tea Conversion Analysis for Guild CLI

## Summary

This document identifies all Guild CLI commands that need to be converted from old-style terminal interfaces (fmt.Printf, readInput, bufio.NewReader) to modern Bubble Tea interfaces.

## Commands Using Old-Style Interfaces

### 1. **init.go** (HIGH PRIORITY)
- **Status**: Uses extensive fmt.Printf and readInput throughout
- **Interactive Elements**:
  - Campaign name prompt (line 322)
  - Project name prompt (line 337)
  - Demo commission selection (lines 562-597)
  - Yes/No prompts via askYesNo() function
  - readInput() helper function using bufio.NewReader (lines 632-645)
- **Complexity**: HIGH - Core initialization flow with multiple interactive prompts
- **Note**: Already has some Bubble Tea integration for setup wizard (line 196-202)

### 2. **setup.go** (PARTIALLY CONVERTED)
- **Status**: Already uses Bubble Tea for main wizard (line 128-135)
- **Interactive Elements**:
  - Main wizard already converted to Bubble Tea
  - Still uses fmt.Printf for status messages
- **Complexity**: LOW - Mostly converted already

### 3. **config.go** (LOW PRIORITY)
- **Status**: Uses fmt.Printf extensively for output only
- **Interactive Elements**:
  - No interactive prompts detected
  - Only uses fmt.Printf for displaying configuration
- **Complexity**: LOW - Output only, no user interaction

### 4. **agent.go** (LOW PRIORITY)
- **Status**: Uses fmt.Printf for output
- **Interactive Elements**:
  - No interactive prompts detected
  - Only status display and listings
- **Complexity**: LOW - Output only

### 5. **commission.go** (MEDIUM PRIORITY)
- **Status**: Uses fmt.Printf for output
- **Interactive Elements**:
  - No direct user input detected
  - Command-line flag based interaction
- **Complexity**: MEDIUM - Could benefit from interactive task selection

### 6. **campaign.go** (MEDIUM PRIORITY)
- **Status**: Uses fmt.Printf for output
- **Interactive Elements**:
  - No direct user input detected
  - Uses command-line flags
- **Complexity**: MEDIUM - Could add interactive campaign management

### 7. **chat.go** (ALREADY CONVERTED)
- **Status**: Fully converted to Bubble Tea
- **Interactive Elements**:
  - Complete TUI implementation (1,951 lines)
  - Only one fmt.Printf on line 115 (non-interactive)
- **Complexity**: COMPLETE - No conversion needed

### 8. **corpus.go** (LOW PRIORITY)
- **Status**: Has corpus UI already (line 32)
- **Interactive Elements**:
  - Already has UI mode
  - Command-line subcommands for non-UI operations
- **Complexity**: LOW - Already has UI support

### 9. **prompt.go** (LOW PRIORITY)
- **Status**: Uses fmt.Printf for output
- **Interactive Elements**:
  - No interactive prompts detected
  - gRPC client commands
- **Complexity**: LOW - Output only

### 10. **migrate.go** (LOW PRIORITY)
- **Status**: Uses fmt.Printf for output
- **Interactive Elements**:
  - No interactive prompts detected
  - Progress display only
- **Complexity**: LOW - Output only

### 11. **serve.go** (LOW PRIORITY)
- **Status**: Uses fmt.Printf for daemon output
- **Interactive Elements**:
  - No interactive prompts
  - Daemon/server command
- **Complexity**: LOW - Not suitable for TUI

### 12. **init_legacy.go** (DEPRECATED)
- **Status**: Legacy initialization command
- **Interactive Elements**:
  - Uses fmt.Printf
- **Complexity**: SKIP - Legacy code

## Priority Order for Conversion

1. **init.go** - Critical user-facing initialization with multiple prompts
2. **commission.go** - Could benefit from interactive task management
3. **campaign.go** - Could add interactive campaign selection/management
4. All other commands - Low priority, mostly output-only

## Recommendations

### Immediate Action Items:
1. Convert `init.go` to use Bubble Tea for all interactive prompts
2. Create reusable Bubble Tea components for:
   - Text input prompts
   - Yes/No confirmations
   - Selection lists (for demo types)
   - Progress indicators

### Design Patterns to Follow:
1. Use the existing chat.go implementation as a reference
2. Create a shared `internal/ui/components` package for reusable TUI components
3. Maintain backward compatibility with command-line flags
4. Provide `--non-interactive` flag for CI/CD environments

### Components Needed:
- `TextInputModel` - For simple text prompts
- `ConfirmModel` - For yes/no questions
- `SelectModel` - For choosing from options
- `ProgressModel` - For long-running operations

## Technical Considerations

1. **Context Support**: Ensure all Bubble Tea models properly handle context cancellation
2. **Error Handling**: Use gerror consistently in TUI components
3. **Testing**: Create test helpers for TUI components
4. **Accessibility**: Ensure keyboard navigation works properly
5. **Theme**: Consider creating a consistent Guild theme/style