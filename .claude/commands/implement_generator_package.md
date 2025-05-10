## Implement Generator Package

@context

This guide focuses on implementing the generator package for Guild's objective features, which will handle LLM-based content generation using prompts.

### 1. Check Existing Implementation

```bash
# Check for existing generator-related code
find . -type f -name "*.go" | xargs grep -l "generator\|Generator" | grep -v "_test.go" | sort

# Look for LLM client usage
find . -type f -name "*.go" | xargs grep -l "providers\|LLMClient\|Complete" | grep -v "_test.go" | sort

# Check for existing objective manipulation code
find . -type f -name "*.go" | xargs grep -l "objective\|Objective" | grep -v "_test.go" | sort
```

### 2. Implementation Structure

The generator package should follow this structure:

```
pkg/
└── generator/
    ├── interface.go                    # Generator interfaces
    └── objective/
        ├── generator.go                # Objective generator implementation
        ├── generator_test.go           # Tests for objective generator
        ├── testdata/                   # Test fixtures
        │   ├── sample_objective.md     # Sample objective for testing
        │   └── expected_output.md      # Expected output for testing
        └── mocks/                      # Mock implementations
            └── mock_client.go          # Mock LLM client
```

### 3. Implementation Steps

#### A. Define Generator Interfaces

In `pkg/generator/interface.go`:

1. Create generator interfaces:

   ```go
   // ObjectiveGenerator defines the interface for objective-related content generation
   type ObjectiveGenerator interface {
       // GenerateObjective creates a new objective from a description
       GenerateObjective(ctx context.Context, description string) (*objective.Objective, error)

       // GenerateAIDocs generates AI documentation based on an objective
       GenerateAIDocs(ctx context.Context, obj *objective.Objective, additionalContext string) (map[string]string, error)

       // GenerateSpecs generates technical specifications based on an objective
       GenerateSpecs(ctx context.Context, obj *objective.Objective, additionalContext string) (map[string]string, error)

       // SuggestImprovements suggests improvements to an objective
       SuggestImprovements(ctx context.Context, obj *objective.Objective) (string, error)
   }

   // Other generator interfaces as needed...
   ```

2. Define any additional interfaces for specialized generation tasks

#### B. Implement Objective Generator

In `pkg/generator/objective/generator.go`:

1. Create the generator struct:

   ```go
   // Generator handles LLM-based generation of objectives and related documents
   type Generator struct {
       client      providers.LLMClient
       promptMgr   *prompts.PromptManager
   }

   // NewGenerator creates a new objective generator
   func NewGenerator(client providers.LLMClient) (*Generator, error) {
       // Implementation...
   }
   ```

2. Implement interface methods:

   - `GenerateObjective()`
   - `GenerateAIDocs()`
   - `GenerateSpecs()`
   - `SuggestImprovements()`

3. Add helper methods:
   - Parse multiple documents from responses
   - Format objectives for prompts
   - Handle error cases

#### C. Implement Integration with Prompt System

Connect the generator to the prompt system:

1. Initialize the prompt manager:

   ```go
   // Inside NewGenerator()
   pm, err := prompts.NewPromptManager()
   if err != nil {
       return nil, fmt.Errorf("error creating prompt manager: %w", err)
   }
   ```

2. Use prompt manager for content generation:

   ```go
   // Inside a method like GenerateObjective()
   prompt, err := g.promptMgr.RenderPrompt("objective.creation", data)
   if err != nil {
       return nil, fmt.Errorf("error rendering prompt: %w", err)
   }
   ```

#### D. Implement LLM Client Integration

Connect to LLM providers:

1. Use the provider interface:

   ```go
   // Inside a method like GenerateObjective()
   response, err := g.client.Complete(ctx, &providers.CompletionRequest{
       Prompt:      prompt,
       MaxTokens:   2048,
       Temperature: 0.7,
   })
   if err != nil {
       return nil, fmt.Errorf("error calling LLM: %w", err)
   }
   ```

