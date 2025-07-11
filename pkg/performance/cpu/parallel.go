package cpu

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// ParallelProcessor provides utilities for parallel data processing
type ParallelProcessor struct {
	workers     int
	chunkSize   int
	concurrency int
}

// ParallelConfig configures parallel processing behavior
type ParallelConfig struct {
	Workers     int
	ChunkSize   int
	Concurrency int
}

// NewParallelProcessor creates a new parallel processor
func NewParallelProcessor(cfg ParallelConfig) *ParallelProcessor {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 1000
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = cfg.Workers
	}

	return &ParallelProcessor{
		workers:     cfg.Workers,
		chunkSize:   cfg.ChunkSize,
		concurrency: cfg.Concurrency,
	}
}

// ProcessSliceBytes processes a byte slice in parallel using the provided function
func (pp *ParallelProcessor) ProcessSliceBytes(ctx context.Context, data []byte, fn func(context.Context, []byte) error) error {
	if len(data) == 0 {
		return nil
	}

	// Calculate chunk size
	chunkSize := len(data) / pp.workers
	if chunkSize < pp.chunkSize {
		chunkSize = pp.chunkSize
	}
	if chunkSize == 0 {
		chunkSize = 1
	}

	// Create chunks
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	// Process chunks in parallel
	return pp.processChunksBytes(ctx, chunks, fn)
}

// processChunksBytes processes byte chunks in parallel with concurrency control
func (pp *ParallelProcessor) processChunksBytes(ctx context.Context, chunks [][]byte, fn func(context.Context, []byte) error) error {
	semaphore := make(chan struct{}, pp.concurrency)
	errCh := make(chan error, len(chunks))
	var wg sync.WaitGroup

	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunk []byte) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
			defer func() { <-semaphore }()

			// Process chunk
			if err := fn(ctx, chunk); err != nil {
				errCh <- err
			}
		}(chunk)
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// MapInts applies a function to each integer in parallel
func (pp *ParallelProcessor) MapInts(ctx context.Context, data []int, fn func(context.Context, int) (int, error)) ([]int, error) {
	if len(data) == 0 {
		return []int{}, nil
	}

	results := make([]int, len(data))
	var wg sync.WaitGroup
	var firstErr atomic.Value // Store first error
	semaphore := make(chan struct{}, pp.concurrency)

	for i, item := range data {
		wg.Add(1)
		go func(idx int, item int) {
			defer wg.Done()

			// Check if we already have an error
			if firstErr.Load() != nil {
				return
			}

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				firstErr.Store(ctx.Err())
				return
			}
			defer func() { <-semaphore }()

			// Process item
			result, err := fn(ctx, item)
			if err != nil {
				firstErr.Store(err)
				return
			}

			results[idx] = result
		}(i, item)
	}

	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return nil, err.(error)
	}

	return results, nil
}

// MapBools applies a predicate to each integer returning booleans in parallel
func (pp *ParallelProcessor) MapBools(ctx context.Context, data []int, fn func(context.Context, int) (bool, error)) ([]bool, error) {
	if len(data) == 0 {
		return []bool{}, nil
	}

	results := make([]bool, len(data))
	var wg sync.WaitGroup
	var firstErr atomic.Value // Store first error
	semaphore := make(chan struct{}, pp.concurrency)

	for i, item := range data {
		wg.Add(1)
		go func(idx int, item int) {
			defer wg.Done()

			// Check if we already have an error
			if firstErr.Load() != nil {
				return
			}

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				firstErr.Store(ctx.Err())
				return
			}
			defer func() { <-semaphore }()

			// Process item
			result, err := fn(ctx, item)
			if err != nil {
				firstErr.Store(err)
				return
			}

			results[idx] = result
		}(i, item)
	}

	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return nil, err.(error)
	}

	return results, nil
}

// ReduceInts applies a reduction function to integers in parallel
func (pp *ParallelProcessor) ReduceInts(ctx context.Context, data []int, identity int, fn func(context.Context, int, int) (int, error), combine func(int, int) int) (int, error) {
	if len(data) == 0 {
		return identity, nil
	}

	// Calculate chunk size
	chunkSize := len(data) / pp.workers
	if chunkSize == 0 {
		chunkSize = 1
	}

	// Create chunks
	var chunks [][]int
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	// Process chunks in parallel
	results := make([]int, len(chunks))
	var wg sync.WaitGroup
	var firstErr atomic.Value

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, chunk []int) {
			defer wg.Done()

			if firstErr.Load() != nil {
				return
			}

			// Reduce chunk
			result := identity
			for _, item := range chunk {
				var err error
				result, err = fn(ctx, result, item)
				if err != nil {
					firstErr.Store(err)
					return
				}
			}

			results[idx] = result
		}(i, chunk)
	}

	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return identity, err.(error)
	}

	// Combine results
	final := identity
	for _, result := range results {
		final = combine(final, result)
	}

	return final, nil
}

