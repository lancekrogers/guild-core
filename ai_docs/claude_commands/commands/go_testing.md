## Go Testing Practices for Guild

When writing tests for Go code in Guild:

1. Use the standard `testing` package
2. Place tests in the same package with `_test.go` suffix
3. Use table-driven tests for multiple test cases
4. Use `t.Parallel()` for tests that can run concurrently
5. Use `t.Helper()` for helper functions
6. Use subtests with `t.Run()` for organizing test cases

### Testing Tools

1. Use `go test -race` to detect race conditions
2. Use `go test -cover` to measure test coverage
3. Use `go test -v` for verbose output

### Mock Generation

For interface mocks, we use manual mocks or the `mockery` tool:

```bash
go install github.com/vektra/mockery/v2@latest
mockery --name=Interface --filename=mock_interface.go --output=mocks
```

Please ensure all interfaces have corresponding mocks for testing.
