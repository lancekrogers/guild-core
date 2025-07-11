package memory

import (
	"sync"
	"sync/atomic"
)

// BufferPool provides efficient buffer pooling with size classes
type BufferPool struct {
	pools     []*sizedPool
	maxSize   int
	hitCount  atomic.Uint64
	missCount atomic.Uint64
}

// sizedPool represents a pool for buffers of a specific size
type sizedPool struct {
	size int
	pool sync.Pool
}

// BufferPoolConfig configures buffer pool behavior
type BufferPoolConfig struct {
	MinSize   int
	MaxSize   int
	SizeClass int
}

// NewBufferPool creates a new buffer pool with size classes
func NewBufferPool(cfg BufferPoolConfig) *BufferPool {
	if cfg.MinSize <= 0 {
		cfg.MinSize = 64
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 64 * 1024
	}
	if cfg.SizeClass <= 0 {
		cfg.SizeClass = 2 // Double each size class
	}

	var pools []*sizedPool
	for size := cfg.MinSize; size <= cfg.MaxSize; size *= cfg.SizeClass {
		pools = append(pools, &sizedPool{
			size: size,
			pool: sync.Pool{
				New: func(s int) func() interface{} {
					return func() interface{} {
						return make([]byte, 0, s)
					}
				}(size),
			},
		})
	}

	return &BufferPool{
		pools:   pools,
		maxSize: cfg.MaxSize,
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get(size int) []byte {
	pool := bp.findPool(size)
	if pool == nil {
		bp.missCount.Add(1)
		return make([]byte, 0, size)
	}

	bp.hitCount.Add(1)
	buf := pool.pool.Get().([]byte)
	return buf[:0] // Reset length but keep capacity
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if cap(buf) > bp.maxSize {
		return // Don't pool oversized buffers
	}

	pool := bp.findPool(cap(buf))
	if pool != nil {
		pool.pool.Put(buf)
	}
}

// findPool finds the appropriate pool for a given size
func (bp *BufferPool) findPool(size int) *sizedPool {
	for _, pool := range bp.pools {
		if size <= pool.size {
			return pool
		}
	}
	return nil
}

// Stats returns buffer pool statistics
func (bp *BufferPool) Stats() BufferPoolStats {
	return BufferPoolStats{
		Hits:    bp.hitCount.Load(),
		Misses:  bp.missCount.Load(),
		Pools:   len(bp.pools),
		MaxSize: bp.maxSize,
	}
}

// BufferPoolStats provides buffer pool statistics
type BufferPoolStats struct {
	Hits    uint64
	Misses  uint64
	Pools   int
	MaxSize int
}

// HitRate returns the cache hit rate
func (bps *BufferPoolStats) HitRate() float64 {
	total := bps.Hits + bps.Misses
	if total == 0 {
		return 0
	}
	return float64(bps.Hits) / float64(total)
}

// RingBuffer provides a circular buffer for streaming data
type RingBuffer struct {
	data []byte
	head int
	tail int
	size int
	mu   sync.RWMutex
	full bool
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write writes data to the ring buffer
func (rb *RingBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	written := 0
	for i := 0; i < len(p); i++ {
		if rb.full && rb.head == rb.tail {
			// Buffer full, overwrite oldest data
			rb.tail = (rb.tail + 1) % rb.size
		}

		rb.data[rb.head] = p[i]
		rb.head = (rb.head + 1) % rb.size
		written++

		if rb.head == rb.tail {
			rb.full = true
		}
	}

	return written, nil
}

// Read reads data from the ring buffer
func (rb *RingBuffer) Read(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.isEmpty() {
		return 0, nil
	}

	read := 0
	for i := 0; i < len(p) && !rb.isEmpty(); i++ {
		p[i] = rb.data[rb.tail]
		rb.tail = (rb.tail + 1) % rb.size
		rb.full = false
		read++
	}

	return read, nil
}

// Available returns the number of bytes available for reading
func (rb *RingBuffer) Available() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.full {
		return rb.size
	}

	if rb.head >= rb.tail {
		return rb.head - rb.tail
	}

	return rb.size - rb.tail + rb.head
}

// Free returns the number of bytes available for writing
func (rb *RingBuffer) Free() int {
	return rb.size - rb.Available()
}

// isEmpty checks if the buffer is empty
func (rb *RingBuffer) isEmpty() bool {
	return !rb.full && rb.head == rb.tail
}

// Reset resets the ring buffer
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.full = false
}

// ByteBuffer provides a growable byte buffer with pooling
type ByteBuffer struct {
	data []byte
	pool *BufferPool
}

// NewByteBuffer creates a new byte buffer
func NewByteBuffer(pool *BufferPool, initialSize int) *ByteBuffer {
	if pool == nil {
		return &ByteBuffer{
			data: make([]byte, 0, initialSize),
		}
	}

	return &ByteBuffer{
		data: pool.Get(initialSize),
		pool: pool,
	}
}

// Write appends data to the buffer
func (bb *ByteBuffer) Write(p []byte) (int, error) {
	bb.data = append(bb.data, p...)
	return len(p), nil
}

// WriteByte appends a single byte
func (bb *ByteBuffer) WriteByte(b byte) error {
	bb.data = append(bb.data, b)
	return nil
}

// WriteString appends a string
func (bb *ByteBuffer) WriteString(s string) (int, error) {
	bb.data = append(bb.data, s...)
	return len(s), nil
}

// Bytes returns the buffer contents
func (bb *ByteBuffer) Bytes() []byte {
	return bb.data
}

// String returns the buffer contents as a string
func (bb *ByteBuffer) String() string {
	return string(bb.data)
}

// Len returns the buffer length
func (bb *ByteBuffer) Len() int {
	return len(bb.data)
}

// Cap returns the buffer capacity
func (bb *ByteBuffer) Cap() int {
	return cap(bb.data)
}

// Reset clears the buffer
func (bb *ByteBuffer) Reset() {
	bb.data = bb.data[:0]
}

// Grow ensures the buffer has at least n more bytes of capacity
func (bb *ByteBuffer) Grow(n int) {
	if cap(bb.data)-len(bb.data) < n {
		newSize := len(bb.data) + n
		if newSize < cap(bb.data)*2 {
			newSize = cap(bb.data) * 2
		}

		newData := make([]byte, len(bb.data), newSize)
		copy(newData, bb.data)
		bb.data = newData
	}
}

// Release returns the buffer to the pool
func (bb *ByteBuffer) Release() {
	if bb.pool != nil {
		bb.pool.Put(bb.data)
	}
	bb.data = nil
}

// MemoryMapper provides memory mapping utilities for large buffers
type MemoryMapper struct {
	mappings map[string][]byte
	mu       sync.RWMutex
}

// NewMemoryMapper creates a new memory mapper
func NewMemoryMapper() *MemoryMapper {
	return &MemoryMapper{
		mappings: make(map[string][]byte),
	}
}

// MapMemory maps memory with a given key
func (mm *MemoryMapper) MapMemory(key string, size int) ([]byte, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if existing, exists := mm.mappings[key]; exists {
		return existing, nil
	}

	data := make([]byte, size)
	mm.mappings[key] = data
	return data, nil
}

// GetMapping retrieves an existing mapping
func (mm *MemoryMapper) GetMapping(key string) ([]byte, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	data, exists := mm.mappings[key]
	return data, exists
}

// UnmapMemory removes a mapping
func (mm *MemoryMapper) UnmapMemory(key string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	delete(mm.mappings, key)
}

// ListMappings returns all mapping keys
func (mm *MemoryMapper) ListMappings() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	keys := make([]string, 0, len(mm.mappings))
	for key := range mm.mappings {
		keys = append(keys, key)
	}
	return keys
}

// TotalSize returns the total size of all mappings
func (mm *MemoryMapper) TotalSize() int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	total := 0
	for _, data := range mm.mappings {
		total += len(data)
	}
	return total
}
