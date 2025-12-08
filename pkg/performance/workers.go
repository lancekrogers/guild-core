package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Task represents a unit of work
type Task func(context.Context) error

// Result represents the result of a task execution
type Result struct {
	TaskID int
	Error  error
	Took   time.Duration
}

// WorkerPool provides a pool of workers for executing tasks
type WorkerPool struct {
	workers    int
	queue      chan taskWrapper
	results    chan Result
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	stats      WorkerPoolStats
	started    atomic.Bool
	nextTaskID atomic.Int64
}

// taskWrapper wraps a task with metadata
type taskWrapper struct {
	id        int64
	task      Task
	ctx       context.Context
	submitted time.Time
}

// WorkerPoolStats tracks worker pool performance
type WorkerPoolStats struct {
	Submitted   atomic.Uint64
	Completed   atomic.Uint64
	Failed      atomic.Uint64
	Panics      atomic.Uint64
	QueueDepth  atomic.Int32
	AvgWaitTime atomic.Uint64 // nanoseconds
	AvgExecTime atomic.Uint64 // nanoseconds
}

// WorkerPoolStatsSnapshot represents a point-in-time snapshot of worker pool statistics
type WorkerPoolStatsSnapshot struct {
	Submitted   uint64
	Completed   uint64
	Failed      uint64
	Panics      uint64
	QueueDepth  int32
	AvgWaitTime uint64 // nanoseconds
	AvgExecTime uint64 // nanoseconds
}

// WorkerPoolConfig configures worker pool behavior
type WorkerPoolConfig struct {
	Workers          int
	QueueSize        int
	ResultBufferSize int
	TaskTimeout      time.Duration
	EnableMetrics    bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(cfg WorkerPoolConfig) *WorkerPool {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = cfg.Workers * 10
	}
	if cfg.ResultBufferSize <= 0 {
		cfg.ResultBufferSize = cfg.QueueSize
	}
	if cfg.TaskTimeout == 0 {
		cfg.TaskTimeout = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers: cfg.Workers,
		queue:   make(chan taskWrapper, cfg.QueueSize),
		results: make(chan Result, cfg.ResultBufferSize),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start initializes and starts the worker pool
func (wp *WorkerPool) Start() error {
	if !wp.started.CompareAndSwap(false, true) {
		return gerror.New(gerror.ErrCodeInternal, "worker pool already started", nil)
	}

	// Start workers
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	// Start result collector
	go wp.resultCollector()

	return nil
}

// Submit adds a task to the worker pool
func (wp *WorkerPool) Submit(task Task) error {
	return wp.SubmitWithContext(context.Background(), task)
}

// SubmitWithContext submits a task with a specific context
func (wp *WorkerPool) SubmitWithContext(ctx context.Context, task Task) error {
	if !wp.started.Load() {
		return gerror.New(gerror.ErrCodeInternal, "worker pool not started", nil)
	}

	taskID := wp.nextTaskID.Add(1)
	wrapper := taskWrapper{
		id:        taskID,
		task:      task,
		ctx:       ctx,
		submitted: time.Now(),
	}

	wp.stats.Submitted.Add(1)
	wp.stats.QueueDepth.Add(1)

	select {
	case wp.queue <- wrapper:
		return nil
	case <-wp.ctx.Done():
		wp.stats.QueueDepth.Add(-1)
		return gerror.New(gerror.ErrCodeInternal, "worker pool shutting down", nil)
	case <-ctx.Done():
		wp.stats.QueueDepth.Add(-1)
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled")
	default:
		wp.stats.QueueDepth.Add(-1)
		return gerror.New(gerror.ErrCodeInternal, "queue is full", nil)
	}
}

// SubmitBatch submits multiple tasks efficiently
func (wp *WorkerPool) SubmitBatch(tasks []Task) error {
	for i, task := range tasks {
		if err := wp.Submit(task); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, fmt.Sprintf("failed to submit task %d", i))
		}
	}
	return nil
}

// worker is the main worker goroutine
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case wrapper := <-wp.queue:
			wp.stats.QueueDepth.Add(-1)
			wp.executeTask(wrapper)

		case <-wp.ctx.Done():
			return
		}
	}
}

