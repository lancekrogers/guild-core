## Go Best Practices

@context

This command outlines Go best practices to follow when implementing Guild components. Adhering to these practices ensures code quality and maintainability.

### Code Organization

1. **Package Structure**

   - Keep packages focused on a single responsibility
   - Use meaningful package names that describe purpose, not type
   - Aim for packages with 10-30 files maximum

2. **File Organization**

   - Group related functionality in the same file
   - Keep files under 500 lines when possible
   - Order declarations: constants, variables, types, functions

3. **Imports**
   - Group imports in standard library, external, and internal blocks
   - Use aliasing only when necessary to avoid conflicts
   ```go
   import (
       // Standard library
       "context"
       "fmt"
       "time"

       // External dependencies
       "github.com/charmbracelet/bubbles/list"
       "github.com/charmbracelet/bubbletea"

       // Internal packages
       "github.com/blockhead-consulting/guild/pkg/objective"
   )
   ```

### Error Handling

1. **Error Creation**

   - Use `fmt.Errorf()` with `%w` for wrapping errors
   - Create custom error types for specific error conditions

   ```go
   if err != nil {
       return fmt.Errorf("parsing objective %s: %w", path, err)
   }
   ```

2. **Error Checking**

   - Always check errors
   - Don't use `_` to ignore errors without justification
   - Use `errors.Is()` and `errors.As()` for error inspection

3. **Error Propagation**
   - Add context when returning errors up the call stack
   - Include relevant values in error messages
   - Consider defining sentinel errors for important conditions
   ```go
   var ErrObjectiveNotFound = errors.New("objective not found")
   ```

### Concurrency

1. **Context Usage**

   - Pass `context.Context` as the first parameter to functions
   - Check for context cancellation in long-running operations
   - Create context timeouts for external operations

   ```go
   func (g *Generator) GenerateObjective(ctx context.Context, description string) (*objective.Objective, error) {
       select {
       case <-ctx.Done():
           return nil, ctx.Err()
       default:
           // Continue operation
       }
   }
   ```

2. **Goroutine Management**

   - Always provide a way to stop goroutines
   - Use `sync.WaitGroup` to wait for goroutines to complete
   - Pass data through channels rather than shared memory

   ```go
   var wg sync.WaitGroup
   wg.Add(1)
   go func() {
       defer wg.Done()
       // Do work
   }()
   wg.Wait()
   ```

3. **Channel Patterns**
   - Close channels from the sender, not the receiver
   - Use select with default to make non-blocking operations
   - Consider buffered channels to decouple producers and consumers

### Testing

1. **Test Organization**

   - Name tests as `TestFunctionName` or `TestFunctionName_Scenario`
   - Structure tests with setup, execution, and assertion phases
   - Group related test assertions together

   ```go
   func TestParseObjective_ValidInput(t *testing.T) {
       // Setup
       input := readTestFile(t, "testdata/valid_objective.md")

       // Execute
       obj, err := ParseObjective(input)

       // Assert
       require.NoError(t, err)
       assert.Equal(t, "Expected Title", obj.Title)
       assert.Equal(t, "Expected Goal", obj.Goal)
   }
   ```

2. **Mocks and Interfaces**

   - Design interfaces for testability
   - Create mock implementations for testing
   - Use table-driven tests for multiple scenarios

   ```go
   type mockLLMClient struct {
       responses map[string]string
   }

   func (m *mockLLMClient) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
       return &providers.CompletionResponse{
           Text: m.responses[req.Prompt],
       }, nil
   }
   ```

3. **Test Helpers**
   - Create helper functions for common test operations
   - Use `t.Helper()` to mark helper functions
   - Include setup and teardown helpers when needed
   ```go
   func loadTestObjective(t *testing.T, path string) *objective.Objective {
       t.Helper()
       // Implementation
   }
   ```

### Documentation

1. **Package Documentation**

   - Include a package comment at the top of a doc.go file
   - Explain the package's purpose and usage

   ```go
   // Package objective provides functionality for parsing and
   // managing markdown objectives within the Guild framework.
   //
   // Objectives are structured markdown documents that define
   // project goals, requirements, and related metadata.
   package objective
   ```

2. **Function Documentation**

   - Document all exported functions, types, and methods
   - Include example usage for complex functions
   - Document parameters and return values

   ```go
   // ParseObjective parses a markdown file into an Objective structure.
   //
   // The function expects a file that follows the Guild objective format
   // with Goal, Context, Requirements, Tags, and Related sections.
   //
   // If the file cannot be parsed, an error is returned with details.
   func ParseObjective(path string) (*Objective, error) {
   ```

3. **Example Code**
   - Include runnable examples in doc tests
   - Demonstrate typical usage patterns
   ```go
   func ExampleParseObjective() {
       obj, err := objective.ParseObjective("sample.md")
       if err != nil {
           fmt.Println("Error:", err)
           return
       }
       fmt.Println("Title:", obj.Title)
       // Output: Title: Sample Objective
   }
   ```

### Performance

1. **Resource Management**

   - Use `defer` for cleanup operations
   - Close resources in reverse order of acquisition
   - Release locks and connections explicitly

   ```go
   f, err := os.Open(path)
   if err != nil {
       return nil, err
   }
   defer f.Close()
   ```

2. **Memory Efficiency**

   - Avoid unnecessary allocations, especially in hot paths
   - Reuse buffers for repeated operations
   - Consider using sync.Pool for frequently allocated objects

   ```go
   var bufPool = sync.Pool{
       New: func() interface{} {
           return new(bytes.Buffer)
       },
   }
   ```

3. **Optimization Guidelines**
   - Profile before optimizing
   - Focus optimization on hot paths
   - Document performance characteristics for critical functions
   - Prefer readability over premature optimization

### Dependency Management

1. **Version Specification**
   - Specify explicit versions in go.mod
   - Avoid using incompatible replacement directives
   - Run `go mod tidy` before committing changes
2. **Vendoring (Optional)**

   - Consider vendoring dependencies for deployment stability
   - Include `vendor/` in version control if used
   - Update vendored dependencies with `go mod vendor`

3. **Dependency Guidelines**
   - Minimize external dependencies
   - Prefer standard library solutions when available
   - Evaluate dependency maintenance status before adding

When implementing Guild components, apply these best practices consistently to ensure high-quality, maintainable code.
