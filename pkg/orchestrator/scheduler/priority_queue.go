// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"container/heap"
	"sync"
)

// PriorityQueueItem wraps a task for the priority queue
type PriorityQueueItem struct {
	Task     *SchedulableTask
	Priority int
	Index    int // Index in the heap
}

// priorityQueueHeap implements heap.Interface
type priorityQueueHeap []*PriorityQueueItem

func (pq priorityQueueHeap) Len() int { return len(pq) }

func (pq priorityQueueHeap) Less(i, j int) bool {
	// Higher priority values come first
	return pq[i].Priority > pq[j].Priority
}

func (pq priorityQueueHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *priorityQueueHeap) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueueHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// PriorityQueue is a thread-safe priority queue for tasks
type PriorityQueue struct {
	items  priorityQueueHeap
	lookup map[string]*PriorityQueueItem
	mu     sync.RWMutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items:  make(priorityQueueHeap, 0),
		lookup: make(map[string]*PriorityQueueItem),
	}
	heap.Init(&pq.items)
	return pq
}

// Push adds a task to the queue
func (pq *PriorityQueue) Push(task *SchedulableTask) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Check if task already exists
	if _, exists := pq.lookup[task.ID]; exists {
		return
	}

	item := &PriorityQueueItem{
		Task:     task,
		Priority: task.Priority,
	}

	heap.Push(&pq.items, item)
	pq.lookup[task.ID] = item
}

// Pop removes and returns the highest priority task
func (pq *PriorityQueue) Pop() *SchedulableTask {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		return nil
	}

	item := heap.Pop(&pq.items).(*PriorityQueueItem)
	delete(pq.lookup, item.Task.ID)
	return item.Task
}

// Peek returns the highest priority task without removing it
func (pq *PriorityQueue) Peek() *SchedulableTask {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if len(pq.items) == 0 {
		return nil
	}

	return pq.items[0].Task
}

// Remove removes a specific task from the queue
func (pq *PriorityQueue) Remove(taskID string) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item, exists := pq.lookup[taskID]
	if !exists {
		return false
	}

	heap.Remove(&pq.items, item.Index)
	delete(pq.lookup, taskID)
	return true
}

// UpdatePriority changes the priority of a task in the queue
func (pq *PriorityQueue) UpdatePriority(taskID string, newPriority int) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item, exists := pq.lookup[taskID]
	if !exists {
		return false
	}

	item.Priority = newPriority
	item.Task.Priority = newPriority
	heap.Fix(&pq.items, item.Index)
	return true
}

// Contains checks if a task is in the queue
func (pq *PriorityQueue) Contains(taskID string) bool {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	_, exists := pq.lookup[taskID]
	return exists
}

// Len returns the number of tasks in the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return len(pq.items)
}

// Range calls f for each task in the queue
func (pq *PriorityQueue) Range(f func(*SchedulableTask) bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// Create a copy to iterate safely
	items := make([]*PriorityQueueItem, len(pq.items))
	copy(items, pq.items)

	for _, item := range items {
		if !f(item.Task) {
			break
		}
	}
}

// Clear removes all tasks from the queue
func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.items = make(priorityQueueHeap, 0)
	pq.lookup = make(map[string]*PriorityQueueItem)
	heap.Init(&pq.items)
}

// GetTasks returns all tasks in priority order
func (pq *PriorityQueue) GetTasks() []*SchedulableTask {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	tasks := make([]*SchedulableTask, len(pq.items))
	for i, item := range pq.items {
		tasks[i] = item.Task
	}

	return tasks
}