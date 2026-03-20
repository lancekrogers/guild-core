// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// DependencyNode represents a task in the dependency graph
type DependencyNode struct {
	TaskID       string
	Status       TaskStatus
	Dependencies []string
	Dependents   []string
	StartAfter   *time.Time
	CompletedAt  *time.Time
}

// DependencyGraph manages task dependencies
type DependencyGraph struct {
	nodes     map[string]*DependencyNode
	edges     map[string][]string // task -> dependencies
	completed map[string]bool
	mu        sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:     make(map[string]*DependencyNode),
		edges:     make(map[string][]string),
		completed: make(map[string]bool),
	}
}

// AddTask adds a task with its dependencies to the graph
func (dg *DependencyGraph) AddTask(taskID string, dependencies []string) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// Check for self-dependency
	for _, dep := range dependencies {
		if dep == taskID {
			return gerror.New(gerror.ErrCodeInvalidInput, "task cannot depend on itself", nil).
				WithComponent("orchestrator.scheduler").
				WithOperation("AddTask").
				WithDetails("task_id", taskID)
		}
	}

	// Create or update node
	node, exists := dg.nodes[taskID]
	if !exists {
		node = &DependencyNode{
			TaskID:       taskID,
			Status:       TaskStatusPending,
			Dependencies: make([]string, 0),
			Dependents:   make([]string, 0),
		}
		dg.nodes[taskID] = node
	}

	// Update dependencies
	node.Dependencies = dependencies
	dg.edges[taskID] = dependencies

	// Update dependents for each dependency
	for _, depID := range dependencies {
		depNode, exists := dg.nodes[depID]
		if !exists {
			depNode = &DependencyNode{
				TaskID:     depID,
				Status:     TaskStatusPending,
				Dependents: make([]string, 0),
			}
			dg.nodes[depID] = depNode
		}

		// Add current task as dependent
		found := false
		for _, d := range depNode.Dependents {
			if d == taskID {
				found = true
				break
			}
		}
		if !found {
			depNode.Dependents = append(depNode.Dependents, taskID)
		}
	}

	// Check for cycles
	if dg.hasCycle() {
		// Rollback changes
		delete(dg.nodes, taskID)
		delete(dg.edges, taskID)
		return gerror.New(gerror.ErrCodeInvalidInput, "adding task would create a dependency cycle", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("AddTask").
			WithDetails("task_id", taskID)
	}

	return nil
}

// AreSatisfied checks if all dependencies for a task are completed
func (dg *DependencyGraph) AreSatisfied(taskID string) bool {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	node, exists := dg.nodes[taskID]
	if !exists {
		return true // No dependencies if task doesn't exist
	}

	// Check each dependency
	for _, depID := range node.Dependencies {
		if !dg.completed[depID] {
			return false
		}
	}

	// Check time constraint
	if node.StartAfter != nil && time.Now().Before(*node.StartAfter) {
		return false
	}

	return true
}

// MarkComplete marks a task as completed
func (dg *DependencyGraph) MarkComplete(taskID string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	dg.completed[taskID] = true

	if node, exists := dg.nodes[taskID]; exists {
		node.Status = TaskStatusCompleted
		now := time.Now()
		node.CompletedAt = &now
	}
}

// MarkFailed marks a task as failed and blocks its dependents
func (dg *DependencyGraph) MarkFailed(taskID string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// Mark as failed (not completed)
	dg.completed[taskID] = false

	if node, exists := dg.nodes[taskID]; exists {
		node.Status = TaskStatusFailed
	}

	// TODO: Track failed status separately when needed for retry logic
}

// IsCompleted checks if a task is completed
func (dg *DependencyGraph) IsCompleted(taskID string) bool {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	return dg.completed[taskID]
}

// GetDependencies returns the dependencies of a task
func (dg *DependencyGraph) GetDependencies(taskID string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	if deps, exists := dg.edges[taskID]; exists {
		result := make([]string, len(deps))
		copy(result, deps)
		return result
	}

	return []string{}
}

// GetDependents returns tasks that depend on the given task
func (dg *DependencyGraph) GetDependents(taskID string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	if node, exists := dg.nodes[taskID]; exists {
		result := make([]string, len(node.Dependents))
		copy(result, node.Dependents)
		return result
	}

	return []string{}
}

