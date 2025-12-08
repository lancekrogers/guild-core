package performance

import (
	"bufio"
	"context"
	"io"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// BufferedWriter provides high-performance buffered writing with batching
type BufferedWriter struct {
	w         io.Writer
	buf       []byte
	n         int
	batchSize int
	flushCh   chan struct{}
	done      chan struct{}
	mu        sync.Mutex
	autoFlush bool
	interval  time.Duration
}

// BufferedWriterConfig configures buffered writer behavior
type BufferedWriterConfig struct {
	BufferSize int
	BatchSize  int
	AutoFlush  bool
	Interval   time.Duration
}

// NewBufferedWriter creates a new high-performance buffered writer
func NewBufferedWriter(w io.Writer, cfg BufferedWriterConfig) *BufferedWriter {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 64 * 1024 // 64KB default
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = cfg.BufferSize
	}
	if cfg.Interval == 0 {
		cfg.Interval = 100 * time.Millisecond
	}

	bw := &BufferedWriter{
		w:         w,
		buf:       make([]byte, cfg.BufferSize),
		batchSize: cfg.BatchSize,
		flushCh:   make(chan struct{}, 1),
		done:      make(chan struct{}),
		autoFlush: cfg.AutoFlush,
		interval:  cfg.Interval,
	}

	if cfg.AutoFlush {
		go bw.flushLoop()
	}

	return bw
}

// Write writes data to the buffer
func (bw *BufferedWriter) Write(p []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if len(p) > len(bw.buf)-bw.n {
		// Flush if buffer would overflow
		if err := bw.flushLocked(); err != nil {
			return 0, err
		}
	}

	// If data is larger than buffer, write directly
	if len(p) > len(bw.buf) {
		return bw.w.Write(p)
	}

	// Copy to buffer
	n := copy(bw.buf[bw.n:], p)
	bw.n += n

	// Trigger flush if batch size reached
	if bw.n >= bw.batchSize {
		bw.triggerFlush()
	}

	return n, nil
}

// Flush manually flushes the buffer
func (bw *BufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.flushLocked()
}

// flushLocked flushes the buffer while holding the lock
func (bw *BufferedWriter) flushLocked() error {
	if bw.n == 0 {
		return nil
	}

	_, err := bw.w.Write(bw.buf[:bw.n])
	bw.n = 0
	return err
}

// triggerFlush triggers an async flush
func (bw *BufferedWriter) triggerFlush() {
	select {
	case bw.flushCh <- struct{}{}:
	default:
		// Channel full, flush already pending
	}
}

// flushLoop runs the auto-flush loop
func (bw *BufferedWriter) flushLoop() {
	ticker := time.NewTicker(bw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-bw.flushCh:
			bw.Flush()
		case <-ticker.C:
			bw.Flush()
		case <-bw.done:
			return
		}
	}
}

// Close closes the buffered writer
func (bw *BufferedWriter) Close() error {
	close(bw.done)
	return bw.Flush()
}

// ZeroCopyFile provides zero-copy file operations using memory mapping
type ZeroCopyFile struct {
	fd     int
	data   []byte
	size   int64
	offset int64
}

// OpenZeroCopy opens a file for zero-copy operations
func OpenZeroCopy(path string) (*ZeroCopyFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to open file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stat file")
	}

	fd := int(file.Fd())
	size := stat.Size()

	if size == 0 {
		return &ZeroCopyFile{fd: fd, size: size}, nil
	}

	// Memory map the file
	data, err := syscall.Mmap(fd, 0, int(size), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to mmap file")
	}

	return &ZeroCopyFile{
		fd:   fd,
		data: data,
		size: size,
	}, nil
}

// Read reads data from the memory-mapped file
func (zcf *ZeroCopyFile) Read(p []byte) (int, error) {
	if zcf.offset >= zcf.size {
		return 0, io.EOF
	}

	remaining := zcf.size - zcf.offset
	toRead := int64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	copy(p, zcf.data[zcf.offset:zcf.offset+toRead])
	zcf.offset += toRead

	return int(toRead), nil
}

// ReadAt reads data at a specific offset
func (zcf *ZeroCopyFile) ReadAt(p []byte, off int64) (int, error) {
	if off >= zcf.size || off < 0 {
		return 0, io.EOF
	}

	remaining := zcf.size - off
	toRead := int64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	copy(p, zcf.data[off:off+toRead])
	return int(toRead), nil
}

