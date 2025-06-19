# Post-Init Validation System

The post-init validation system ensures that `guild chat` will work immediately after running `guild init`. It performs comprehensive checks of all required components and provides clear, actionable feedback about any issues.

## Overview

The validation system checks:
1. **Project Structure** - Ensures .guild directory and subdirectories exist
2. **Campaign Configuration** - Validates campaign setup and detection
3. **Guild Configuration** - Checks guild definitions and agent assignments
4. **Agent Configuration** - Validates agent setup and manager availability
5. **Provider Configuration** - Checks AI provider credentials and connectivity
6. **Database Initialization** - Validates SQLite database and migrations
7. **Socket Registry** - Checks daemon socket registry setup
8. **Daemon Readiness** - Verifies daemon can be started

## Usage

### In Guild Init Command

The validation runs automatically at the end of `guild init`:

```bash
guild init
# ... setup process ...
🔍 Validating setup...

Validation Results:
------------------------------------------------------------
✓ Project Structure
  Checking .guild directory and subdirectories
  guild_dir: /path/to/project/.guild

✓ Campaign Configuration
  Checking campaign reference and configuration
  campaign: my-campaign
  guild_yaml: present

# ... more results ...

✓ All checks passed! Guild chat is ready to use.
```

### Skip Validation

For advanced users who want to skip validation:

```bash
guild init --skip-validation
```

### Programmatic Usage

```go
import (
    "context"
    "github.com/guild-ventures/guild-core/pkg/setup"
)

// Create validator
validator := setup.NewInitValidator("/path/to/project")

// Run validation
ctx := context.Background()
err := validator.Validate(ctx)

// Check results
if err != nil {
    // Validation failed
    validator.PrintResults()
    // Handle error...
}

// Access individual results
results := validator.GetResults()
for _, result := range results {
    if !result.Success {
        fmt.Printf("Failed: %s - %v\n", result.Name, result.Error)
    }
}
```

## Validation Results

Each validation check returns a result with:
- **Name** - The check name
- **Description** - What is being checked
- **Success** - Whether the check passed
- **Error** - Error details if failed
- **Warning** - Non-critical issues
- **Details** - Additional information

### Result Types

1. **Success (✓)** - Check passed completely
2. **Warning (⚠)** - Check passed but with issues that may affect functionality
3. **Failure (✗)** - Check failed and must be fixed

## Common Issues and Solutions

### Project Structure Issues

**Missing directories**
```
⚠ Project Structure
  Missing directories: agents, prompts, archives
```
**Solution**: Run `guild init` again or manually create missing directories

### Campaign Not Detected

**No campaign reference**
```
✗ Campaign Configuration
  Error: campaign not detected
```
**Solution**: Ensure you're in a guild project directory or run `guild init`

### No Agents Configured

**Empty agent list**
```
✗ Agent Configuration
  Error: no agents configured
```
**Solution**: Run setup wizard again with `guild init --force`

### Missing API Credentials

**Provider credentials not set**
```
✗ Provider Configuration
  Error: missing API credentials
  missing: OPENAI_API_KEY, ANTHROPIC_API_KEY
```
**Solution**: Set environment variables:
```bash
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
```

### Database Not Initialized

**Missing database file**
```
✗ Database Initialization
  Error: database file not found
```
**Solution**: Run `guild init` to create and initialize the database

## Context Support

The validation system fully supports context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := validator.Validate(ctx)
if err != nil {
    if gerror.GetCode(err) == gerror.ErrCodeCancelled {
        // Validation was cancelled
    }
}
```

## Custom Validation

You can run individual validation checks:

```go
validator := setup.NewInitValidator(projectPath)

// Run specific check
result := validator.ValidateProjectStructure(ctx)
if !result.Success {
    fmt.Printf("Project structure check failed: %v\n", result.Error)
}
```

## Integration with CI/CD

The validation system can be used in CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Validate Guild Setup
  run: |
    guild init --quick
    # Exit code will be non-zero if validation fails
```

## Best Practices

1. **Always run validation** after setup unless you have a specific reason to skip
2. **Pay attention to warnings** - they indicate potential issues
3. **Fix failures immediately** - guild chat won't work with validation failures
4. **Use context timeouts** in automated scenarios to prevent hanging
5. **Check validation in CI** to ensure deployments are properly configured

## Technical Details

The validator uses:
- **gerror** for consistent error handling with proper error codes
- **Context propagation** for cancellation support
- **Parallel validation** where possible for performance
- **Clear result formatting** with color-coded output
- **Actionable error messages** with suggested fixes

See `pkg/setup/init_validator.go` for implementation details.