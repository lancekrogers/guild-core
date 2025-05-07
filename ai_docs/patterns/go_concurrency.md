# Go Concurrency Patterns

Guild uses Go's native concurrency features for agent parallelism and communication.

## Design Guidelines

- Use `context.Context` for cancellations and deadlines.
- Use channels for safe, typed communication.
- Avoid global state with `sync.Mutex` where needed.

## Example

```go
resultCh := make(chan Result)
go func() {
  res, err := agent.Execute(ctx, task)
  resultCh <- Result{Data: res, Err: err}
}()