// executeTask executes a single task with error handling
func (wp *WorkerPool) executeTask(wrapper taskWrapper) {
	start := time.Now()
	waitTime := start.Sub(wrapper.submitted)

	// Update wait time metrics
	wp.updateAvgTime(&wp.stats.AvgWaitTime, uint64(waitTime.Nanoseconds()))

	var err error

	// Execute with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				wp.stats.Panics.Add(1)
				err = gerror.New(gerror.ErrCodeInternal, fmt.Sprintf("task panicked: %v", r), nil)
			}
		}()

		err = wrapper.task(wrapper.ctx)
	}()

	execTime := time.Since(start)
	wp.updateAvgTime(&wp.stats.AvgExecTime, uint64(execTime.Nanoseconds()))

	// Record completion
	if err != nil {
		wp.stats.Failed.Add(1)
	} else {
		wp.stats.Completed.Add(1)
	}

	// Send result
	result := Result{
		TaskID: int(wrapper.id),
		Error:  err,
		Took:   execTime,
	}

	select {
	case wp.results <- result:
	default:
		// Result buffer full, drop result
	}
}

// updateAvgTime updates an average time metric using exponential moving average
func (wp *WorkerPool) updateAvgTime(avg *atomic.Uint64, newValue uint64) {
	for {
		current := avg.Load()
		// Simple exponential moving average: new_avg = 0.9 * old_avg + 0.1 * new_value
		newAvg := (current*9 + newValue) / 10
		if avg.CompareAndSwap(current, newAvg) {
			break
		}
	}
}

// Results returns the results channel for reading task results
func (wp *WorkerPool) Results() <-chan Result {
	return wp.results
}

// resultCollector processes results (mainly for metrics)
func (wp *WorkerPool) resultCollector() {
	for {
		select {
		case <-wp.results:
			// Results are processed by consumers, we just ensure the channel doesn't block
		case <-wp.ctx.Done():
			return
		}
	}
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() error {
	if !wp.started.Load() {
		return gerror.New(gerror.ErrCodeInternal, "worker pool not started", nil)
	}

	wp.cancel()
	wp.wg.Wait()
	wp.started.Store(false)

	close(wp.results)
	return nil
}

// StopWithTimeout stops the worker pool with a timeout
func (wp *WorkerPool) StopWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		wp.Stop()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return gerror.New(gerror.ErrCodeInternal, "worker pool shutdown timeout", nil)
	}
}

// Stats returns current worker pool statistics
func (wp *WorkerPool) Stats() WorkerPoolStatsSnapshot {
	return WorkerPoolStatsSnapshot{
		Submitted:   wp.stats.Submitted.Load(),
		Completed:   wp.stats.Completed.Load(),
		Failed:      wp.stats.Failed.Load(),
		Panics:      wp.stats.Panics.Load(),
		QueueDepth:  wp.stats.QueueDepth.Load(),
		AvgWaitTime: wp.stats.AvgWaitTime.Load(),
		AvgExecTime: wp.stats.AvgExecTime.Load(),
	}
}

// WaitForCompletion waits for all submitted tasks to complete
func (wp *WorkerPool) WaitForCompletion() {
	for wp.stats.QueueDepth.Load() > 0 {
		time.Sleep(10 * time.Millisecond)
	}
}

// WorkStealingQueue implements a work-stealing queue for better load distribution
type WorkStealingQueue struct {
	queues  []chan Task
	workers []*stealingWorker
	stats   WorkStealingStats
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	victim  atomic.Int32
	started atomic.Bool
}

// stealingWorker represents a worker in the work-stealing queue
type stealingWorker struct {
	id      int
	localQ  chan Task
	globalQ *WorkStealingQueue
	stats   WorkerStats
}

// WorkerStats tracks individual worker statistics
type WorkerStats struct {
	Executed atomic.Uint64
	Stolen   atomic.Uint64
	Steals   atomic.Uint64
}

// WorkerStatsSnapshot represents a point-in-time snapshot of worker statistics
type WorkerStatsSnapshot struct {
	Executed uint64
	Stolen   uint64
	Steals   uint64
}

// WorkStealingStats tracks work-stealing queue statistics
type WorkStealingStats struct {
	Submitted     atomic.Uint64
	Executed      atomic.Uint64
	StealAttempts atomic.Uint64
	StealSuccess  atomic.Uint64
}

// WorkStealingStatsSnapshot represents a point-in-time snapshot of work-stealing statistics
type WorkStealingStatsSnapshot struct {
	Submitted     uint64
	Executed      uint64
	StealAttempts uint64
	StealSuccess  uint64
}