// Slice returns a slice of the mapped data without copying
func (zcf *ZeroCopyFile) Slice(offset, length int64) ([]byte, error) {
	if offset < 0 || offset >= zcf.size {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid offset", nil)
	}
	if offset+length > zcf.size {
		length = zcf.size - offset
	}

	return zcf.data[offset : offset+length], nil
}

// Close closes the zero-copy file
func (zcf *ZeroCopyFile) Close() error {
	if zcf.data != nil {
		if err := syscall.Munmap(zcf.data); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmap file")
		}
		zcf.data = nil
	}
	return nil
}

// Size returns the file size
func (zcf *ZeroCopyFile) Size() int64 {
	return zcf.size
}

// ParallelProcessor processes files in parallel with optimal resource usage
type ParallelProcessor struct {
	workers   int
	batchSize int
	pool      *WorkerPool
}

// NewParallelProcessor creates a new parallel file processor
func NewParallelProcessor(workers int, batchSize int) *ParallelProcessor {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if batchSize <= 0 {
		batchSize = 10
	}

	pool := NewWorkerPool(WorkerPoolConfig{
		Workers:   workers,
		QueueSize: workers * 2,
	})
	pool.Start()

	return &ParallelProcessor{
		workers:   workers,
		batchSize: batchSize,
		pool:      pool,
	}
}

// ProcessFiles processes multiple files in parallel
func (pp *ParallelProcessor) ProcessFiles(ctx context.Context, files []string, processor func(string, []byte) error) error {
	// Process files in batches to control memory usage
	for i := 0; i < len(files); i += pp.batchSize {
		end := i + pp.batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		if err := pp.processBatch(ctx, batch, processor); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to process batch")
		}
	}

	return nil
}

// processBatch processes a batch of files
func (pp *ParallelProcessor) processBatch(ctx context.Context, files []string, processor func(string, []byte) error) error {
	errCh := make(chan error, len(files))

	for _, file := range files {
		file := file // Capture loop variable

		pp.pool.SubmitWithContext(ctx, func(ctx context.Context) error {
			err := pp.processFile(ctx, file, processor)
			errCh <- err
			return err
		})
	}

	// Wait for all files to be processed
	var firstErr error
	for i := 0; i < len(files); i++ {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// processFile processes a single file
func (pp *ParallelProcessor) processFile(ctx context.Context, file string, processor func(string, []byte) error) error {
	// Try zero-copy first for large files
	stat, err := os.Stat(file)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stat file")
	}

	if stat.Size() > 1024*1024 { // Use zero-copy for files > 1MB
		return pp.processFileZeroCopy(ctx, file, processor)
	}

	// Use regular file reading for smaller files
	data, err := os.ReadFile(file)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read file")
	}

	return processor(file, data)
}

// processFileZeroCopy processes a file using zero-copy operations
func (pp *ParallelProcessor) processFileZeroCopy(ctx context.Context, file string, processor func(string, []byte) error) error {
	zcf, err := OpenZeroCopy(file)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to open zero-copy file")
	}
	defer zcf.Close()

	data, err := zcf.Slice(0, zcf.Size())
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get file slice")
	}

	return processor(file, data)
}

// Stop stops the parallel processor
func (pp *ParallelProcessor) Stop() error {
	return pp.pool.Stop()
}

// AsyncWriter provides asynchronous writing with batching
type AsyncWriter struct {
	w       io.Writer
	ch      chan []byte
	done    chan struct{}
	wg      sync.WaitGroup
	bufPool *Pool[[]byte]
}

// NewAsyncWriter creates a new asynchronous writer
func NewAsyncWriter(w io.Writer, bufferSize int) *AsyncWriter {
	aw := &AsyncWriter{
		w:    w,
		ch:   make(chan []byte, bufferSize),
		done: make(chan struct{}),
		bufPool: NewPool(PoolConfig[[]byte]{
			Factory: func() []byte {
				return make([]byte, 0, 4096)
			},
			Reset: func(b []byte) {
				b = b[:0]
			},
		}),
	}

	aw.wg.Add(1)
	go aw.writeLoop()

	return aw
}

