# Go Concurrency Patterns

This document explains common concurrency patterns used in the Guild framework.

## Overview

Go's concurrency primitives (goroutines and channels) are central to Guild's design, enabling agents to work in parallel while coordinating through shared structures. This document covers the core patterns used throughout the codebase.

## Goroutines

Goroutines are lightweight threads managed by the Go runtime. In Guild, they're used for:

1. **Running agents concurrently**
2. **Handling asynchronous events**
3. **Processing tasks in parallel**
4. **Managing background operations**

### Basic Pattern

```go
// Launch a goroutine
go func() {
    // Do work
    fmt.Println("Working in background")
}()

// Continue with other operations
fmt.Println("Main thread continues")
```

### With Context

```go
// Create a context with cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // Ensure resources are released

// Start a worker goroutine
go func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            // Context was cancelled
            fmt.Println("Worker shutting down")
            return
        default:
            // Do some work
            fmt.Println("Worker doing task")
            time.Sleep(1 * time.Second)
        }
    }
}(ctx)

// Cancel after 5 seconds
time.Sleep(5 * time.Second)
cancel()
```

## Channels

Channels are typed conduits for communicating between goroutines. Guild uses channels for:

1. **Passing tasks between components**
2. **Signaling events**
3. **Coordinating agent activity**
4. **Managing resource access**

### Basic Channel Usage

```go
// Create a channel
ch := make(chan string)

// Send from one goroutine
go func() {
    ch <- "Hello from goroutine"
}()

// Receive in another goroutine
go func() {
    msg := <-ch
    fmt.Println(msg)
}()
```

### Buffered Channels

```go
// Create a buffered channel with capacity 5
tasks := make(chan Task, 5)

// Producer can send up to 5 tasks without blocking
for i := 0; i < 5; i++ {
    tasks <- Task{ID: fmt.Sprintf("task-%d", i)}
}

// Consumer processes tasks
go func() {
    for task := range tasks {
        processTask(task)
    }
}()
```

### Select Pattern

```go
// Multiple channels
taskCh := make(chan Task)
doneCh := make(chan bool)
errCh := make(chan error)

// Process from multiple channels
go func() {
    for {
        select {
        case task := <-taskCh:
            // Process task
            fmt.Printf("Processing task: %s\n", task.ID)

        case <-doneCh:
            // All tasks complete
            fmt.Println("Work complete")
            return

        case err := <-errCh:
            // Handle error
            fmt.Printf("Error: %v\n", err)

        case <-time.After(5 * time.Second):
            // Timeout
            fmt.Println("Timed out waiting for tasks")
            return
        }
    }
}()
```

## Worker Pool Pattern

Guild uses worker pools for parallel task processing.

```go
// pkg/orchestrator/runner.go
package orchestrator

import (
	"context"
	"sync"
)

// TaskRunner manages parallel task execution
type TaskRunner struct {
	numWorkers int
	tasks      chan Task
	results    chan Result
	done       chan struct{}
	wg         sync.WaitGroup
}

// NewTaskRunner creates a new task runner
func NewTaskRunner(numWorkers int) *TaskRunner {
	return &TaskRunner{
		numWorkers: numWorkers,
		tasks:      make(chan Task, numWorkers),
		results:    make(chan Result, numWorkers),
		done:       make(chan struct{}),
	}
}

// Start launches worker goroutines
func (r *TaskRunner) Start(ctx context.Context) {
	for i := 0; i < r.numWorkers; i++ {
		r.wg.Add(1)
		go r.worker(ctx, i)
	}
}

// worker processes tasks
func (r *TaskRunner) worker(ctx context.Context, id int) {
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled
			return

		case <-r.done:
			// Runner shutting down
			return

		case task, ok := <-r.tasks:
			if !ok {
				// Channel closed
				return
			}

			// Process task
			result := executeTask(ctx, task)

			// Send result
			select {
			case r.results <- result:
				// Result sent
			case <-ctx.Done():
				// Context cancelled
				return
			case <-r.done:
				// Runner shutting down
				return
			}
		}
	}
}

// Submit adds a task to the queue
func (r *TaskRunner) Submit(task Task) {
	r.tasks <- task
}

// Results returns the results channel
func (r *TaskRunner) Results() <-chan Result {
	return r.results
}

// Stop shuts down the runner
func (r *TaskRunner) Stop() {
	close(r.done)
	r.wg.Wait()
	close(r.results)
}

// Usage example
func RunTasks(ctx context.Context, tasks []Task) []Result {
	// Create runner with 5 workers
	runner := NewTaskRunner(5)

	// Start workers
	runner.Start(ctx)

	// Collect results
	var results []Result

	// Start result collector
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range runner.Results() {
			results = append(results, result)
		}
	}()

	// Submit tasks
	for _, task := range tasks {
		runner.Submit(task)
	}

	// Stop runner
	runner.Stop()

	// Wait for result collection
	wg.Wait()

	return results
}
```

## Fan-Out, Fan-In Pattern

Used for parallel processing with result aggregation.

