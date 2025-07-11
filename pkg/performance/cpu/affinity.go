package cpu

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// AffinityManager manages CPU affinity for optimal performance
type AffinityManager struct {
	cpuCount    int
	assignments map[int]int // goroutine ID -> CPU core
	cpuUsage    []atomic.Int32
	nextCPU     atomic.Int32
	mu          sync.RWMutex
	strategy    AffinityStrategy
	bindEnabled bool
}

// AffinityStrategy defines how CPU affinity is assigned
type AffinityStrategy int

const (
	StrategyRoundRobin AffinityStrategy = iota
	StrategyLeastUsed
	StrategyNUMA
	StrategyDedicated
)

// AffinityConfig configures CPU affinity management
type AffinityConfig struct {
	Strategy       AffinityStrategy
	BindEnabled    bool
	DedicatedCores []int
}

// NewAffinityManager creates a new CPU affinity manager
func NewAffinityManager(cfg AffinityConfig) *AffinityManager {
	cpuCount := runtime.NumCPU()

	am := &AffinityManager{
		cpuCount:    cpuCount,
		assignments: make(map[int]int),
		cpuUsage:    make([]atomic.Int32, cpuCount),
		strategy:    cfg.Strategy,
		bindEnabled: cfg.BindEnabled,
	}

	return am
}

// AssignCPU assigns a CPU core to the current goroutine
func (am *AffinityManager) AssignCPU() (int, error) {
	if !am.bindEnabled {
		return -1, nil // Affinity disabled
	}

	gid := getGoroutineID()

	am.mu.Lock()
	defer am.mu.Unlock()

	// Check if already assigned
	if cpu, exists := am.assignments[gid]; exists {
		return cpu, nil
	}

	// Assign new CPU based on strategy
	cpu := am.selectCPU()
	am.assignments[gid] = cpu
	am.cpuUsage[cpu].Add(1)

	return cpu, nil
}

// ReleaseCPU releases CPU assignment for the current goroutine
func (am *AffinityManager) ReleaseCPU() error {
	if !am.bindEnabled {
		return nil
	}

	gid := getGoroutineID()

	am.mu.Lock()
	defer am.mu.Unlock()

	if cpu, exists := am.assignments[gid]; exists {
		am.cpuUsage[cpu].Add(-1)
		delete(am.assignments, gid)
	}

	return nil
}

// selectCPU selects the best CPU core based on strategy
func (am *AffinityManager) selectCPU() int {
	switch am.strategy {
	case StrategyRoundRobin:
		return am.selectRoundRobin()
	case StrategyLeastUsed:
		return am.selectLeastUsed()
	case StrategyNUMA:
		return am.selectNUMA()
	case StrategyDedicated:
		return am.selectDedicated()
	default:
		return am.selectRoundRobin()
	}
}

// selectRoundRobin selects CPU using round-robin
func (am *AffinityManager) selectRoundRobin() int {
	cpu := am.nextCPU.Add(1) - 1
	return int(cpu) % am.cpuCount
}

// selectLeastUsed selects the least used CPU
func (am *AffinityManager) selectLeastUsed() int {
	minUsage := am.cpuUsage[0].Load()
	minCPU := 0

	for i := 1; i < am.cpuCount; i++ {
		usage := am.cpuUsage[i].Load()
		if usage < minUsage {
			minUsage = usage
			minCPU = i
		}
	}

	return minCPU
}

// selectNUMA selects CPU considering NUMA topology (simplified)
func (am *AffinityManager) selectNUMA() int {
	// Simplified NUMA-aware selection
	// In a real implementation, this would consider NUMA topology
	return am.selectLeastUsed()
}

// selectDedicated selects from dedicated cores
func (am *AffinityManager) selectDedicated() int {
	// For now, use least used strategy
	// In a real implementation, this would use a dedicated core pool
	return am.selectLeastUsed()
}

// GetCPUUsage returns current CPU usage statistics
func (am *AffinityManager) GetCPUUsage() []int32 {
	usage := make([]int32, am.cpuCount)
	for i := 0; i < am.cpuCount; i++ {
		usage[i] = am.cpuUsage[i].Load()
	}
	return usage
}

// GetAssignments returns current CPU assignments
func (am *AffinityManager) GetAssignments() map[int]int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	assignments := make(map[int]int)
	for gid, cpu := range am.assignments {
		assignments[gid] = cpu
	}
	return assignments
}

// CPUPool manages a pool of CPU cores for dedicated workloads
type CPUPool struct {
	cores     []int
	available chan int
	allocated map[int]bool
	mu        sync.RWMutex
}

