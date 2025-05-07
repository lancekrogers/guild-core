# Interface-First Design in Go

Interfaces define behavior expectations across agents and tools.

## Benefits

- Enables flexible testing via mocks.
- Decouples implementation from architecture.
- Required for clean multi-agent coordination.

## Example

```go
type Tool interface {
  Invoke(ctx context.Context, input Input) (Output, error)
}
```