// NewWorkStealingQueue creates a new work-stealing queue
func NewWorkStealingQueue(workers int, queueSize int) *WorkStealingQueue {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if queueSize <= 0 {
		queueSize = 256
	}

	ctx, cancel := context.WithCancel(context.Background())

	wsq := &WorkStealingQueue{
		queues:  make([]chan Task, workers),
		workers: make([]*stealingWorker, workers),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize workers and their local queues
	for i := 0; i < workers; i++ {
		localQ := make(chan Task, queueSize)
		wsq.queues[i] = localQ
		wsq.workers[i] = &stealingWorker{
			id:      i,
			localQ:  localQ,
			globalQ: wsq,
		}
	}

	return wsq
}

// Start begins the work-stealing queue
func (wsq *WorkStealingQueue) Start() error {
	if !wsq.started.CompareAndSwap(false, true) {
		return gerror.New(gerror.ErrCodeInternal, "work-stealing queue already started", nil)
	}

	for _, worker := range wsq.workers {
		wsq.wg.Add(1)
		go worker.run()
	}

	return nil
}

// Submit adds a task to the work-stealing queue
func (wsq *WorkStealingQueue) Submit(task Task) error {
	if !wsq.started.Load() {
		return gerror.New(gerror.ErrCodeInternal, "work-stealing queue not started", nil)
	}

	wsq.stats.Submitted.Add(1)

	// Try to submit to the least loaded queue
	minDepth := len(wsq.queues[0])
	targetQueue := 0

	for i, queue := range wsq.queues {
		if len(queue) < minDepth {
			minDepth = len(queue)
			targetQueue = i
		}
	}

	select {
	case wsq.queues[targetQueue] <- task:
		return nil
	case <-wsq.ctx.Done():
		return gerror.New(gerror.ErrCodeInternal, "work-stealing queue shutting down", nil)
	default:
		return gerror.New(gerror.ErrCodeInternal, "all queues are full", nil)
	}
}

// run executes the worker's main loop
func (sw *stealingWorker) run() {
	defer sw.globalQ.wg.Done()

	for {
		select {
		case task := <-sw.localQ:
			sw.executeTask(task)

		case <-sw.globalQ.ctx.Done():
			return

		default:
			// Try to steal work
			if stolen := sw.steal(); stolen != nil {
				sw.executeTask(stolen)
			} else {
				// No work available, yield
				runtime.Gosched()
			}
		}
	}
}

// steal attempts to steal work from other workers
func (sw *stealingWorker) steal() Task {
	sw.globalQ.stats.StealAttempts.Add(1)
	sw.stats.Steals.Add(1)

	// Start from a random position to avoid always stealing from the same worker
	start := sw.globalQ.victim.Add(1) % int32(len(sw.globalQ.queues))

	for i := 0; i < len(sw.globalQ.queues); i++ {
		idx := (start + int32(i)) % int32(len(sw.globalQ.queues))
		if int(idx) == sw.id {
			continue // Don't steal from yourself
		}

		select {
		case task := <-sw.globalQ.queues[idx]:
			sw.globalQ.stats.StealSuccess.Add(1)
			sw.stats.Stolen.Add(1)
			return task
		default:
			continue
		}
	}

	return nil
}

// executeTask executes a task with error handling
func (sw *stealingWorker) executeTask(task Task) {
	defer func() {
		if r := recover(); r != nil {
			// Handle panic gracefully
		}
	}()

	sw.stats.Executed.Add(1)
	sw.globalQ.stats.Executed.Add(1)

	task(sw.globalQ.ctx)
}

// Stop gracefully shuts down the work-stealing queue
func (wsq *WorkStealingQueue) Stop() {
	if wsq.started.Load() {
		wsq.cancel()
		wsq.wg.Wait()
		wsq.started.Store(false)

		for _, queue := range wsq.queues {
			close(queue)
		}
	}
}

// Stats returns current work-stealing queue statistics
func (wsq *WorkStealingQueue) Stats() WorkStealingStatsSnapshot {
	return WorkStealingStatsSnapshot{
		Submitted:     wsq.stats.Submitted.Load(),
		Executed:      wsq.stats.Executed.Load(),
		StealAttempts: wsq.stats.StealAttempts.Load(),
		StealSuccess:  wsq.stats.StealSuccess.Load(),
	}
}

// WorkerStats returns statistics for individual workers
func (wsq *WorkStealingQueue) WorkerStats() []WorkerStatsSnapshot {
	stats := make([]WorkerStatsSnapshot, len(wsq.workers))
	for i, worker := range wsq.workers {
		stats[i] = WorkerStatsSnapshot{
			Executed: worker.stats.Executed.Load(),
			Stolen:   worker.stats.Stolen.Load(),
			Steals:   worker.stats.Steals.Load(),
		}
	}
	return stats
}