// FilterInts filters integers in parallel
func (pp *ParallelProcessor) FilterInts(ctx context.Context, data []int, predicate func(context.Context, int) (bool, error)) ([]int, error) {
	if len(data) == 0 {
		return []int{}, nil
	}

	// Use MapInts to get boolean results
	keep, err := pp.MapBools(ctx, data, predicate)
	if err != nil {
		return nil, err
	}

	// Filter based on results
	var result []int
	for i, shouldKeep := range keep {
		if shouldKeep {
			result = append(result, data[i])
		}
	}

	return result, nil
}

// ParallelForInts executes a function for each integer in parallel
func (pp *ParallelProcessor) ParallelForInts(ctx context.Context, data []int, fn func(context.Context, int, int) error) error {
	if len(data) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var firstErr atomic.Value
	semaphore := make(chan struct{}, pp.concurrency)

	for i, item := range data {
		wg.Add(1)
		go func(idx int, item int) {
			defer wg.Done()

			if firstErr.Load() != nil {
				return
			}

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				firstErr.Store(ctx.Err())
				return
			}
			defer func() { <-semaphore }()

			if err := fn(ctx, idx, item); err != nil {
				firstErr.Store(err)
			}
		}(i, item)
	}

	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}

	return nil
}

// Pipeline processes data through a series of stages in parallel
type Pipeline[T any] struct {
	stages []Stage[T]
	config PipelineConfig
}

// Stage represents a processing stage in the pipeline
type Stage[T any] struct {
	Name     string
	Process  func(context.Context, T) (T, error)
	Parallel bool
	Workers  int
}

// PipelineConfig configures pipeline behavior
type PipelineConfig struct {
	BufferSize int
	MaxWorkers int
}

// NewPipeline creates a new processing pipeline
func NewPipeline[T any](cfg PipelineConfig) *Pipeline[T] {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 100
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = runtime.NumCPU()
	}

	return &Pipeline[T]{
		config: cfg,
	}
}

// AddStage adds a processing stage to the pipeline
func (p *Pipeline[T]) AddStage(stage Stage[T]) {
	if stage.Workers <= 0 {
		stage.Workers = p.config.MaxWorkers
	}
	p.stages = append(p.stages, stage)
}

// Process processes data through the entire pipeline
func (p *Pipeline[T]) Process(ctx context.Context, data []T) ([]T, error) {
	if len(p.stages) == 0 {
		return data, nil
	}

	current := data
	for _, stage := range p.stages {
		result, err := p.processStage(ctx, stage, current)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "stage failed")
		}
		current = result
	}

	return current, nil
}

// processStage processes data through a single stage
func (p *Pipeline[T]) processStage(ctx context.Context, stage Stage[T], data []T) ([]T, error) {
	if !stage.Parallel || len(data) <= stage.Workers {
		// Sequential processing
		result := make([]T, len(data))
		for i, item := range data {
			processed, err := stage.Process(ctx, item)
			if err != nil {
				return nil, err
			}
			result[i] = processed
		}
		return result, nil
	}

	// Parallel processing

	// Since Map method doesn't exist, we'll implement inline
	result := make([]T, len(data))
	var wg sync.WaitGroup
	var firstErr atomic.Value
	semaphore := make(chan struct{}, stage.Workers)

	for i, item := range data {
		wg.Add(1)
		go func(idx int, item T) {
			defer wg.Done()

			if firstErr.Load() != nil {
				return
			}

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				firstErr.Store(ctx.Err())
				return
			}
			defer func() { <-semaphore }()

			processed, err := stage.Process(ctx, item)
			if err != nil {
				firstErr.Store(err)
				return
			}

			result[idx] = processed
		}(i, item)
	}

	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return nil, err.(error)
	}

	return result, nil
}

// StreamingPipeline processes data in a streaming fashion
type StreamingPipeline[T any] struct {
	stages []StreamingStage[T]
	config PipelineConfig
}

// StreamingStage represents a streaming processing stage
type StreamingStage[T any] struct {
	Name    string
	Process func(context.Context, <-chan T, chan<- T) error
	Workers int
}