// NewCPUPool creates a new CPU pool
func NewCPUPool(cores []int) *CPUPool {
	if len(cores) == 0 {
		// Default to all available cores
		cores = make([]int, runtime.NumCPU())
		for i := range cores {
			cores[i] = i
		}
	}

	pool := &CPUPool{
		cores:     cores,
		available: make(chan int, len(cores)),
		allocated: make(map[int]bool),
	}

	// Initialize available cores
	for _, core := range cores {
		pool.available <- core
	}

	return pool
}

// AllocateCore allocates a CPU core from the pool
func (cp *CPUPool) AllocateCore() (int, error) {
	select {
	case core := <-cp.available:
		cp.mu.Lock()
		cp.allocated[core] = true
		cp.mu.Unlock()
		return core, nil
	default:
		return -1, gerror.New(gerror.ErrCodeInternal, "no CPU cores available", nil)
	}
}

// ReleaseCore releases a CPU core back to the pool
func (cp *CPUPool) ReleaseCore(core int) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.allocated[core] {
		return gerror.New(gerror.ErrCodeInternal, "core not allocated", nil)
	}

	delete(cp.allocated, core)

	select {
	case cp.available <- core:
		return nil
	default:
		return gerror.New(gerror.ErrCodeInternal, "failed to return core to pool", nil)
	}
}

// AvailableCores returns the number of available cores
func (cp *CPUPool) AvailableCores() int {
	return len(cp.available)
}

// AllocatedCores returns the list of allocated cores
func (cp *CPUPool) AllocatedCores() []int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	cores := make([]int, 0, len(cp.allocated))
	for core := range cp.allocated {
		cores = append(cores, core)
	}
	return cores
}

// ThreadPinner provides thread pinning functionality
type ThreadPinner struct {
	enabled bool
	pool    *CPUPool
	pinned  sync.Map // goroutine ID -> CPU core
}

// NewThreadPinner creates a new thread pinner
func NewThreadPinner(enabled bool, pool *CPUPool) *ThreadPinner {
	return &ThreadPinner{
		enabled: enabled,
		pool:    pool,
	}
}

// Pin pins the current goroutine to a CPU core
func (tp *ThreadPinner) Pin() (int, error) {
	if !tp.enabled {
		return -1, nil
	}

	gid := getGoroutineID()

	// Check if already pinned
	if core, exists := tp.pinned.Load(gid); exists {
		return core.(int), nil
	}

	// Allocate new core
	core, err := tp.pool.AllocateCore()
	if err != nil {
		return -1, err
	}

	tp.pinned.Store(gid, core)
	return core, nil
}

// Unpin unpins the current goroutine
func (tp *ThreadPinner) Unpin() error {
	if !tp.enabled {
		return nil
	}

	gid := getGoroutineID()

	if coreInterface, exists := tp.pinned.LoadAndDelete(gid); exists {
		core := coreInterface.(int)
		return tp.pool.ReleaseCore(core)
	}

	return nil
}

// GetPinnedCore returns the CPU core the current goroutine is pinned to
func (tp *ThreadPinner) GetPinnedCore() (int, bool) {
	if !tp.enabled {
		return -1, false
	}

	gid := getGoroutineID()
	if core, exists := tp.pinned.Load(gid); exists {
		return core.(int), true
	}
	return -1, false
}

// NUMANode represents a NUMA node
type NUMANode struct {
	ID    int
	CPUs  []int
	Usage atomic.Int32
}

// NUMAManager manages NUMA-aware CPU allocation
type NUMAManager struct {
	nodes   []*NUMANode
	nodeMap map[int]*NUMANode // CPU -> NUMA node
	mu      sync.RWMutex
}

// NewNUMAManager creates a new NUMA manager
func NewNUMAManager() *NUMAManager {
	// Simplified NUMA detection - in practice this would query the system
	cpuCount := runtime.NumCPU()
	nodesCount := (cpuCount + 7) / 8 // Assume 8 CPUs per NUMA node

	nodes := make([]*NUMANode, nodesCount)
	nodeMap := make(map[int]*NUMANode)

	for i := 0; i < nodesCount; i++ {
		start := i * 8
		end := start + 8
		if end > cpuCount {
			end = cpuCount
		}

		cpus := make([]int, end-start)
		for j := start; j < end; j++ {
			cpus[j-start] = j
			nodeMap[j] = &NUMANode{ID: i}
		}

		nodes[i] = &NUMANode{
			ID:   i,
			CPUs: cpus,
		}

		for _, cpu := range cpus {
			nodeMap[cpu] = nodes[i]
		}
	}

	return &NUMAManager{
		nodes:   nodes,
		nodeMap: nodeMap,
	}
}

