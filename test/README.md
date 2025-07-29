# Guild Framework Test Guidelines

This document defines the testing standards and conventions for the Guild Framework. These rules ensure consistent, reliable, and maintainable tests across the codebase.

## Database & Migrations

### 1. Dirty Migrations
**Rule**: Fail fast on dirty migrations.
- If a migration leaves the schema "dirty" (checksum mismatch, partial apply), the test must call `t.Fatalf`
- The test harness auto-drops and recreates the DB for the next case
- No silent cleanup; failures should stay loud

### 2. Per-test DB Isolation
- **Unit tests**: Use fresh in-memory SQLite (`:memory:` DSN) via `sqlmock` or `github.com/glebarez/sqlite`
- **Integration tests**: Each test spins up its own temp file under `$TMPDIR/guild-test-*/db.sqlite`, removed in `t.Cleanup`
- Never share a handle across `t.Run` sub-tests unless explicitly verifying concurrency

### 3. Migration Version to Start From
- Default is version 0 (empty DB) then run all migrations
- This guarantees forward compatibility
- If a spec requires seeding from a mid-stream version, call `migrations.To(version)` explicitly and document why

## Port Allocation

### 4. Port Range
- Let the OS pick by binding to port 0 whenever possible
- When a fixed port is required (e.g., `grpc-health-probe`), choose from **56000-56999**
- Reserve the sub-range in `docs/ports.md`

### 5. "Address Already in Use"
- Retry up to 3 times with exponential back-off (100ms → 400ms → 1.6s)
- After that, call `t.Fatalf`
- Flaky port collisions shouldn't mask real bugs

## Mock Implementations

### 6. Required vs. Optional Methods
- **Required**: Any method the SUT calls in normal operation
- Leave optional methods unimplemented and have them `panic("unexpected call")`
- This surfaces accidental couplings early

### 7. Streaming Mocks
- Emit chunks in realistic sizes (32–128 KiB for binary, line-by-line for text)
- Support context cancellation: stop sending when `<-ctx.Done()`
- Allow an optional `Delay` field (default 0) so tests can inject latency without sprinkling `time.Sleep`

## Timing & Synchronization

### 8. Standard Timeouts
Overridable via environment variables:

| Purpose | Default | Env Override |
|---------|---------|--------------|
| Service/container boot | 2s | `GUILD_BOOT_TIMEOUT` |
| Single agent task | 5s | `GUILD_TASK_TIMEOUT` |
| Event-bus delivery | 1s | `GUILD_EVENT_TIMEOUT` |

### 9. Delays vs. Conditions
- Never hard-sleep
- Wait on a condition (e.g., `wait.For(func() bool { return status.Ready() })`) with the timeouts above
- Keeps CI fast and deterministic

## Error Handling

### 10. Check Specific gerror Codes
- Assert the code (e.g., `codes.InvalidArgument`)
- Ignore the text unless the message is user-facing

### 11. Are Messages Part of the Contract?
- Only for public/user APIs
- Internal errors may change wording; tests should not pin them

## Environment Setup

### 12. Required Environment Variables
```bash
GUILD_TEST_MODE=1
GUILD_CONFIG_DIR=$TMPDIR/guild-test-*/config
GUILD_DATA_DIR=$TMPDIR/guild-test-*/data
```

### 13. SQLite Availability
- Yes—CI images include SQLite with CGO enabled
- If `sqlite3.Open` fails, the test skips (`t.Skip`), not fails

### 14. Real Providers vs. Mocks
- **Unit / happy-path integration**: Mocks only
- **Contract / canary tests**: Opt-in with `-run=^Canary`, may hit real providers behind a secret-scoped API key

## Resource Limits

### 15. Execution Time Budget
- **Unit tests**: ≤ 5s total per package
- **Integration suite**: ≤ 120s overall (enforced by `go test -timeout 2m` in CI)

### 16. Memory & Goroutines
- **Memory**: < 100 MiB RSS per test process
- **Goroutine leak check**: Finish with `runtime.NumGoroutine() <= baseline+5`
- Use `go.uber.org/goleak` in `TestMain`

## Additional Guidelines

### Test Organization
- Place unit tests alongside source files (`foo.go` → `foo_test.go`)
- Integration tests go in `integration/` subdirectories
- Shared test utilities go in `internal/testutil/`

### Test Data
- Store fixtures in `testdata/` directories (ignored by Go tools)
- Generate unique IDs with `uuid.New()` or timestamp-based schemes
- Clean up generated files in `t.Cleanup()`

### Logging and Diagnostics
- Use `t.Log()` for diagnostic output
- On failure, dump relevant state (error stacks, event bus traffic)
- Use `-v` flag to enable verbose logging during development

### Test Naming
- Follow Go conventions: `TestFunctionName_scenario`
- Use descriptive names that explain what's being tested
- Group related tests with subtests (`t.Run()`)

## Exceptions and Deviations

Adopt these defaults, tweak as you uncover edge cases, and document any deviation inline so the next contributor—or agent—knows why the rule was bent.

When deviating from these rules, add a comment explaining:
1. Why the standard approach doesn't work
2. What alternative approach you're using
3. Any risks or trade-offs involved

Example:
```go
// Deviation: Using fixed port 8080 instead of 0 because the legacy
// client library doesn't support dynamic port discovery.
// Risk: May fail if port is already in use.
// TODO: Update once client library supports dynamic ports.
```