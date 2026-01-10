package memory

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Arena provides fast memory allocation using pre-allocated blocks
type Arena struct {
	blocks      [][]byte
	current     int
	offset      int
	blockSize   int
	totalSize   atomic.Uint64
	allocations atomic.Uint64
	mu          sync.Mutex
}

// ArenaConfig configures arena behavior
type ArenaConfig struct {
	BlockSize     int
	InitialBlocks int
	MaxBlocks     int
}

// NewArena creates a new memory arena
func NewArena(cfg ArenaConfig) *Arena {
	if cfg.BlockSize <= 0 {
		cfg.BlockSize = 64 * 1024 // 64KB default
	}
	if cfg.InitialBlocks <= 0 {
		cfg.InitialBlocks = 1
	}
	if cfg.MaxBlocks <= 0 {
		cfg.MaxBlocks = 1024
	}

	arena := &Arena{
		blocks:    make([][]byte, 0, cfg.InitialBlocks),
		blockSize: cfg.BlockSize,
	}

	// Pre-allocate initial blocks
	for i := 0; i < cfg.InitialBlocks; i++ {
		arena.addBlock()
	}

	return arena
}

// Alloc allocates memory from the arena
func (a *Arena) Alloc(size int) ([]byte, error) {
	if size <= 0 {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid allocation size", nil)
	}

	// For very large allocations, allocate directly
	if size > a.blockSize/2 {
		a.allocations.Add(1)
		a.totalSize.Add(uint64(size))
		return make([]byte, size), nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if current block has enough space
	if a.current >= len(a.blocks) || a.offset+size > a.blockSize {
		if err := a.addBlock(); err != nil {
			return nil, err
		}
	}

	// Allocate from current block
	block := a.blocks[a.current]
	result := block[a.offset : a.offset+size : a.offset+size]
	a.offset += size

	a.allocations.Add(1)
	a.totalSize.Add(uint64(size))

	return result, nil
}

// AllocAligned allocates aligned memory from the arena
func (a *Arena) AllocAligned(size, alignment int) ([]byte, error) {
	if size <= 0 {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid allocation size", nil)
	}
	if alignment <= 0 || (alignment&(alignment-1)) != 0 {
		return nil, gerror.New(gerror.ErrCodeInternal, "alignment must be a power of 2", nil)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Calculate aligned offset
	alignedOffset := (a.offset + alignment - 1) &^ (alignment - 1)

	// Check if current block has enough space
	if a.current >= len(a.blocks) || alignedOffset+size > a.blockSize {
		if err := a.addBlock(); err != nil {
			return nil, err
		}
		alignedOffset = 0 // New block starts aligned
	}

	// Allocate from current block
	block := a.blocks[a.current]
	result := block[alignedOffset : alignedOffset+size : alignedOffset+size]
	a.offset = alignedOffset + size

	a.allocations.Add(1)
	a.totalSize.Add(uint64(size))

	return result, nil
}

// AllocString allocates a string from the arena
func (a *Arena) AllocString(s string) (string, error) {
	if len(s) == 0 {
		return "", nil
	}

	data, err := a.Alloc(len(s))
	if err != nil {
		return "", err
	}

	copy(data, s)
	return *(*string)(unsafe.Pointer(&data)), nil
}

// AllocBytes allocates and copies a byte slice
func (a *Arena) AllocBytes(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}

	data, err := a.Alloc(len(b))
	if err != nil {
		return nil, err
	}

	copy(data, b)
	return data, nil
}

// addBlock adds a new block to the arena
func (a *Arena) addBlock() error {
	// Create new block
	block := make([]byte, a.blockSize)
	a.blocks = append(a.blocks, block)
	a.current = len(a.blocks) - 1
	a.offset = 0

	return nil
}

// Reset resets the arena for reuse
func (a *Arena) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.current = 0
	a.offset = 0
	// Keep allocated blocks for reuse
}

// Stats returns arena statistics
func (a *Arena) Stats() ArenaStats {
	a.mu.Lock()
	defer a.mu.Unlock()

	return ArenaStats{
		Blocks:      len(a.blocks),
		BlockSize:   a.blockSize,
		TotalSize:   a.totalSize.Load(),
		Allocations: a.allocations.Load(),
		UsedBytes:   uint64(a.current*a.blockSize + a.offset),
	}
}

// ArenaStats provides arena statistics
type ArenaStats struct {
	Blocks      int
	BlockSize   int
	TotalSize   uint64
	Allocations uint64
	UsedBytes   uint64
}

// Free releases all arena memory (unsafe - invalidates all allocations)
func (a *Arena) Free() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.blocks = nil
	a.current = 0
	a.offset = 0
	runtime.GC() // Encourage GC to clean up
}

// ThreadSafeArena provides a thread-safe arena with per-thread allocation
type ThreadSafeArena struct {
	arenas sync.Map // map[int]*Arena - keyed by goroutine ID
	config ArenaConfig
}

// NewThreadSafeArena creates a new thread-safe arena
func NewThreadSafeArena(cfg ArenaConfig) *ThreadSafeArena {
	return &ThreadSafeArena{
		config: cfg,
	}
}

// Alloc allocates memory from a per-thread arena
func (tsa *ThreadSafeArena) Alloc(size int) ([]byte, error) {
	arena := tsa.getOrCreateArena()
	return arena.Alloc(size)
}