// GetTopologicalOrder returns tasks in topological order
func (dg *DependencyGraph) GetTopologicalOrder(ctx context.Context) ([]string, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	queue := []string{}
	result := []string{}

	// Calculate in-degrees
	for taskID := range dg.nodes {
		inDegree[taskID] = len(dg.edges[taskID])
		if inDegree[taskID] == 0 {
			queue = append(queue, taskID)
		}
	}

	// Process queue
	for len(queue) > 0 {
		// Check context
		if ctx.Err() != nil {
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
				WithComponent("orchestrator.scheduler").
				WithOperation("GetTopologicalOrder")
		}

		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Update dependents
		if node, exists := dg.nodes[current]; exists {
			for _, dependent := range node.Dependents {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}
	}

	// Check if all nodes were processed
	if len(result) != len(dg.nodes) {
		return nil, gerror.New(gerror.ErrCodeInternal, "dependency graph contains cycles", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("GetTopologicalOrder")
	}

	return result, nil
}

// GetCriticalPath finds the longest path through the dependency graph
func (dg *DependencyGraph) GetCriticalPath(ctx context.Context, taskDurations map[string]time.Duration) ([]string, time.Duration, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	// Get topological order
	order, err := dg.GetTopologicalOrder(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Calculate earliest start times
	earliestStart := make(map[string]time.Duration)
	predecessor := make(map[string]string)

	for _, taskID := range order {
		maxStart := time.Duration(0)
		var pred string

		// Find maximum of all dependencies
		for _, depID := range dg.edges[taskID] {
			depEnd := earliestStart[depID] + taskDurations[depID]
			if depEnd > maxStart {
				maxStart = depEnd
				pred = depID
			}
		}

		earliestStart[taskID] = maxStart
		if pred != "" {
			predecessor[taskID] = pred
		}
	}

	// Find task with latest end time
	var endTask string
	maxEnd := time.Duration(0)

	for taskID, start := range earliestStart {
		end := start + taskDurations[taskID]
		if end > maxEnd {
			maxEnd = end
			endTask = taskID
		}
	}

	// Reconstruct critical path
	path := []string{}
	current := endTask
	for current != "" {
		path = append([]string{current}, path...)
		current = predecessor[current]
	}

	return path, maxEnd, nil
}

// hasCycle detects if the graph contains a cycle using DFS
func (dg *DependencyGraph) hasCycle() bool {
	// Color states: white (0) = unvisited, gray (1) = visiting, black (2) = visited
	colors := make(map[string]int)

	var visit func(taskID string) bool
	visit = func(taskID string) bool {
		colors[taskID] = 1 // Mark as visiting

		// Check all dependencies
		for _, depID := range dg.edges[taskID] {
			if colors[depID] == 1 {
				// Found a back edge (cycle)
				return true
			}
			if colors[depID] == 0 && visit(depID) {
				return true
			}
		}

		colors[taskID] = 2 // Mark as visited
		return false
	}

	// Check all nodes
	for taskID := range dg.nodes {
		if colors[taskID] == 0 {
			if visit(taskID) {
				return true
			}
		}
	}

	return false
}

// GetReadyTasks returns all tasks whose dependencies are satisfied
func (dg *DependencyGraph) GetReadyTasks() []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	ready := []string{}

	for taskID, node := range dg.nodes {
		// Skip if already completed
		if dg.completed[taskID] {
			continue
		}

		// Check if all dependencies are satisfied
		allSatisfied := true
		for _, depID := range node.Dependencies {
			if !dg.completed[depID] {
				allSatisfied = false
				break
			}
		}

		if allSatisfied {
			ready = append(ready, taskID)
		}
	}

	return ready
}

// Clear removes all tasks from the graph
func (dg *DependencyGraph) Clear() {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	dg.nodes = make(map[string]*DependencyNode)
	dg.edges = make(map[string][]string)
	dg.completed = make(map[string]bool)
}

// GetStats returns statistics about the dependency graph
func (dg *DependencyGraph) GetStats() map[string]interface{} {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	totalDeps := 0
	maxDeps := 0
	for _, deps := range dg.edges {
		totalDeps += len(deps)
		if len(deps) > maxDeps {
			maxDeps = len(deps)
		}
	}

	avgDeps := 0.0
	if len(dg.nodes) > 0 {
		avgDeps = float64(totalDeps) / float64(len(dg.nodes))
	}

	return map[string]interface{}{
		"total_tasks":      len(dg.nodes),
		"completed_tasks":  len(dg.completed),
		"total_edges":      totalDeps,
		"max_dependencies": maxDeps,
		"avg_dependencies": avgDeps,
	}
}
