package performance

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Pool provides generic object pooling with lifecycle management
type Pool[T any] struct {
	pool     sync.Pool
	factory  func() T
	reset    func(T)
	validate func(T) bool
	stats    PoolStats
	maxAge   time.Duration
	lastGC   atomic.Int64
}

// PoolStats tracks pool utilization and performance
type PoolStats struct {
	Gets      atomic.Uint64
	Puts      atomic.Uint64
	Misses    atomic.Uint64
	Created   atomic.Uint64
	Destroyed atomic.Uint64
	Reused    atomic.Uint64
}

// PoolConfig configures pool behavior
type PoolConfig[T any] struct {
	Factory  func() T
	Reset    func(T)
	Validate func(T) bool
	MaxAge   time.Duration
}

// NewPool creates a new generic object pool
func NewPool[T any](cfg PoolConfig[T]) *Pool[T] {
	if cfg.Factory == nil {
		panic("factory function is required")
	}
	if cfg.Reset == nil {
		cfg.Reset = func(T) {} // No-op reset
	}
	if cfg.Validate == nil {
		cfg.Validate = func(T) bool { return true }
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 10 * time.Minute // Default max age
	}

	p := &Pool[T]{
		factory:  cfg.Factory,
		reset:    cfg.Reset,
		validate: cfg.Validate,
		maxAge:   cfg.MaxAge,
	}
	p.lastGC.Store(time.Now().Unix())

	p.pool.New = func() interface{} {
		p.stats.Created.Add(1)
		return &poolItem[T]{
			obj:     p.factory(),
			created: time.Now(),
		}
	}

	return p
}

// poolItem wraps objects with metadata
type poolItem[T any] struct {
	obj     T
	created time.Time
}

// Get retrieves an object from the pool
func (p *Pool[T]) Get() T {
	p.stats.Gets.Add(1)

	if item := p.pool.Get(); item != nil {
		pooled := item.(*poolItem[T])

		// Check age and validity
		if time.Since(pooled.created) < p.maxAge && p.validate(pooled.obj) {
			p.stats.Reused.Add(1)
			return pooled.obj
		}

		// Object is too old or invalid
		p.stats.Destroyed.Add(1)
	}

	// Pool miss or invalid object
	p.stats.Misses.Add(1)
	return p.factory()
}

// Put returns an object to the pool
func (p *Pool[T]) Put(obj T) {
	p.stats.Puts.Add(1)
	p.reset(obj)

	item := &poolItem[T]{
		obj:     obj,
		created: time.Now(),
	}

	p.pool.Put(item)
	p.maybeGC()
}

// maybeGC performs periodic garbage collection
func (p *Pool[T]) maybeGC() {
	now := time.Now().Unix()
	if now-p.lastGC.Load() > 300 { // 5 minutes
		if p.lastGC.CompareAndSwap(p.lastGC.Load(), now) {
			go p.gc()
		}
	}
}

// gc triggers garbage collection (handled by sync.Pool internally)
func (p *Pool[T]) gc() {
	// sync.Pool handles its own garbage collection
	// This is just a marker for when we last checked
	p.lastGC.Store(time.Now().Unix())
}

// Stats returns current pool statistics
func (p *Pool[T]) Stats() PoolStats {
	return PoolStats{
		Gets:      atomic.Uint64{},
		Puts:      atomic.Uint64{},
		Misses:    atomic.Uint64{},
		Created:   atomic.Uint64{},
		Destroyed: atomic.Uint64{},
		Reused:    atomic.Uint64{},
	}
}

// Common pool instances for frequently used types
var (
	// BufferPool provides reusable byte buffers
	BufferPool = NewPool(PoolConfig[*bytes.Buffer]{
		Factory: func() *bytes.Buffer {
			return new(bytes.Buffer)
		},
		Reset: func(b *bytes.Buffer) {
			b.Reset()
		},
		Validate: func(b *bytes.Buffer) bool {
			return b.Cap() < 64*1024 // Don't keep huge buffers
		},
	})

	// StringBuilderPool provides reusable string builders
	StringBuilderPool = NewPool(PoolConfig[*strings.Builder]{
		Factory: func() *strings.Builder {
			return new(strings.Builder)
		},
		Reset: func(b *strings.Builder) {
			b.Reset()
		},
		Validate: func(b *strings.Builder) bool {
			return b.Cap() < 32*1024 // Don't keep huge builders
		},
	})

	// SlicePool provides reusable byte slices
	SlicePool = NewPool(PoolConfig[[]byte]{
		Factory: func() []byte {
			return make([]byte, 0, 4096)
		},
		Reset: func(s []byte) {
			s = s[:0] // Reset length but keep capacity
		},
		Validate: func(s []byte) bool {
			return cap(s) >= 1024 && cap(s) <= 64*1024
		},
	})
)

