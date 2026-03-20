package cpu

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// BatchProcessor provides efficient batch processing for CPU-intensive operations
type BatchProcessor[T any] struct {
	batchSize int
	maxWait   time.Duration
	workers   int
	processor func(context.Context, []T) error
	inputCh   chan T
	batchCh   chan []T
	done      chan struct{}
	wg        sync.WaitGroup
	stats     BatchStats
}

// BatchConfig configures batch processing behavior
type BatchConfig struct {
	BatchSize int
	MaxWait   time.Duration
	Workers   int
}

// BatchStats tracks batch processing statistics
type BatchStats struct {
	ItemsProcessed   atomic.Uint64
	BatchesCreated   atomic.Uint64
	BatchesProcessed atomic.Uint64
	ProcessingTime   atomic.Uint64 // nanoseconds
	Errors           atomic.Uint64
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor[T any](cfg BatchConfig, processor func(context.Context, []T) error) *BatchProcessor[T] {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.MaxWait == 0 {
		cfg.MaxWait = 100 * time.Millisecond
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	bp := &BatchProcessor[T]{
		batchSize: cfg.BatchSize,
		maxWait:   cfg.MaxWait,
		workers:   cfg.Workers,
		processor: processor,
		inputCh:   make(chan T, cfg.BatchSize*2),
		batchCh:   make(chan []T, cfg.Workers*2),
		done:      make(chan struct{}),
	}

	// Start batch assembler
	bp.wg.Add(1)
	go bp.batchAssembler()

	// Start workers
	for i := 0; i < cfg.Workers; i++ {
		bp.wg.Add(1)
		go bp.worker()
	}

	return bp
}

// Submit submits an item for batch processing
func (bp *BatchProcessor[T]) Submit(ctx context.Context, item T) error {
	select {
	case bp.inputCh <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-bp.done:
		return gerror.New(gerror.ErrCodeInternal, "batch processor stopped", nil)
	}
}

// SubmitBatch submits a pre-formed batch for processing
func (bp *BatchProcessor[T]) SubmitBatch(ctx context.Context, batch []T) error {
	if len(batch) == 0 {
		return nil
	}

	select {
	case bp.batchCh <- batch:
		bp.stats.BatchesCreated.Add(1)
		bp.stats.ItemsProcessed.Add(uint64(len(batch)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-bp.done:
		return gerror.New(gerror.ErrCodeInternal, "batch processor stopped", nil)
	}
}

// batchAssembler assembles individual items into batches
func (bp *BatchProcessor[T]) batchAssembler() {
	defer bp.wg.Done()

	batch := make([]T, 0, bp.batchSize)
	timer := time.NewTimer(bp.maxWait)
	timer.Stop()

	for {
		select {
		case item := <-bp.inputCh:
			batch = append(batch, item)
			bp.stats.ItemsProcessed.Add(1)

			// Start timer on first item
			if len(batch) == 1 {
				timer.Reset(bp.maxWait)
			}

			// Send batch when full
			if len(batch) >= bp.batchSize {
				bp.sendBatch(batch)
				batch = make([]T, 0, bp.batchSize)
				timer.Stop()
			}

		case <-timer.C:
			// Send partial batch on timeout
			if len(batch) > 0 {
				bp.sendBatch(batch)
				batch = make([]T, 0, bp.batchSize)
			}

		case <-bp.done:
			// Send remaining batch
			if len(batch) > 0 {
				bp.sendBatch(batch)
			}
			timer.Stop()
			return
		}
	}
}

// sendBatch sends a batch to workers
func (bp *BatchProcessor[T]) sendBatch(batch []T) {
	batchCopy := make([]T, len(batch))
	copy(batchCopy, batch)

	select {
	case bp.batchCh <- batchCopy:
		bp.stats.BatchesCreated.Add(1)
	case <-bp.done:
		return
	}
}

// worker processes batches
func (bp *BatchProcessor[T]) worker() {
	defer bp.wg.Done()

	for {
		select {
		case batch := <-bp.batchCh:
			start := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := bp.processor(ctx, batch)
			cancel()

			duration := time.Since(start)
			bp.stats.ProcessingTime.Add(uint64(duration.Nanoseconds()))
			bp.stats.BatchesProcessed.Add(1)

			if err != nil {
				bp.stats.Errors.Add(1)
			}

		case <-bp.done:
			return
		}
	}
}

// Stop stops the batch processor
func (bp *BatchProcessor[T]) Stop() {
	close(bp.done)
	bp.wg.Wait()
}

// Stats returns current processing statistics
func (bp *BatchProcessor[T]) Stats() BatchStats {
	return BatchStats{
		ItemsProcessed:   atomic.Uint64{},
		BatchesCreated:   atomic.Uint64{},
		BatchesProcessed: atomic.Uint64{},
		ProcessingTime:   atomic.Uint64{},
		Errors:           atomic.Uint64{},
	}
}

// PipelineBatchProcessor processes items through multiple batch processing stages
type PipelineBatchProcessor[T any] struct {
	stages []BatchProcessorStage[T]
	config PipelineBatchConfig
}

// BatchProcessorStage represents a stage in the pipeline
type BatchProcessorStage[T any] struct {
	Name      string
	Processor func(context.Context, []T) ([]T, error)
	Config    BatchConfig
}

// PipelineBatchConfig configures pipeline batch processing
type PipelineBatchConfig struct {
	BufferSize int
	Timeout    time.Duration
}

// NewPipelineBatchProcessor creates a new pipeline batch processor
func NewPipelineBatchProcessor[T any](cfg PipelineBatchConfig) *PipelineBatchProcessor[T] {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	return &PipelineBatchProcessor[T]{
		config: cfg,
	}
}

// AddStage adds a processing stage to the pipeline
func (pbp *PipelineBatchProcessor[T]) AddStage(stage BatchProcessorStage[T]) {
	pbp.stages = append(pbp.stages, stage)
}

// Process processes items through all pipeline stages
func (pbp *PipelineBatchProcessor[T]) Process(ctx context.Context, items []T) ([]T, error) {
	current := items

	for _, stage := range pbp.stages {
		result, err := pbp.processStage(ctx, stage, current)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "stage failed")
		}
		current = result
	}

	return current, nil
}

// processStage processes items through a single stage
func (pbp *PipelineBatchProcessor[T]) processStage(ctx context.Context, stage BatchProcessorStage[T], items []T) ([]T, error) {
	if len(items) == 0 {
		return items, nil
	}

	// Process in batches
	var results []T
	batchSize := stage.Config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		batchResult, err := stage.Processor(ctx, batch)
		if err != nil {
			return nil, err
		}

		results = append(results, batchResult...)
	}

	return results, nil
}

// AdaptiveBatchProcessor dynamically adjusts batch size based on performance
type AdaptiveBatchProcessor[T any] struct {
	minBatchSize    int
	maxBatchSize    int
	currentBatch    int
	targetLatency   time.Duration
	adjustmentRate  float64
	processor       func(context.Context, []T) error
	inputCh         chan T
	done            chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	recentLatencies []time.Duration
	stats           AdaptiveStats
}

// AdaptiveConfig configures adaptive batch processing
type AdaptiveConfig struct {
	MinBatchSize   int
	MaxBatchSize   int
	TargetLatency  time.Duration
	AdjustmentRate float64
	Workers        int
}

// AdaptiveStats tracks adaptive batch processing statistics
type AdaptiveStats struct {
	CurrentBatchSize atomic.Int32
	AverageLatency   atomic.Uint64 // nanoseconds
	Adjustments      atomic.Uint64
	OptimalBatchSize atomic.Int32
}

// NewAdaptiveBatchProcessor creates a new adaptive batch processor
func NewAdaptiveBatchProcessor[T any](cfg AdaptiveConfig, processor func(context.Context, []T) error) *AdaptiveBatchProcessor[T] {
	if cfg.MinBatchSize <= 0 {
		cfg.MinBatchSize = 10
	}
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 1000
	}
	if cfg.TargetLatency == 0 {
		cfg.TargetLatency = 100 * time.Millisecond
	}
	if cfg.AdjustmentRate <= 0 {
		cfg.AdjustmentRate = 0.1
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	abp := &AdaptiveBatchProcessor[T]{
		minBatchSize:    cfg.MinBatchSize,
		maxBatchSize:    cfg.MaxBatchSize,
		currentBatch:    cfg.MinBatchSize,
		targetLatency:   cfg.TargetLatency,
		adjustmentRate:  cfg.AdjustmentRate,
		processor:       processor,
		inputCh:         make(chan T, cfg.MaxBatchSize*2),
		done:            make(chan struct{}),
		recentLatencies: make([]time.Duration, 0, 10),
	}

	abp.stats.CurrentBatchSize.Store(int32(cfg.MinBatchSize))
	abp.stats.OptimalBatchSize.Store(int32(cfg.MinBatchSize))

	// Start workers
	for i := 0; i < cfg.Workers; i++ {
		abp.wg.Add(1)
		go abp.adaptiveWorker()
	}

	return abp
}

// Submit submits an item for adaptive batch processing
func (abp *AdaptiveBatchProcessor[T]) Submit(ctx context.Context, item T) error {
	select {
	case abp.inputCh <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-abp.done:
		return gerror.New(gerror.ErrCodeInternal, "adaptive batch processor stopped", nil)
	}
}

// adaptiveWorker processes batches with adaptive sizing
func (abp *AdaptiveBatchProcessor[T]) adaptiveWorker() {
	defer abp.wg.Done()

	batch := make([]T, 0, abp.maxBatchSize)
	timer := time.NewTimer(100 * time.Millisecond)
	timer.Stop()

	for {
		select {
		case item := <-abp.inputCh:
			batch = append(batch, item)

			// Start timer on first item
			if len(batch) == 1 {
				timer.Reset(100 * time.Millisecond)
			}

			// Process when current batch size reached
			currentSize := int(abp.stats.CurrentBatchSize.Load())
			if len(batch) >= currentSize {
				abp.processBatch(batch)
				batch = make([]T, 0, abp.maxBatchSize)
				timer.Stop()
			}

		case <-timer.C:
			// Process partial batch on timeout
			if len(batch) > 0 {
				abp.processBatch(batch)
				batch = make([]T, 0, abp.maxBatchSize)
			}

		case <-abp.done:
			// Process remaining batch
			if len(batch) > 0 {
				abp.processBatch(batch)
			}
			timer.Stop()
			return
		}
	}
}

// processBatch processes a batch and adjusts batch size based on latency
func (abp *AdaptiveBatchProcessor[T]) processBatch(batch []T) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	abp.processor(ctx, batch)
	cancel()

	latency := time.Since(start)
	abp.recordLatency(latency)
	abp.adjustBatchSize(latency)
}

