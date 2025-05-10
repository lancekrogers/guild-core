## Implement Prompt Management System

@context

@context

This guide focuses on implementing the internal prompt management system for Guild's objective features.

### 1. Check Existing Implementation

```bash
# Check for existing prompt-related code
find . -type f -name "*.go" | xargs grep -l "prompt\|template" | grep -v "_test.go" | sort

# Look for existing embed usage
find . -type f -name "*.go" | xargs grep -l "embed\|//go:embed" | grep -v "_test.go" | sort

# Check for existing markdown-related code
find . -type f -name "*.go" | xargs grep -l "markdown\|md" | grep -v "_test.go" | sort
```

### 2. Implementation Structure

The prompt system should follow this structure:

```
internal/
└── prompts/
    ├── manager.go                # Central prompt manager
    └── objective/
        ├── loader.go             # Objective prompt loader
        ├── testdata/             # Test fixtures
        │   └── test_prompts.md   # Test prompts
        └── markdown/             # Actual prompt files
            ├── creation.md       # Objective creation
            ├── ai_docs_gen.md    # AI docs generation
            ├── specs_gen.md      # Specs generation
            ├── refinement.md     # Objective refinement
            └── suggestion.md     # Improvement suggestions
```

### 3. Implementation Steps

#### A. Create Prompt Loader

In `internal/prompts/objective/loader.go`:

1. Set up Go embed to include markdown files:

   ```go
   package objective

   import (
       "embed"
       "io/fs"
       // Other imports...
   )

   //go:embed markdown/*.md
   var promptFS embed.FS
   ```

2. Implement prompt loading functions:

   - `LoadPrompts()` to load all prompts
   - `GetPrompt()` to get a specific prompt
   - `ListPromptNames()` to list available prompts

3. Add helper functions for template parsing and management

#### B. Implement Central Prompt Manager

In `internal/prompts/manager.go`:

1. Create the `PromptManager` struct:

   - Store templates in a map
   - Add mutex for thread safety
   - Include initialization logic

2. Implement core functions:

   - `NewPromptManager()` to create a new manager
   - `RenderPrompt()` to render a prompt with data
   - `HasPrompt()` to check if a prompt exists
   - `ListPrompts()` to list all available prompts
   - `RefreshPrompts()` to reload prompts from disk

3. Add error handling and logging

#### C. Create Prompt Files

In `internal/prompts/objective/markdown/`:

1. Create each prompt file as a markdown document
2. Ensure they follow a consistent structure:

   - Title and purpose
   - Input variables (using Go template syntax)
   - Format description
   - Example inputs/outputs
   - Guidelines for the LLM

3. Include placeholder markers using Go template syntax:
   ```markdown
   {{.VariableName}}
   ```

#### D. Create Tests

In `internal/prompts/objective/loader_test.go` and `internal/prompts/manager_test.go`:

1. Test loading functionality:

   - Test loading all prompts
   - Test loading specific prompts
   - Test error handling for missing prompts

2. Test template rendering:

   - Test variable substitution
   - Test conditional sections
   - Test error handling for invalid data

3. Use test fixtures in `testdata/` folder

### 4. Integration with Generators

Connect the prompt system to generators:

1. In `pkg/generator/objective/generator.go`:

   - Create a generator that uses the prompt manager
   - Render prompts with appropriate data
   - Pass rendered prompts to LLM clients
   - Process LLM responses

2. Add helper functions for common prompt operations:
   - Format objectives as markdown
   - Prepare data for templates
   - Extract generated content from responses

### 5. Key Implementation Details

#### A. Template Variables

Common template variables to support:

- `{{.Description}}` - User's initial description
- `{{.Objective}}` - The current objective content
- `{{.CurrentObjective}}` - Original objective content for refinement
- `{{.UserContext}}` - Additional context from user
- `{{.DocumentContext}}` - Content from referenced documents
- `{{.AdditionalContext}}` - Any other provided context

#### B. Embedding Files

Use proper Go embed directives:

```go
//go:embed markdown/*.md
var promptFS embed.FS
```

Ensure proper error handling when reading embedded files.

#### C. Template Functions

Consider adding custom template functions:

```go
funcMap := template.FuncMap{
    "trim":      strings.TrimSpace,
    "lowercase": strings.ToLower,
    "join":      strings.Join,
    // Other useful functions...
}

// Use when creating templates
template.New(name).Funcs(funcMap).Parse(content)
```

#### D. Thread Safety

Ensure the prompt manager is thread-safe:

```go
type PromptManager struct {
    templates map[string]*template.Template
    mu        sync.RWMutex
}

// Use mutex in methods
func (pm *PromptManager) RenderPrompt(name string, data interface{}) (string, error) {
    pm.mu.RLock()
    tmpl, exists := pm.templates[name]
    pm.mu.RUnlock()

    // Rest of the method...
}
```

### 6. Testing Strategies

1. **Unit Tests**:

   - Test each function in isolation
   - Use test fixtures for prompt content
   - Test error handling paths

2. **Mock Templates**:

   - Create simple test templates
   - Test variable substitution
   - Test conditional rendering

3. **Integration Tests**:
   - Test with a real LLM client
   - Verify end-to-end prompt rendering and processing
   - Test all prompt types

### Resources

- [Go Embed Documentation](https://pkg.go.dev/embed)
- [Go Template Package](https://pkg.go.dev/text/template)
- [Go Testing](https://pkg.go.dev/testing)