// GoroutinePool manages a pool of goroutines for task execution
type GoroutinePool struct {
	workers int
	queue   chan func()
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stats   GoroutinePoolStats
	started atomic.Bool
}

// GoroutinePoolStats tracks goroutine pool performance
type GoroutinePoolStats struct {
	Submitted atomic.Uint64
	Executed  atomic.Uint64
	Panics    atomic.Uint64
	QueueFull atomic.Uint64
}

// NewGoroutinePool creates a new goroutine pool
func NewGoroutinePool(workers int, queueSize int) *GoroutinePool {
	ctx, cancel := context.WithCancel(context.Background())

	return &GoroutinePool{
		workers: workers,
		queue:   make(chan func(), queueSize),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins the goroutine pool workers
func (gp *GoroutinePool) Start() error {
	if !gp.started.CompareAndSwap(false, true) {
		return gerror.New(gerror.ErrCodeInternal, "goroutine pool already started", nil)
	}

	for i := 0; i < gp.workers; i++ {
		gp.wg.Add(1)
		go gp.worker(i)
	}

	return nil
}

// Submit adds a task to the pool
func (gp *GoroutinePool) Submit(task func()) error {
	if !gp.started.Load() {
		return gerror.New(gerror.ErrCodeInternal, "goroutine pool not started", nil)
	}

	gp.stats.Submitted.Add(1)

	select {
	case gp.queue <- task:
		return nil
	case <-gp.ctx.Done():
		return gerror.New(gerror.ErrCodeInternal, "goroutine pool shutting down", nil)
	default:
		gp.stats.QueueFull.Add(1)
		return gerror.New(gerror.ErrCodeInternal, "queue is full", nil)
	}
}

// SubmitWithContext submits a task with context support
func (gp *GoroutinePool) SubmitWithContext(ctx context.Context, task func(context.Context)) error {
	return gp.Submit(func() {
		task(ctx)
	})
}

// worker is the main worker goroutine
func (gp *GoroutinePool) worker(id int) {
	defer gp.wg.Done()

	for {
		select {
		case task := <-gp.queue:
			func() {
				defer func() {
					if r := recover(); r != nil {
						gp.stats.Panics.Add(1)
					}
				}()

				gp.stats.Executed.Add(1)
				task()
			}()

		case <-gp.ctx.Done():
			return
		}
	}
}

// Stop gracefully shuts down the goroutine pool
func (gp *GoroutinePool) Stop() {
	if gp.started.Load() {
		gp.cancel()
		gp.wg.Wait()
		gp.started.Store(false)
	}
}

// Stats returns current goroutine pool statistics
func (gp *GoroutinePool) Stats() GoroutinePoolStats {
	return GoroutinePoolStats{
		Submitted: atomic.Uint64{},
		Executed:  atomic.Uint64{},
		Panics:    atomic.Uint64{},
		QueueFull: atomic.Uint64{},
	}
}

// PoolManager manages multiple pools with lifecycle
type PoolManager struct {
	pools map[string]interface{}
	mu    sync.RWMutex
}

// NewPoolManager creates a new pool manager
func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[string]interface{}),
	}
}

// RegisterPool registers a named pool
func (pm *PoolManager) RegisterPool(name string, pool interface{}) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pools[name] = pool
}

// GetPool retrieves a named pool
func (pm *PoolManager) GetPool(name string) (interface{}, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pool, exists := pm.pools[name]
	return pool, exists
}

// Stats returns statistics for all managed pools
func (pm *PoolManager) Stats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, pool := range pm.pools {
		// Use type assertion to get stats based on pool type
		switch p := pool.(type) {
		case *GoroutinePool:
			stats[name] = p.Stats()
		default:
			stats[name] = "unknown pool type"
		}
	}

	return stats
}