// GetNUMANode returns the NUMA node for a CPU
func (nm *NUMAManager) GetNUMANode(cpu int) *NUMANode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.nodeMap[cpu]
}

// GetOptimalNode returns the NUMA node with least usage
func (nm *NUMAManager) GetOptimalNode() *NUMANode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if len(nm.nodes) == 0 {
		return nil
	}

	optimal := nm.nodes[0]
	minUsage := optimal.Usage.Load()

	for _, node := range nm.nodes[1:] {
		usage := node.Usage.Load()
		if usage < minUsage {
			minUsage = usage
			optimal = node
		}
	}

	return optimal
}

// AllocateFromNode allocates a CPU from a specific NUMA node
func (nm *NUMAManager) AllocateFromNode(nodeID int) (int, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if nodeID >= len(nm.nodes) {
		return -1, gerror.New(gerror.ErrCodeInternal, "invalid NUMA node ID", nil)
	}

	node := nm.nodes[nodeID]
	if len(node.CPUs) == 0 {
		return -1, gerror.New(gerror.ErrCodeInternal, "no CPUs available in NUMA node", nil)
	}

	// Simple allocation - use first available CPU
	cpu := node.CPUs[0]
	node.Usage.Add(1)
	return cpu, nil
}

// ReleaseFromNode releases a CPU from a NUMA node
func (nm *NUMAManager) ReleaseFromNode(cpu int) error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	node := nm.nodeMap[cpu]
	if node == nil {
		return gerror.New(gerror.ErrCodeInternal, "CPU not found in any NUMA node", nil)
	}

	node.Usage.Add(-1)
	return nil
}

// GetNodes returns all NUMA nodes
func (nm *NUMAManager) GetNodes() []*NUMANode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	nodes := make([]*NUMANode, len(nm.nodes))
	copy(nodes, nm.nodes)
	return nodes
}

// CPUTopology provides information about CPU topology
type CPUTopology struct {
	Cores          int
	Threads        int
	Sockets        int
	CoresPerSocket int
	ThreadsPerCore int
	NUMANodes      int
}

// GetCPUTopology returns the CPU topology (simplified implementation)
func GetCPUTopology() CPUTopology {
	cores := runtime.NumCPU()

	// Simplified topology detection
	return CPUTopology{
		Cores:          cores,
		Threads:        cores, // Assume 1 thread per core
		Sockets:        1,     // Assume single socket
		CoresPerSocket: cores,
		ThreadsPerCore: 1,
		NUMANodes:      (cores + 7) / 8, // Assume 8 cores per NUMA node
	}
}

// PerformanceMonitor monitors CPU performance metrics
type PerformanceMonitor struct {
	samples    []CPUSample
	maxSamples int
	mu         sync.RWMutex
}

// CPUSample represents a CPU performance sample
type CPUSample struct {
	Timestamp       int64
	CoreUsage       []float64
	LoadAverage     float64
	ContextSwitches int64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(maxSamples int) *PerformanceMonitor {
	if maxSamples <= 0 {
		maxSamples = 1000
	}

	return &PerformanceMonitor{
		samples:    make([]CPUSample, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// AddSample adds a performance sample
func (pm *PerformanceMonitor) AddSample(sample CPUSample) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.samples) >= pm.maxSamples {
		// Remove oldest sample
		copy(pm.samples, pm.samples[1:])
		pm.samples = pm.samples[:len(pm.samples)-1]
	}

	pm.samples = append(pm.samples, sample)
}

// GetSamples returns all performance samples
func (pm *PerformanceMonitor) GetSamples() []CPUSample {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	samples := make([]CPUSample, len(pm.samples))
	copy(samples, pm.samples)
	return samples
}

// GetAverageUsage returns average CPU usage across all cores
func (pm *PerformanceMonitor) GetAverageUsage() float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.samples) == 0 {
		return 0
	}

	total := 0.0
	count := 0

	for _, sample := range pm.samples {
		for _, usage := range sample.CoreUsage {
			total += usage
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / float64(count)
}

// getGoroutineID returns a pseudo-goroutine ID for affinity management
func getGoroutineID() int {
	// This is a simplified implementation for demonstration
	// In production, you might use a more robust method or avoid per-goroutine tracking
	return int(uintptr(unsafe.Pointer(&struct{}{}))) % 10000
}
