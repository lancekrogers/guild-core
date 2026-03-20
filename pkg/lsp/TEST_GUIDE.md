# LSP Package Testing Guide

## Overview

The LSP package has been refactored to support testing without requiring real LSP servers (like gopls). This prevents tests from hanging or failing due to external dependencies.

## Test Types

### 1. Unit Tests (Recommended for CI/CD)

Unit tests use mock implementations and run quickly without external dependencies:

```bash
# Run unit tests only
cd guild-core
go test ./pkg/lsp/manager_unit_test.go

# Or using make
make test PKG=./pkg/lsp
```

**Features of Unit Tests:**

- ✅ No external LSP servers required
- ✅ Fast execution (< 1 second)
- ✅ Predictable results
- ✅ Test error conditions easily
- ✅ CI/CD friendly

### 2. Integration Tests (Optional)

Integration tests require real LSP servers and are tagged to run only when explicitly requested:

```bash
# Run integration tests (requires gopls)
cd guild-core
go test -tags="integration,lsp" ./pkg/lsp/...

# Skip if gopls not installed
# Tests will automatically skip with message: "gopls not found in PATH"
```

**Features of Integration Tests:**

- ⚠️ Requires gopls installed: `go install golang.org/x/tools/gopls@latest`
- ⚠️ Slower execution (10-30 seconds)
- ⚠️ May fail due to external factors
- ✅ Tests real LSP server integration
- ✅ Has 30-second timeout to prevent hanging

## Mock Architecture

### Interfaces Created

1. **ClientInterface** (`interfaces.go`)
   - Defines contract for LSP client operations
   - Allows swapping real client with mock

2. **ServerManagerInterface** (`interfaces.go`)
   - Manages language server lifecycle
   - Can be mocked for testing

3. **ProcessLauncherInterface** (`interfaces.go`)
   - Abstracts process launching
   - Prevents spawning real processes in tests

### Mock Implementations

1. **MockLSPClient** (`mocks/mock_lsp_client.go`)
   - Simulates LSP server responses
   - Tracks method calls for assertions
   - Allows error injection for testing

2. **MockProcessLauncher** (`mocks/mock_process_launcher.go`)
   - Returns mock clients instead of launching processes
   - Tracks launch attempts

## Usage Examples

### Example: Testing with Mocks

```go
func TestMyFeature(t *testing.T) {
    // Create mock client
    mockClient := mocks.NewMockLSPClient()
    
    // Configure expected responses
    mockClient.CompletionResponse = &protocol.CompletionList{
        Items: []protocol.CompletionItem{
            {Label: "TestMethod"},
        },
    }
    
    // Test your feature
    result, err := myFeature(mockClient)
    
    // Verify behavior
    assert.Equal(t, 1, mockClient.CompletionCalls)
}
```

### Example: Testing Error Conditions

```go
func TestErrorHandling(t *testing.T) {
    mockClient := mocks.NewMockLSPClient()
    
    // Inject error
    mockClient.CompletionError = errors.New("LSP server crashed")
    
    // Test error handling
    _, err := myFeature(mockClient)
    assert.Error(t, err)
}
```

## CI/CD Configuration

For CI/CD pipelines, exclude integration tests by default:

```yaml
# GitHub Actions example
- name: Run Tests
  run: |
    cd guild-core
    # Run all tests except those requiring integration tag
    go test ./...
    
# Optional: Run integration tests in separate job
- name: Run Integration Tests
  run: |
    go install golang.org/x/tools/gopls@latest
    cd guild-core
    go test -tags="integration,lsp" -timeout 5m ./pkg/lsp/...
```

## Troubleshooting

### Tests Still Hanging?

1. Check you're not running integration tests accidentally:

   ```bash
   # This will run integration tests too!
   go test -tags=integration ./...
   
   # Run only unit tests
   go test ./pkg/lsp/manager_unit_test.go
   ```

2. Ensure timeouts are set:
   - Unit tests: Should complete in < 1 second
   - Integration tests: 30-second timeout per test

3. Check for gopls processes:

   ```bash
   ps aux | grep gopls
   # Kill any hanging gopls processes
   pkill gopls
   ```

### Mock Not Working?

1. Ensure interfaces are used instead of concrete types
2. Check mock is properly initialized
3. Verify mock responses are configured before use

## Future Improvements

1. **Interface Extraction**: Refactor Manager to use interfaces for better testability
2. **Mock Builder**: Create fluent API for building mock responses
3. **Test Fixtures**: Add common test scenarios as reusable fixtures
4. **Performance Benchmarks**: Add benchmarks using mocks for consistent results