2. Handle responses and extract content:

   ```go
   // Parse response text into an objective
   obj, err := parseObjectiveFromText(response.Text)
   if err != nil {
       return nil, fmt.Errorf("error parsing objective: %w", err)
   }
   ```

#### E. Create Parsing Functions

Add document parsing functions:

1. Parse objectives from text:

   ```go
   func parseObjectiveFromText(text string) (*objective.Objective, error) {
       // Implementation...
   }
   ```

2. Parse multiple documents from responses:

   ```go
   func parseMultipleMarkdownDocs(text string) map[string]string {
       // Implementation...
   }
   ```

#### F. Implement Tests

Create comprehensive tests in `pkg/generator/objective/generator_test.go`:

1. Set up mock LLM client:

   ```go
   type mockLLMClient struct {
       responses map[string]string
   }

   func (m *mockLLMClient) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
       // Return pre-defined responses based on prompt content
   }

   // Implement other required methods...
   ```

2. Test all generator methods:

   - Test objective generation
   - Test AI docs generation
   - Test specs generation
   - Test suggestion generation

3. Test error handling:
   - LLM client errors
   - Parsing errors
   - Context cancellation

### 4. Key Implementation Details

#### A. Context Handling

Always respect context cancellation:

```go
// In every method
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
    // Continue processing
}
```

#### B. Error Wrapping

Use proper error wrapping:

```go
if err != nil {
    return nil, fmt.Errorf("error doing X: %w", err)
}
```

#### C. Temporary Files

When needed, use temporary files carefully:

```go
tempFile := filepath.Join(os.TempDir(), "temp_objective.md")
if err := os.WriteFile(tempFile, []byte(response.Text), 0644); err != nil {
    return nil, fmt.Errorf("error writing temp file: %w", err)
}
defer os.Remove(tempFile)
```

#### D. Document Parsing

Implement robust markdown parsing:

````go
func parseMultipleMarkdownDocs(text string) map[string]string {
    docs := make(map[string]string)

    // Split on markdown code blocks or document boundaries
    sections := strings.Split(text, "```markdown")

    for i, section := range sections {
        if i == 0 {
            continue // Skip preamble
        }

        // Extract content between markdown fences
        content := strings.Split(section, "```")[0]
        content = strings.TrimSpace(content)

        // Determine filename from content
        filename := extractFilenameFromContent(content)

        docs[filename] = content
    }

    return docs
}
````

### 5. Integration with Other Components

#### A. Objective Package

The generator should use the objective package for:

- Parsing objective files
- Creating new objectives
- Accessing objective content

#### B. Prompt System

The generator should use the prompt system for:

- Loading prompt templates
- Rendering prompts with data
- Managing prompt context

#### C. LLM Providers

The generator should use the provider system for:

- Sending requests to LLMs
- Handling responses
- Managing contexts and timeouts

### 6. Testing Strategies

#### A. Mock LLM Client

Create a mock LLM client that returns predefined responses:

```go
type mockLLMClient struct {
    responses map[string]providers.CompletionResponse
}

func (m *mockLLMClient) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
    // Return a response based on the prompt content
    for promptSubstr, resp := range m.responses {
        if strings.Contains(req.Prompt, promptSubstr) {
            return &resp, nil
        }
    }
    return nil, fmt.Errorf("no mock response for prompt")
}
```

#### B. Test Fixtures

Use test fixtures for:

- Sample objectives
- Expected AI docs output
- Expected specs output
- Expected suggestion output

#### C. Test All Paths

Test:

- Happy paths (successful generation)
- Error paths (LLM errors, parsing errors)
- Context cancellation
- Edge cases (empty input, large input)

### Resources

- [Go Context Package](https://pkg.go.dev/context)
- [Go Testing](https://pkg.go.dev/testing)
- [Go Temporary Files](https://pkg.go.dev/os#TempDir)
