// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// ResourceAllocation tracks resources allocated to a task
type ResourceAllocation struct {
	TaskID       string
	CPUCores     float64
	MemoryMB     int64
	GPUAllocated bool
	APIQuotas    map[string]int
	AllocatedAt  time.Time
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPUCoresUsed     float64
	CPUCoresTotal    float64
	MemoryMBUsed     int64
	MemoryMBTotal    int64
	GPUsUsed         int
	GPUsTotal        int
	APIQuotasUsed    map[string]int
	APIQuotasLimits  map[string]int
}

// SystemResources represents available system resources
type SystemResources struct {
	CPUCores      float64
	MemoryMB      int64
	GPUCount      int
	APIRateLimits map[string]int // provider -> requests/min
}

// ResourceManager manages resource allocation for tasks
type ResourceManager struct {
	system       SystemResources
	allocations  map[string]*ResourceAllocation
	apiUsage     map[string]*rateLimiter
	maxTasks     int
	mu           sync.RWMutex
}

// rateLimiter tracks API usage with sliding window
type rateLimiter struct {
	limit       int
	window      time.Duration
	requests    []time.Time
	mu          sync.Mutex
}

// NewResourceManager creates a new resource manager
func NewResourceManager(maxConcurrentTasks int) *ResourceManager {
	// Detect system resources
	cpuCores := float64(runtime.NumCPU())
	
	// Estimate available memory (simplified - in production use proper system calls)
	memoryMB := int64(8192) // Default 8GB
	
	// Default API rate limits
	apiLimits := map[string]int{
		"openai":    60,   // 60 requests/min
		"anthropic": 100,  // 100 requests/min
		"deepseek":  120,  // 120 requests/min
	}

	rm := &ResourceManager{
		system: SystemResources{
			CPUCores:      cpuCores,
			MemoryMB:      memoryMB,
			GPUCount:      0, // GPU detection would require specific libraries
			APIRateLimits: apiLimits,
		},
		allocations: make(map[string]*ResourceAllocation),
		apiUsage:    make(map[string]*rateLimiter),
		maxTasks:    maxConcurrentTasks,
	}

	// Initialize rate limiters
	for provider, limit := range apiLimits {
		rm.apiUsage[provider] = &rateLimiter{
			limit:    limit,
			window:   time.Minute,
			requests: make([]time.Time, 0),
		}
	}

	return rm
}

// CanAllocate checks if resources can be allocated for a task
func (rm *ResourceManager) CanAllocate(ctx context.Context, req ResourceRequirements) bool {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return false
	}
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Check task limit
	if len(rm.allocations) >= rm.maxTasks {
		return false
	}

	// Calculate current usage
	usage := rm.calculateUsage()

	// Check CPU
	if usage.CPUCoresUsed+req.CPUCores > rm.system.CPUCores {
		return false
	}

	// Check memory
	if usage.MemoryMBUsed+req.MemoryMB > rm.system.MemoryMB {
		return false
	}

	// Check GPU
	if req.GPURequired && usage.GPUsUsed >= rm.system.GPUCount {
		return false
	}

	// Check API quotas
	for provider, needed := range req.APIQuotas {
		limiter, exists := rm.apiUsage[provider]
		if !exists {
			return false // Unknown provider
		}

		if !limiter.canAllocate(needed) {
			return false
		}
	}

	return true
}

// Allocate reserves resources for a task
func (rm *ResourceManager) Allocate(ctx context.Context, taskID string, req ResourceRequirements) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("Allocate")
	}
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if already allocated
	if _, exists := rm.allocations[taskID]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "resources already allocated for task", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("Allocate").
			WithDetails("task_id", taskID)
	}

	// Verify we can allocate
	usage := rm.calculateUsage()
	if !rm.canAllocateUnsafe(req) {
		return gerror.New(gerror.ErrCodeInternal, "insufficient resources", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("Allocate").
			WithDetails("task_id", taskID).
			WithDetails("requested_cpu", req.CPUCores).
			WithDetails("available_cpu", rm.system.CPUCores-usage.CPUCoresUsed).
			WithDetails("requested_memory", req.MemoryMB).
			WithDetails("available_memory", rm.system.MemoryMB-usage.MemoryMBUsed)
	}

	// Create allocation
	allocation := &ResourceAllocation{
		TaskID:       taskID,
		CPUCores:     req.CPUCores,
		MemoryMB:     req.MemoryMB,
		GPUAllocated: req.GPURequired,
		APIQuotas:    make(map[string]int),
		AllocatedAt:  time.Now(),
	}

	// Copy API quotas
	for provider, quota := range req.APIQuotas {
		allocation.APIQuotas[provider] = quota
	}

	// Reserve API quotas
	for provider, needed := range req.APIQuotas {
		if limiter, exists := rm.apiUsage[provider]; exists {
			limiter.reserve(needed)
		}
	}

	rm.allocations[taskID] = allocation
	return nil
}

