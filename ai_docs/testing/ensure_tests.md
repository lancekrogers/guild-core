## Testing Requirements

All code written for Guild must include comprehensive unit tests following these principles:

1. **Test Coverage**: Aim for at least 80% test coverage for all packages
2. **Table-Driven Tests**: Use table-driven tests for functions with multiple input/output cases
3. **Mock Dependencies**: Use interface mocks for external dependencies
4. **Context Testing**: Test behavior with canceled contexts and timeouts
5. **Error Cases**: Include explicit tests for error conditions
6. **Parallel Tests**: Write tests that can run in parallel when possible

## Testing Patterns

### Basic Test Structure

```go
func TestFunction(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"

    // Act
    result, err := Function(input)

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result != expected {
        t.Errorf("expected %q, got %q", expected, result)
    }
}
```

### Table-Driven Tests

```go
func TestFunction(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "basic case",
            input:    "test",
            expected: "result",
            wantErr:  false,
        },
        {
            name:     "error case",
            input:    "invalid",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := Function(tc.input)

            if tc.wantErr {
                if err == nil {
                    t.Error("expected error but got none")
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if result != tc.expected {
                t.Errorf("expected %q, got %q", tc.expected, result)
            }
        })
    }
}
```

### Mocking Dependencies

```go
// Create mock implementations of interfaces
type MockStore struct {
    SaveFunc func(ctx context.Context, data interface{}) error
    GetFunc  func(ctx context.Context, id string) (interface{}, error)
}

func (m *MockStore) Save(ctx context.Context, data interface{}) error {
    return m.SaveFunc(ctx, data)
}

func (m *MockStore) Get(ctx context.Context, id string) (interface{}, error) {
    return m.GetFunc(ctx, id)
}

// Use in tests
func TestService(t *testing.T) {
    mockStore := &MockStore{
        SaveFunc: func(ctx context.Context, data interface{}) error {
            return nil
        },
        GetFunc: func(ctx context.Context, id string) (interface{}, error) {
            return "test data", nil
        },
    }

    service := NewService(mockStore)
    result, err := service.DoSomething(context.Background(), "test")

    // Assert results
}
```

Remember to add tests for every file you create. Every function exposed in a package's public API must have test coverage.
