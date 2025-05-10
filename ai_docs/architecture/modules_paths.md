## Module Paths

@context

This command provides essential information about correct module paths and import conventions for the Guild project.

### Module Path Configuration

The Guild project uses the following module path:

```go
module github.com/blockhead-consulting/guild
```

### Important Path Guidelines

1. **Use lowercase** for the module path, following Go conventions:

   - CORRECT: `github.com/blockhead-consulting/guild`
   - INCORRECT: `github.com/Blockhead-Consulting/Guild`

2. **Import packages** using the full module path:

   ```go
   import (
       "github.com/blockhead-consulting/guild/pkg/objective"
       "github.com/blockhead-consulting/guild/pkg/generator"
   )
   ```

3. **Avoid relative imports** between packages:
   - CORRECT: `import "github.com/blockhead-consulting/guild/pkg/objective"`
   - INCORRECT: `import "../objective"`

### Checking for Old Module Paths

Run these commands to ensure all imports use the correct path:

```bash
# Check for any remaining references to old module path
grep -r "github.com/lancekrogers/guild" --include="*.go" .

# Check for incorrect casing
grep -r "github.com/Blockhead-Consulting" --include="*.go" .

# Verify correct path usage
grep -r "github.com/blockhead-consulting/guild" --include="*.go" .
```

### Fixing Module Paths

If you find incorrect module paths, update them:

```bash
# Update old personal module path to organization path
find . -type f -name "*.go" -exec sed -i 's|github.com/lancekrogers/guild|github.com/blockhead-consulting/guild|g' {} \;

# Fix casing if needed
find . -type f -name "*.go" -exec sed -i 's|github.com/Blockhead-Consulting/Guild|github.com/blockhead-consulting/guild|g' {} \;
find . -type f -name "*.go" -exec sed -i 's|github.com/Blockhead-Consulting/guild|github.com/blockhead-consulting/guild|g' {} \;
```

### Standard Package Structure

The Guild project follows this package structure:

```
github.com/blockhead-consulting/guild/
├── cmd/              # Command-line applications
├── pkg/              # Public libraries
├── internal/         # Private application code
└── tools/            # Tool definitions and implementations
```

### Module Dependency Management

When adding new dependencies:

1. Use explicit versions in go.mod

   ```go
   require (
       github.com/charmbracelet/bubbletea v0.30.0
       github.com/sashabaranov/go-openai v1.38.2
   )
   ```

2. Run go mod tidy after adding dependencies

   ```bash
   go mod tidy
   ```

3. Verify module integrity
   ```bash
   go mod verify
   ```

When implementing new code or moving existing code, always ensure all imports use the correct module path to avoid confusion and build problems.