// recordLatency records processing latency for analysis
func (abp *AdaptiveBatchProcessor[T]) recordLatency(latency time.Duration) {
	abp.mu.Lock()
	defer abp.mu.Unlock()

	abp.recentLatencies = append(abp.recentLatencies, latency)
	if len(abp.recentLatencies) > 10 {
		abp.recentLatencies = abp.recentLatencies[1:]
	}

	// Update average latency
	var total time.Duration
	for _, lat := range abp.recentLatencies {
		total += lat
	}
	avg := total / time.Duration(len(abp.recentLatencies))
	abp.stats.AverageLatency.Store(uint64(avg.Nanoseconds()))
}

// adjustBatchSize adjusts batch size based on latency feedback
func (abp *AdaptiveBatchProcessor[T]) adjustBatchSize(latency time.Duration) {
	current := int(abp.stats.CurrentBatchSize.Load())

	if latency > abp.targetLatency {
		// Latency too high, reduce batch size
		newSize := int(float64(current) * (1.0 - abp.adjustmentRate))
		if newSize < abp.minBatchSize {
			newSize = abp.minBatchSize
		}
		abp.stats.CurrentBatchSize.Store(int32(newSize))
		abp.stats.Adjustments.Add(1)
	} else if latency < abp.targetLatency/2 {
		// Latency low, increase batch size
		newSize := int(float64(current) * (1.0 + abp.adjustmentRate))
		if newSize > abp.maxBatchSize {
			newSize = abp.maxBatchSize
		}
		abp.stats.CurrentBatchSize.Store(int32(newSize))
		abp.stats.Adjustments.Add(1)
	}

	// Update optimal batch size
	abp.stats.OptimalBatchSize.Store(abp.stats.CurrentBatchSize.Load())
}