```go
// fanOut splits work across multiple goroutines
func fanOut(ctx context.Context, tasks []Task) <-chan Result {
	resultCh := make(chan Result)

	// Launch worker for each task
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(task Task) {
			defer wg.Done()

			// Process task
			result := executeTask(ctx, task)

			// Send result
			select {
			case resultCh <- result:
				// Result sent
			case <-ctx.Done():
				// Context cancelled
				return
			}
		}(task)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	return resultCh
}

// fanIn combines results from multiple channels
func fanIn(ctx context.Context, channels ...<-chan Result) <-chan Result {
	var wg sync.WaitGroup
	multiplexed := make(chan Result)

	// Start multiplex function for each channel
	multiplex := func(ch <-chan Result) {
		defer wg.Done()
		for result := range ch {
			select {
			case multiplexed <- result:
				// Result forwarded
			case <-ctx.Done():
				// Context cancelled
				return
			}
		}
	}

	// Set up multiplexing
	wg.Add(len(channels))
	for _, ch := range channels {
		go multiplex(ch)
	}

	// Close multiplexed channel when all inputs complete
	go func() {
		wg.Wait()
		close(multiplexed)
	}()

	return multiplexed
}

// Usage example
func ProcessObjective(ctx context.Context, objective Objective) []Result {
	// Split objective into tasks
	tasks := splitIntoTasks(objective)

	// Group tasks by agent
	tasksByAgent := groupTasksByAgent(tasks)

	// Create result channels, one per agent
	resultChannels := make([]<-chan Result, 0, len(tasksByAgent))

	// Process each agent's tasks in parallel
	for agent, agentTasks := range tasksByAgent {
		// Fan out tasks for this agent
		resultCh := fanOut(ctx, agentTasks)
		resultChannels = append(resultChannels, resultCh)
	}

	// Combine all results
	combinedResults := fanIn(ctx, resultChannels...)

	// Collect results
	var results []Result
	for result := range combinedResults {
		results = append(results, result)
	}

	return results
}
```

## Context Usage

Context provides cancellation, deadlines, and value propagation.

### Cancellation

```go
// Create a context with cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // Always call cancel to release resources

// Use in function calls
result, err := agent.Execute(ctx, task)
if err != nil {
	// Handle error
	cancel() // Cancel ongoing operations
}
```

### Timeout

```go
// Create a context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Execute with timeout
result, err := agent.Execute(ctx, task)
if err != nil {
	if ctx.Err() == context.DeadlineExceeded {
		// Handle timeout
		return Result{}, fmt.Errorf("task execution timed out")
	}
	// Handle other errors
	return Result{}, err
}
```

### Value

```go
// Create a context with value
ctx := context.WithValue(context.Background(), "traceID", uuid.New().String())

// Retrieve value in another function
func executeWithTracing(ctx context.Context) {
	traceID, ok := ctx.Value("traceID").(string)
	if !ok {
		traceID = "unknown"
	}

	log.Printf("[Trace: %s] Starting execution", traceID)
	// ...
}
```

## Mutex and Synchronization

Guild uses mutexes for protecting shared data.

```go
// Safe counter with mutex
type SafeCounter struct {
	value int
	mutex sync.Mutex
}

// Increment safely
func (c *SafeCounter) Increment() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value++
}

// GetValue safely
func (c *SafeCounter) GetValue() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.value
}
```

### RWMutex for Read-Heavy Workloads

```go
// Cache with read-write mutex
type Cache struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// Get is read-only operation
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	value, ok := c.data[key]
	return value, ok
}

// Set is write operation
func (c *Cache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = value
}
```

## Error Handling in Concurrent Code

Guild follows these patterns for error handling in concurrent code:

### Error Channels

```go
// Worker with error channel
func worker(ctx context.Context, tasks <-chan Task, results chan<- Result, errors chan<- error) {
	for task := range tasks {
		result, err := executeTask(ctx, task)
		if err != nil {
			select {
			case errors <- err:
				// Error sent
			case <-ctx.Done():
				return
			}
			continue
		}

		select {
		case results <- result:
			// Result sent
		case <-ctx.Done():
			return
		}
	}
}

// Usage
func processAllTasks(tasks []Task) ([]Result, error) {
	taskCh := make(chan Task, len(tasks))
	resultCh := make(chan Result, len(tasks))
	errorCh := make(chan error, len(tasks))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workers
	for i := 0; i < 5; i++ {
		go worker(ctx, taskCh, resultCh, errorCh)
	}

	// Send tasks
	for _, task := range tasks {
		taskCh <- task
	}
	close(taskCh)

	// Collect results and errors
	var results []Result
	var firstErr error

	for i := 0; i < len(tasks); i++ {
		select {
		case result := <-resultCh:
			results = append(results, result)
		case err := <-errorCh:
			if firstErr == nil {
				firstErr = err
				cancel() // Cancel remaining operations
			}
		}
	}

	return results, firstErr
}
```

### WaitGroup with Error Collection

```go
// Result with potential error
type TaskResult struct {
	Result Result
	Error  error
}

// Execute tasks concurrently
func ExecuteTasks(ctx context.Context, tasks []Task) []TaskResult {
	results := make([]TaskResult, len(tasks))

	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for i, task := range tasks {
		// Capture loop variables
		i, task := i, task

		go func() {
			defer wg.Done()

			// Execute task
			result, err := executeTask(ctx, task)

			// Store result and error
			results[i] = TaskResult{
				Result: result,
				Error:  err,
			}
		}()
	}

	wg.Wait()
	return results
}
```

## Best Practices

1. **Always use context for cancellation**

   - Pass context to all functions that might block
   - Respect context cancellation

2. **Avoid goroutine leaks**

   - Use `defer` to ensure resources are released
   - Make sure goroutines can exit when no longer needed
   - Close channels when done sending

3. **Handle channel operations safely**

   - Check for closed channels with `value, ok := <-ch`
   - Use `select` with `default` for non-blocking operations
   - Consider buffered channels for bursty workloads

4. **Protect shared resources**

   - Use mutexes for mutable data
   - Consider RWMutex for read-heavy workloads
   - Keep critical sections small

5. **Error handling**
   - Propagate errors through channels
   - Use WaitGroups to collect errors
   - Cancel context on critical errors

## Related Documentation

- [Go Concurrency Patterns](https://blog.golang.org/pipelines)
- [../patterns/interface_first.md](../patterns/interface_first.md)
- [../architecture/guild_runtime.md](../architecture/guild_runtime.md)
