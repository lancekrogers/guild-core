// Package performance provides high-performance utilities and optimizations for the Guild framework.
//
// This package contains comprehensive performance optimization tools including:
//
// # Object Pooling
//
// Generic object pools with lifecycle management for reducing garbage collection pressure:
//
//	pool := performance.NewPool(performance.PoolConfig[[]byte]{
//		Factory: func() []byte {
//			return make([]byte, 1024)
//		},
//		Reset: func(b []byte) []byte {
//			return b[:0] // Reset slice length
//		},
//	})
//
//	buffer := pool.Get()
//	defer pool.Put(buffer)
//
// # Multi-Level Caching
//
// L1/L2/L3 cache hierarchy with configurable eviction policies:
//
//	cache := performance.NewCache[string, UserData](performance.CacheConfig{
//		L1Size: 1000,   // Fast in-memory cache
//		L2Size: 10000,  // Larger memory cache
//		TTL:    time.Hour,
//	})
//
//	cache.Set("user:123", userData)
//	if data, found := cache.Get("user:123", nil); found {
//		// Cache hit
//	}
//
// # Worker Pools
//
// High-performance worker pools with work-stealing queues:
//
//	pool := performance.NewWorkerPool(performance.WorkerPoolConfig{
//		Workers:   runtime.NumCPU(),
//		QueueSize: 1000,
//	})
//	defer pool.Stop()
//
//	pool.Start()
//	pool.Submit(func() error {
//		// Your work here
//		return nil
//	})
//
// # Memory Optimization
//
// Arena allocators and buffer pools for efficient memory management:
//
//	arena := performance.NewArena(performance.ArenaConfig{
//		BlockSize:     64 * 1024,
//		InitialBlocks: 4,
//	})
//
//	data, err := arena.Alloc(1024)
//	if err != nil {
//		// Handle allocation error
//	}
//
// # I/O Optimization
//
// Zero-copy file operations and high-performance buffered I/O:
//
//	// Zero-copy file reading
//	zcf, err := performance.OpenZeroCopy("large-file.bin", nil)
//	if err != nil {
//		return err
//	}
//	defer zcf.Close()
//
//	data, err := zcf.Slice(0, 1024) // No copying
//
//	// High-performance buffered writing
//	writer := performance.NewBufferedWriter(file, performance.BufferedWriterConfig{
//		BufferSize: 64 * 1024,
//		AutoFlush:  true,
//	})
//
// # CPU Optimization
//
// Parallel processing utilities and CPU affinity management:
//
//	processor := performance.NewParallelProcessor(performance.ParallelConfig{
//		Workers:     runtime.NumCPU(),
//		Concurrency: runtime.NumCPU(),
//	})
//
//	results, err := processor.Map(ctx, data, func(ctx context.Context, item int) (int, error) {
//		return item * 2, nil
//	})
//
// # Batch Processing
//
// Efficient batch processing with adaptive sizing:
//
//	batchProcessor := performance.NewBatchProcessor[DataItem](
//		performance.BatchConfig{
//			BatchSize: 100,
//			MaxWait:   100 * time.Millisecond,
//			Workers:   4,
//		},
//		func(ctx context.Context, batch []DataItem) error {
//			// Process batch
//			return nil
//		},
//	)
//
// # Performance Profiling
//
// Comprehensive profiling and benchmarking framework:
//
//	profiler := performance.NewProfiler()
//	profiler.Enable()
//
//	ctx := profiler.StartProfile("operation", nil)
//	// Your operation
//	profile := ctx.End()
//
//	// Benchmark with regression detection
//	benchmark := profiler.Benchmark("test", func() {
//		// Code to benchmark
//	})
//
// # Performance Testing
//
// Automated performance testing with regression detection:
//
//	suite := performance.NewPerformanceTestSuite()
//	suite.AddTest(performance.TestCase{
//		Name: "critical_operation",
//		Test: func() error {
//			// Performance-critical code
//			return nil
//		},
//		Timeout: time.Second,
//	})
//
//	results := suite.RunAllTests()
//
// # Best Practices
//
// 1. Use object pools for frequently allocated objects
// 2. Implement caching for expensive computations
// 3. Leverage worker pools for concurrent processing
// 4. Use arena allocators for bulk allocations
// 5. Profile critical code paths regularly
// 6. Monitor for performance regressions
//
// # Performance Considerations
//
// - Pool usage reduces GC pressure but increases memory usage
// - Cache hit rates should be monitored and tuned
// - Worker pool sizing should match workload characteristics
// - Arena allocators are fastest but require manual lifecycle management
// - Always benchmark changes to verify improvements
//
// # Thread Safety
//
// All components in this package are designed to be thread-safe and can be used
// concurrently from multiple goroutines unless otherwise specified.
//
// # Memory Management
//
// The package provides tools to minimize allocations and GC pressure, but proper
// lifecycle management is crucial. Always release resources when no longer needed:
//
//	defer pool.Put(item)
//	defer arena.Reset()
//	defer worker.Stop()
package performance