// Release frees resources allocated to a task
func (rm *ResourceManager) Release(ctx context.Context, taskID string) error {
	// Check context but don't fail - we want to clean up resources even if cancelled
	if err := ctx.Err(); err != nil {
		// Log warning but continue with cleanup
	}
	rm.mu.Lock()
	defer rm.mu.Unlock()

	allocation, exists := rm.allocations[taskID]
	if !exists {
		return nil
	}

	// Release API quotas
	for provider, quota := range allocation.APIQuotas {
		if limiter, exists := rm.apiUsage[provider]; exists {
			limiter.release(quota)
		}
	}

	delete(rm.allocations, taskID)
	return nil
}

// GetUsage returns current resource usage
func (rm *ResourceManager) GetUsage() ResourceUsage {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.calculateUsage()
}

// calculateUsage computes current resource usage (must be called with lock held)
func (rm *ResourceManager) calculateUsage() ResourceUsage {
	usage := ResourceUsage{
		CPUCoresTotal:   rm.system.CPUCores,
		MemoryMBTotal:   rm.system.MemoryMB,
		GPUsTotal:       rm.system.GPUCount,
		APIQuotasUsed:   make(map[string]int),
		APIQuotasLimits: rm.system.APIRateLimits,
	}

	// Sum allocated resources
	for _, alloc := range rm.allocations {
		usage.CPUCoresUsed += alloc.CPUCores
		usage.MemoryMBUsed += alloc.MemoryMB
		if alloc.GPUAllocated {
			usage.GPUsUsed++
		}
	}

	// Get API usage
	for provider, limiter := range rm.apiUsage {
		usage.APIQuotasUsed[provider] = limiter.getCurrentUsage()
	}

	return usage
}

// canAllocateUnsafe checks allocation without locking (must be called with lock held)
func (rm *ResourceManager) canAllocateUnsafe(req ResourceRequirements) bool {
	// Check task limit
	if len(rm.allocations) >= rm.maxTasks {
		return false
	}

	usage := rm.calculateUsage()

	// Check CPU
	if usage.CPUCoresUsed+req.CPUCores > rm.system.CPUCores {
		return false
	}

	// Check memory
	if usage.MemoryMBUsed+req.MemoryMB > rm.system.MemoryMB {
		return false
	}

	// Check GPU
	if req.GPURequired && usage.GPUsUsed >= rm.system.GPUCount {
		return false
	}

	// Check API quotas
	for provider, needed := range req.APIQuotas {
		limiter, exists := rm.apiUsage[provider]
		if !exists || !limiter.canAllocate(needed) {
			return false
		}
	}

	return true
}

// SetSystemResources updates available system resources
func (rm *ResourceManager) SetSystemResources(ctx context.Context, resources SystemResources) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("SetSystemResources")
	}
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.system = resources

	// Update or create rate limiters for new API providers
	for provider, limit := range resources.APIRateLimits {
		if _, exists := rm.apiUsage[provider]; !exists {
			rm.apiUsage[provider] = &rateLimiter{
				limit:    limit,
				window:   time.Minute,
				requests: make([]time.Time, 0),
			}
		} else {
			rm.apiUsage[provider].limit = limit
		}
	}
	return nil
}

// GetAllocations returns current resource allocations
func (rm *ResourceManager) GetAllocations() map[string]*ResourceAllocation {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Create a copy
	result := make(map[string]*ResourceAllocation, len(rm.allocations))
	for k, v := range rm.allocations {
		alloc := *v
		result[k] = &alloc
	}

	return result
}

// Rate limiter methods

func (rl *rateLimiter) canAllocate(needed int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cleanup()
	return len(rl.requests)+needed <= rl.limit
}

func (rl *rateLimiter) reserve(count int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for i := 0; i < count; i++ {
		rl.requests = append(rl.requests, now)
	}
}

func (rl *rateLimiter) release(count int) {
	// In a real implementation, we might track specific allocations
	// For now, we rely on the sliding window cleanup
}

func (rl *rateLimiter) getCurrentUsage() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cleanup()
	return len(rl.requests)
}

func (rl *rateLimiter) cleanup() {
	cutoff := time.Now().Add(-rl.window)
	
	// Remove old requests
	i := 0
	for i < len(rl.requests) && rl.requests[i].Before(cutoff) {
		i++
	}
	
	if i > 0 {
		rl.requests = rl.requests[i:]
	}
}

// ResourcePool manages reusable resources across tasks
type ResourcePool struct {
	name      string
	capacity  int
	available chan struct{}
	mu        sync.Mutex
}

// NewResourcePool creates a new resource pool
func NewResourcePool(name string, capacity int) *ResourcePool {
	pool := &ResourcePool{
		name:      name,
		capacity:  capacity,
		available: make(chan struct{}, capacity),
	}

	// Fill the pool
	for i := 0; i < capacity; i++ {
		pool.available <- struct{}{}
	}

	return pool
}

// Acquire gets a resource from the pool
func (rp *ResourcePool) Acquire() bool {
	select {
	case <-rp.available:
		return true
	default:
		return false
	}
}

// Release returns a resource to the pool
func (rp *ResourcePool) Release() {
	select {
	case rp.available <- struct{}{}:
		// Released successfully
	default:
		// Pool is full, ignore
	}
}

// GetAvailable returns the number of available resources
func (rp *ResourcePool) GetAvailable() int {
	return len(rp.available)
}