// AllocAligned allocates aligned memory from a per-thread arena
func (tsa *ThreadSafeArena) AllocAligned(size, alignment int) ([]byte, error) {
	arena := tsa.getOrCreateArena()
	return arena.AllocAligned(size, alignment)
}

// getOrCreateArena gets or creates an arena for the current goroutine
func (tsa *ThreadSafeArena) getOrCreateArena() *Arena {
	gid := getGoroutineID()

	if arena, ok := tsa.arenas.Load(gid); ok {
		return arena.(*Arena)
	}

	arena := NewArena(tsa.config)
	actual, _ := tsa.arenas.LoadOrStore(gid, arena)
	return actual.(*Arena)
}

// Reset resets all per-thread arenas
func (tsa *ThreadSafeArena) Reset() {
	tsa.arenas.Range(func(key, value interface{}) bool {
		arena := value.(*Arena)
		arena.Reset()
		return true
	})
}

// GlobalStats returns combined statistics from all arenas
func (tsa *ThreadSafeArena) GlobalStats() ThreadSafeArenaStats {
	var stats ThreadSafeArenaStats

	tsa.arenas.Range(func(key, value interface{}) bool {
		arena := value.(*Arena)
		arenaStats := arena.Stats()

		stats.TotalArenas++
		stats.TotalBlocks += arenaStats.Blocks
		stats.TotalSize += arenaStats.TotalSize
		stats.TotalAllocations += arenaStats.Allocations
		stats.TotalUsedBytes += arenaStats.UsedBytes

		return true
	})

	return stats
}

// ThreadSafeArenaStats provides combined arena statistics
type ThreadSafeArenaStats struct {
	TotalArenas      int
	TotalBlocks      int
	TotalSize        uint64
	TotalAllocations uint64
	TotalUsedBytes   uint64
}

// getGoroutineID returns the current goroutine ID (hack for per-goroutine arenas)
func getGoroutineID() int {
	// This is a simplified version - in production you might use a more robust method
	// or simply use a thread-local storage approach
	return int(uintptr(unsafe.Pointer(&struct{}{}))) % 10000
}

// Pool provides memory pools with different size classes
type Pool struct {
	pools   []*ArenaPool
	maxSize int
}

// ArenaPool represents a pool for a specific size class
type ArenaPool struct {
	size  int
	arena *Arena
	free  [][]byte
	mu    sync.Mutex
}

// NewPool creates a new memory pool with size classes
func NewPool(maxSize int, sizeClasses []int) *Pool {
	if maxSize <= 0 {
		maxSize = 64 * 1024
	}

	pools := make([]*ArenaPool, len(sizeClasses))
	for i, size := range sizeClasses {
		pools[i] = &ArenaPool{
			size: size,
			arena: NewArena(ArenaConfig{
				BlockSize:     maxSize,
				InitialBlocks: 1,
			}),
			free: make([][]byte, 0, 100),
		}
	}

	return &Pool{
		pools:   pools,
		maxSize: maxSize,
	}
}

// Alloc allocates memory from the appropriate size class
func (p *Pool) Alloc(size int) ([]byte, error) {
	// Find appropriate size class
	pool := p.findPool(size)
	if pool == nil {
		// Size too large, allocate directly
		return make([]byte, size), nil
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Try to reuse from free list
	if len(pool.free) > 0 {
		buf := pool.free[len(pool.free)-1]
		pool.free = pool.free[:len(pool.free)-1]
		return buf[:size], nil
	}

	// Allocate new from arena
	return pool.arena.Alloc(pool.size)
}

// Free returns memory to the pool
func (p *Pool) Free(buf []byte) {
	if len(buf) == 0 {
		return
	}

	size := cap(buf)
	pool := p.findPool(size)
	if pool == nil {
		// Size too large, let GC handle it
		return
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Add to free list if not too many
	if len(pool.free) < 100 {
		pool.free = append(pool.free, buf[:cap(buf)])
	}
}

// findPool finds the appropriate pool for a given size
func (p *Pool) findPool(size int) *ArenaPool {
	for _, pool := range p.pools {
		if size <= pool.size {
			return pool
		}
	}
	return nil
}

// StackAllocator provides stack-based memory allocation
type StackAllocator struct {
	data   []byte
	offset int
	marks  []int
	mu     sync.Mutex
}

// NewStackAllocator creates a new stack allocator
func NewStackAllocator(size int) *StackAllocator {
	return &StackAllocator{
		data:  make([]byte, size),
		marks: make([]int, 0, 32),
	}
}

// Alloc allocates memory from the stack
func (sa *StackAllocator) Alloc(size int) ([]byte, error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.offset+size > len(sa.data) {
		return nil, gerror.New(gerror.ErrCodeInternal, "stack allocator out of memory", nil)
	}

	result := sa.data[sa.offset : sa.offset+size]
	sa.offset += size
	return result, nil
}

// Mark creates a mark point for later restoration
func (sa *StackAllocator) Mark() {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	sa.marks = append(sa.marks, sa.offset)
}

// Restore restores to the last mark point
func (sa *StackAllocator) Restore() error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if len(sa.marks) == 0 {
		return gerror.New(gerror.ErrCodeInternal, "no marks to restore", nil)
	}

	sa.offset = sa.marks[len(sa.marks)-1]
	sa.marks = sa.marks[:len(sa.marks)-1]
	return nil
}

// Reset resets the stack allocator
func (sa *StackAllocator) Reset() {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	sa.offset = 0
	sa.marks = sa.marks[:0]
}