// Write queues data for asynchronous writing
func (aw *AsyncWriter) Write(p []byte) (int, error) {
	// Get buffer from pool and copy data
	buf := aw.bufPool.Get()
	buf = append(buf, p...)

	select {
	case aw.ch <- buf:
		return len(p), nil
	case <-aw.done:
		aw.bufPool.Put(buf)
		return 0, gerror.New(gerror.ErrCodeInternal, "async writer closed", nil)
	default:
		aw.bufPool.Put(buf)
		return 0, gerror.New(gerror.ErrCodeInternal, "write queue full", nil)
	}
}

// writeLoop runs the asynchronous write loop
func (aw *AsyncWriter) writeLoop() {
	defer aw.wg.Done()

	for {
		select {
		case data := <-aw.ch:
			aw.w.Write(data)
			aw.bufPool.Put(data)
		case <-aw.done:
			// Drain remaining writes
			for {
				select {
				case data := <-aw.ch:
					aw.w.Write(data)
					aw.bufPool.Put(data)
				default:
					return
				}
			}
		}
	}
}

// Close closes the async writer
func (aw *AsyncWriter) Close() error {
	close(aw.done)
	aw.wg.Wait()
	return nil
}

// StreamProcessor processes data streams efficiently
type StreamProcessor struct {
	bufferSize int
	readers    []io.Reader
	processor  func([]byte) error
	pool       *Pool[[]byte]
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(bufferSize int, processor func([]byte) error) *StreamProcessor {
	if bufferSize <= 0 {
		bufferSize = 32 * 1024
	}

	return &StreamProcessor{
		bufferSize: bufferSize,
		processor:  processor,
		pool: NewPool(PoolConfig[[]byte]{
			Factory: func() []byte {
				return make([]byte, bufferSize)
			},
			Reset: func(b []byte) {
				b = b[:cap(b)]
			},
		}),
	}
}

// ProcessReader processes data from a reader
func (sp *StreamProcessor) ProcessReader(ctx context.Context, r io.Reader) error {
	reader := bufio.NewReaderSize(r, sp.bufferSize)
	buf := sp.pool.Get()
	defer sp.pool.Put(buf)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 {
			if err := sp.processor(buf[:n]); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "processor failed")
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "read failed")
		}
	}

	return nil
}

// ProcessReaders processes multiple readers concurrently
func (sp *StreamProcessor) ProcessReaders(ctx context.Context, readers []io.Reader) error {
	wp := NewWorkerPool(WorkerPoolConfig{
		Workers:   len(readers),
		QueueSize: len(readers),
	})
	defer wp.Stop()

	wp.Start()

	for _, reader := range readers {
		reader := reader // Capture loop variable
		wp.SubmitWithContext(ctx, func(ctx context.Context) error {
			return sp.ProcessReader(ctx, reader)
		})
	}

	wp.WaitForCompletion()
	return nil
}

// MemoryMappedIO provides memory-mapped I/O utilities
type MemoryMappedIO struct {
	pageSize int
}

// NewMemoryMappedIO creates a new memory-mapped I/O utility
func NewMemoryMappedIO() *MemoryMappedIO {
	return &MemoryMappedIO{
		pageSize: os.Getpagesize(),
	}
}

// MapFile maps a file into memory
func (mmio *MemoryMappedIO) MapFile(path string, writable bool) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to open file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stat file")
	}

	size := stat.Size()
	if size == 0 {
		return []byte{}, nil
	}

	prot := syscall.PROT_READ
	if writable {
		prot |= syscall.PROT_WRITE
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), prot, syscall.MAP_SHARED)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to mmap file")
	}

	return data, nil
}

// UnmapMemory unmaps memory
func (mmio *MemoryMappedIO) UnmapMemory(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if err := syscall.Munmap(data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmap memory")
	}

	return nil
}

// AdviseSequential advises the kernel that memory will be accessed sequentially
func (mmio *MemoryMappedIO) AdviseSequential(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Platform-specific optimization hints could be added here
	// For now, this is a no-op
	return nil
}

// AdviseRandom advises the kernel that memory will be accessed randomly
func (mmio *MemoryMappedIO) AdviseRandom(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Platform-specific optimization hints could be added here
	// For now, this is a no-op
	return nil
}