// NewStreamingPipeline creates a new streaming pipeline
func NewStreamingPipeline[T any](cfg PipelineConfig) *StreamingPipeline[T] {
	return &StreamingPipeline[T]{
		config: cfg,
	}
}

// AddStage adds a streaming stage to the pipeline
func (sp *StreamingPipeline[T]) AddStage(stage StreamingStage[T]) {
	if stage.Workers <= 0 {
		stage.Workers = 1
	}
	sp.stages = append(sp.stages, stage)
}

// Process processes data through the streaming pipeline
func (sp *StreamingPipeline[T]) Process(ctx context.Context, input <-chan T) (<-chan T, error) {
	if len(sp.stages) == 0 {
		return input, nil
	}

	current := input
	for _, stage := range sp.stages {
		output := make(chan T, sp.config.BufferSize)

		// Start workers for this stage
		var wg sync.WaitGroup
		for w := 0; w < stage.Workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stage.Process(ctx, current, output)
			}()
		}

		// Close output when all workers are done
		go func() {
			wg.Wait()
			close(output)
		}()

		current = output
	}

	return current, nil
}

// WorkerPool provides a reusable pool of workers for CPU-intensive tasks
type WorkerPool[T any] struct {
	workers   int
	taskQueue chan task[T]
	results   chan result[T]
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// task represents a unit of work
type task[T any] struct {
	id   int
	data T
	fn   func(context.Context, T) (T, error)
}

// result represents the result of a task
type result[T any] struct {
	id   int
	data T
	err  error
}

// NewWorkerPool creates a new worker pool for CPU tasks
func NewWorkerPool[T any](workers int) *WorkerPool[T] {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool[T]{
		workers:   workers,
		taskQueue: make(chan task[T], workers*2),
		results:   make(chan result[T], workers*2),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker processes tasks from the queue
func (wp *WorkerPool[T]) worker() {
	defer wp.wg.Done()

	for {
		select {
		case task := <-wp.taskQueue:
			data, err := task.fn(wp.ctx, task.data)
			wp.results <- result[T]{
				id:   task.id,
				data: data,
				err:  err,
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool[T]) Submit(id int, data T, fn func(context.Context, T) (T, error)) error {
	select {
	case wp.taskQueue <- task[T]{id: id, data: data, fn: fn}:
		return nil
	case <-wp.ctx.Done():
		return gerror.New(gerror.ErrCodeInternal, "worker pool shutting down", nil)
	default:
		return gerror.New(gerror.ErrCodeInternal, "task queue full", nil)
	}
}

// Results returns the results channel
func (wp *WorkerPool[T]) Results() <-chan result[T] {
	return wp.results
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool[T]) Shutdown() {
	wp.cancel()
	wp.wg.Wait()
	close(wp.results)
}

// CPUIntensiveTask represents a CPU-intensive computation
type CPUIntensiveTask interface {
	Execute(context.Context) error
	Split(chunks int) []CPUIntensiveTask
	Combine(results []interface{}) (interface{}, error)
}

// ParallelExecutor executes CPU-intensive tasks in parallel
type ParallelExecutor struct {
	maxConcurrency int
	chunkSize      int
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(maxConcurrency int) *ParallelExecutor {
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.NumCPU()
	}

	return &ParallelExecutor{
		maxConcurrency: maxConcurrency,
		chunkSize:      1000,
	}
}

// Execute executes a CPU-intensive task in parallel
func (pe *ParallelExecutor) Execute(ctx context.Context, task CPUIntensiveTask) (interface{}, error) {
	// Split task into chunks
	chunks := task.Split(pe.maxConcurrency)
	if len(chunks) == 1 {
		// No need for parallelization
		err := task.Execute(ctx)
		return nil, err
	}

	// Execute chunks in parallel
	results := make([]interface{}, len(chunks))
	var wg sync.WaitGroup
	var firstErr atomic.Value
	semaphore := make(chan struct{}, pe.maxConcurrency)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, chunk CPUIntensiveTask) {
			defer wg.Done()

			if firstErr.Load() != nil {
				return
			}

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				firstErr.Store(ctx.Err())
				return
			}
			defer func() { <-semaphore }()

			// Execute chunk
			if err := chunk.Execute(ctx); err != nil {
				firstErr.Store(err)
				return
			}

			results[idx] = chunk
		}(i, chunk)
	}

	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return nil, err.(error)
	}

	// Combine results
	return task.Combine(results)
}