// Stop stops the adaptive batch processor
func (abp *AdaptiveBatchProcessor[T]) Stop() {
	close(abp.done)
	abp.wg.Wait()
}

// Stats returns current adaptive processing statistics
func (abp *AdaptiveBatchProcessor[T]) Stats() AdaptiveStats {
	return AdaptiveStats{
		CurrentBatchSize: atomic.Int32{},
		AverageLatency:   atomic.Uint64{},
		Adjustments:      atomic.Uint64{},
		OptimalBatchSize: atomic.Int32{},
	}
}

// StreamingBatchProcessor processes streaming data in batches
type StreamingBatchProcessor[T any] struct {
	batchSize int
	processor func(context.Context, []T) error
	inputCh   chan T
	outputCh  chan []T
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewStreamingBatchProcessor creates a new streaming batch processor
func NewStreamingBatchProcessor[T any](batchSize int, processor func(context.Context, []T) error) *StreamingBatchProcessor[T] {
	if batchSize <= 0 {
		batchSize = 100
	}

	sbp := &StreamingBatchProcessor[T]{
		batchSize: batchSize,
		processor: processor,
		inputCh:   make(chan T, batchSize*2),
		outputCh:  make(chan []T, 10),
		done:      make(chan struct{}),
	}

	sbp.wg.Add(1)
	go sbp.streamProcessor()

	return sbp
}

// Input returns the input channel for streaming data
func (sbp *StreamingBatchProcessor[T]) Input() chan<- T {
	return sbp.inputCh
}

// Output returns the output channel for processed batches
func (sbp *StreamingBatchProcessor[T]) Output() <-chan []T {
	return sbp.outputCh
}

// streamProcessor processes streaming input into batches
func (sbp *StreamingBatchProcessor[T]) streamProcessor() {
	defer sbp.wg.Done()
	defer close(sbp.outputCh)

	batch := make([]T, 0, sbp.batchSize)

	for {
		select {
		case item, ok := <-sbp.inputCh:
			if !ok {
				// Input closed, process remaining batch
				if len(batch) > 0 {
					sbp.processBatchAsync(batch)
				}
				return
			}

			batch = append(batch, item)
			if len(batch) >= sbp.batchSize {
				sbp.processBatchAsync(batch)
				batch = make([]T, 0, sbp.batchSize)
			}

		case <-sbp.done:
			// Process remaining batch
			if len(batch) > 0 {
				sbp.processBatchAsync(batch)
			}
			return
		}
	}
}

// processBatchAsync processes a batch asynchronously
func (sbp *StreamingBatchProcessor[T]) processBatchAsync(batch []T) {
	batchCopy := make([]T, len(batch))
	copy(batchCopy, batch)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := sbp.processor(ctx, batchCopy); err == nil {
			select {
			case sbp.outputCh <- batchCopy:
			case <-sbp.done:
			}
		}
	}()
}

// Stop stops the streaming batch processor
func (sbp *StreamingBatchProcessor[T]) Stop() {
	close(sbp.done)
	close(sbp.inputCh)
	sbp.wg.Wait()
}
