# BoltDB Kanban Integration

This document explains how to implement the Kanban board system using BoltDB.

## Overview

Guild uses a Kanban-style board to track tasks and their status. BoltDB provides a simple, embedded key-value store for persisting the Kanban board data.

## BoltDB Basics

BoltDB is a key-value store with the following characteristics:

- Single file storage
- ACID transactions
- No external dependencies
- B+tree data structure
- Buckets for organizing data

## Bucket Structure

Guild uses the following buckets in BoltDB:

| Bucket            | Description        | Key Format                                  | Value Format                |
| ----------------- | ------------------ | ------------------------------------------- | --------------------------- |
| `tasks`           | All tasks          | Task ID (`task-123`)                        | JSON-encoded Task           |
| `boards`          | All boards         | Board ID (`board-123`)                      | JSON-encoded Board metadata |
| `agent_tasks`     | Tasks by agent     | Agent ID + Task ID (`agent-123:task-456`)   | Task ID                     |
| `status_tasks`    | Tasks by status    | Status + Task ID (`ToDo:task-789`)          | Task ID                     |
| `objective_tasks` | Tasks by objective | Objective ID + Task ID (`obj-123:task-456`) | Task ID                     |

## Task Model

```go
// pkg/kanban/taskmodel.go
package kanban

import (
	"encoding/json"
	"time"
)

// TaskStatus represents the state of a task
type TaskStatus string

const (
	StatusToDo       TaskStatus = "ToDo"
	StatusInProgress TaskStatus = "InProgress"
	StatusBlocked    TaskStatus = "Blocked"
	StatusDone       TaskStatus = "Done"
)

// Task represents a unit of work
type Task struct {
	// ID is the unique identifier
	ID string `json:"id"`

	// Title is a short summary
	Title string `json:"title"`

	// Description is the full specification
	Description string `json:"description"`

	// Status is the current state
	Status TaskStatus `json:"status"`

	// AgentID is the assigned agent
	AgentID string `json:"agent_id"`

	// ObjectiveID is the associated objective
	ObjectiveID string `json:"objective_id"`

	// CreatedAt is when the task was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the task was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// Tags contains searchable tags
	Tags []string `json:"tags"`

	// Dependencies are tasks that must be completed first
	Dependencies []string `json:"dependencies,omitempty"`

	// Metadata contains additional information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MarshalJSON implements custom JSON marshaling
func (t Task) MarshalJSON() ([]byte, error) {
	type Alias Task
	return json.Marshal(&struct {
		Alias
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias:     Alias(t),
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling
func (t *Task) UnmarshalJSON(data []byte) error {
	type Alias Task
	aux := &struct {
		*Alias
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	createdAt, err := time.Parse(time.RFC3339, aux.CreatedAt)
	if err != nil {
		return err
	}
	t.CreatedAt = createdAt

	updatedAt, err := time.Parse(time.RFC3339, aux.UpdatedAt)
	if err != nil {
		return err
	}
	t.UpdatedAt = updatedAt

	return nil
}
```

## Board Implementation

```go
// pkg/kanban/board.go
package kanban

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
)

// BoltBoard implements the Board interface using BoltDB
type BoltBoard struct {
	ID     string
	Name   string
	db     *bolt.DB
	events EventPublisher
}

// EventPublisher publishes events when tasks change
type EventPublisher interface {
	PublishEvent(ctx context.Context, eventType, taskID, agentID string, data map[string]interface{}) error
}

// NewBoltBoard creates a new BoltDB-backed board
func NewBoltBoard(name string, dbPath string, events EventPublisher) (*BoltBoard, error) {
	// Open the database
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := []string{"tasks", "boards", "agent_tasks", "status_tasks", "objective_tasks"}
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	// Generate a unique ID for this board
	id := uuid.New().String()

	// Create the board
	board := &BoltBoard{
		ID:     id,
		Name:   name,
		db:     db,
		events: events,
	}

	// Save the board metadata
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("boards"))
		data, err := json.Marshal(map[string]string{
			"id":   id,
			"name": name,
		})
		if err != nil {
			return err
		}
		return b.Put([]byte(id), data)
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return board, nil
}

// Add creates a new task
func (b *BoltBoard) Add(ctx context.Context, task Task) error {
	// Validate task
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	if task.Status == "" {
		task.Status = StatusToDo
	}

	// Set timestamps
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now

	// Save to database
	err := b.db.Update(func(tx *bolt.Tx) error {
		// Save task
		tasksBucket := tx.Bucket([]byte("tasks"))
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		if err := tasksBucket.Put([]byte(task.ID), data); err != nil {
			return err
		}

		// Index by agent
		if task.AgentID != "" {
			agentBucket := tx.Bucket([]byte("agent_tasks"))
			key := []byte(fmt.Sprintf("%s:%s", task.AgentID, task.ID))
			if err := agentBucket.Put(key, []byte(task.ID)); err != nil {
				return err
			}
		}

		// Index by status
		statusBucket := tx.Bucket([]byte("status_tasks"))
		statusKey := []byte(fmt.Sprintf("%s:%s", task.Status, task.ID))
		if err := statusBucket.Put(statusKey, []byte(task.ID)); err != nil {
			return err
		}

		// Index by objective
		if task.ObjectiveID != "" {
			objBucket := tx.Bucket([]byte("objective_tasks"))
			key := []byte(fmt.Sprintf("%s:%s", task.ObjectiveID, task.ID))
			if err := objBucket.Put(key, []byte(task.ID)); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Publish event
	if b.events != nil {
		b.events.PublishEvent(ctx, "task_created", task.ID, task.AgentID, map[string]interface{}{
			"title":       task.Title,
			"description": task.Description,
			"status":      task.Status,
		})
	}

	return nil
}

// Get retrieves a task by ID
func (b *BoltBoard) Get(ctx context.Context, id string) (Task, error) {
	var task Task

	err := b.db.View(func(tx *bolt.Tx) error {
		tasksBucket := tx.Bucket([]byte("tasks"))
		data := tasksBucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("task not found: %s", id)
		}

		return json.Unmarshal(data, &task)
	})

	return task, err
}

// Update modifies an existing task
func (b *BoltBoard) Update(ctx context.Context, task Task) error {
	// Get existing task
	existingTask, err := b.Get(ctx, task.ID)
	if err != nil {
		return err
	}

	// Update timestamp
	task.CreatedAt = existingTask.CreatedAt
	task.UpdatedAt = time.Now()

	// Save to database
	err = b.db.Update(func(tx *bolt.Tx) error {
		// Update task
		tasksBucket := tx.Bucket([]byte("tasks"))
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		if err := tasksBucket.Put([]byte(task.ID), data); err != nil {
			return err
		}

		// Update agent index if changed
		if task.AgentID != existingTask.AgentID {
			agentBucket := tx.Bucket([]byte("agent_tasks"))

			// Remove old index
			if existingTask.AgentID != "" {
				oldKey := []byte(fmt.Sprintf("%s:%s", existingTask.AgentID, task.ID))
				if err := agentBucket.Delete(oldKey); err != nil {
					return err
				}
			}

			// Add new index
			if task.AgentID != "" {
				newKey := []byte(fmt.Sprintf("%s:%s", task.AgentID, task.ID))
				if err := agentBucket.Put(newKey, []byte(task.ID)); err != nil {
					return err
				}
			}
		}

		// Update status index if changed
		if task.Status != existingTask.Status {
			statusBucket := tx.Bucket([]byte("status_tasks"))

			// Remove old index
			oldKey := []byte(fmt.Sprintf("%s:%s", existingTask.Status, task.ID))
			if err := statusBucket.Delete(oldKey); err != nil {
				return err
			}

			// Add new index
			newKey := []byte(fmt.Sprintf("%s:%s", task.Status, task.ID))
			if err := statusBucket.Put(newKey, []byte(task.ID)); err != nil {
				return err
			}
		}

		// Update objective index if changed
		if task.ObjectiveID != existingTask.ObjectiveID {
			objBucket := tx.Bucket([]byte("objective_tasks"))

			// Remove old index
			if existingTask.ObjectiveID != "" {
				oldKey := []byte(fmt.Sprintf("%s:%s", existingTask.ObjectiveID, task.ID))
				if err := objBucket.Delete(oldKey); err != nil {
					return err
				}
			}

			// Add new index
			if task.ObjectiveID != "" {
				newKey := []byte(fmt.Sprintf("%s:%s", task.ObjectiveID, task.ID))
				if err := objBucket.Put(newKey, []byte(task.ID)); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Publish event
	if b.events != nil {
		b.events.PublishEvent(ctx, "task_updated", task.ID, task.AgentID, map[string]interface{}{
			"title":       task.Title,
			"description": task.Description,
			"status":      task.Status,
		})
	}

	return nil
}

// Move changes a task's status
func (b *BoltBoard) Move(ctx context.Context, id string, status TaskStatus) error {
	// Get existing task
	task, err := b.Get(ctx, id)
	if err != nil {
		return err
	}

	// Skip if status hasn't changed
	if task.Status == status {
		return nil
	}

	// Update status
	oldStatus := task.Status
	task.Status = status
	task.UpdatedAt = time.Now()

	// Save to database
	err = b.db.Update(func(tx *bolt.Tx) error {
		// Update task
		tasksBucket := tx.Bucket([]byte("tasks"))
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		if err := tasksBucket.Put([]byte(id), data); err != nil {
			return err
		}

		// Update status index
		statusBucket := tx.Bucket([]byte("status_tasks"))

		// Remove old index
		oldKey := []byte(fmt.Sprintf("%s:%s", oldStatus, id))
		if err := statusBucket.Delete(oldKey); err != nil {
			return err
		}

		// Add new index
		newKey := []byte(fmt.Sprintf("%s:%s", status, id))
		if err := statusBucket.Put(newKey, []byte(id)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Determine event type
	eventType := "task_moved"
	switch status {
	case StatusInProgress:
		if oldStatus == StatusToDo {
			eventType = "task_started"
		} else if oldStatus == StatusBlocked {
			eventType = "task_resumed"
		}
	case StatusBlocked:
		eventType = "task_blocked"
	case StatusDone:
		eventType = "task_completed"
	}

	// Publish event
	if b.events != nil {
		b.events.PublishEvent(ctx, eventType, id, task.AgentID, map[string]interface{}{
			"old_status": oldStatus,
			"new_status": status,
			"title":      task.Title,
		})
	}

	return nil
}

// List returns all tasks with optional filtering
func (b *BoltBoard) List(ctx context.Context, filter map[string]interface{}) ([]Task, error) {
	var tasks []Task

	err := b.db.View(func(tx *bolt.Tx) error {
		// Check if filtering by agent
		if agentID, ok := filter["agent_id"].(string); ok {
			agentBucket := tx.Bucket([]byte("agent_tasks"))
			tasksBucket := tx.Bucket([]byte("tasks"))

			prefix := []byte(fmt.Sprintf("%s:", agentID))
			cursor := agentBucket.Cursor()

			for k, v := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cursor.Next() {
				data := tasksBucket.Get(v)
				if data == nil {
					continue
				}

				var task Task
				if err := json.Unmarshal(data, &task); err != nil {
					return err
				}

				tasks = append(tasks, task)
			}

			return nil
		}

		// Check if filtering by status
		if status, ok := filter["status"].(string); ok {
			statusBucket := tx.Bucket([]byte("status_tasks"))
			tasksBucket := tx.Bucket([]byte("tasks"))

			prefix := []byte(fmt.Sprintf("%s:", status))
			cursor := statusBucket.Cursor()

			for k, v := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cursor.Next() {
				data := tasksBucket.Get(v)
				if data == nil {
					continue
				}

				var task Task
				if err := json.Unmarshal(data, &task); err != nil {
					return err
				}

				tasks = append(tasks, task)
			}

			return nil
		}

		// Check if filtering by objective
		if objectiveID, ok := filter["objective_id"].(string); ok {
			objBucket := tx.Bucket([]byte("objective_tasks"))
			tasksBucket := tx.Bucket([]byte("tasks"))

			prefix := []byte(fmt.Sprintf("%s:", objectiveID))
			cursor := objBucket.Cursor()

			for k, v := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cursor.Next() {
				data := tasksBucket.Get(v)
				if data == nil {
					continue
				}

				var task Task
				if err := json.Unmarshal(data, &task); err != nil {
					return err
				}

				tasks = append(tasks, task)
			}

			return nil
		}

		// No filters, return all tasks
		tasksBucket := tx.Bucket([]byte("tasks"))
		return tasksBucket.ForEach(func(k, v []byte) error {
			var task Task
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}

			tasks = append(tasks, task)
			return nil
		})
	})

	return tasks, err
}

// GetByAgent returns tasks assigned to an agent
func (b *BoltBoard) GetByAgent(ctx context.Context, agentID string) ([]Task, error) {
	return b.List(ctx, map[string]interface{}{
		"agent_id": agentID,
	})
}

// GetByStatus returns tasks with a specific status
func (b *BoltBoard) GetByStatus(ctx context.Context, status TaskStatus) ([]Task, error) {
	return b.List(ctx, map[string]interface{}{
		"status": string(status),
	})
}

// Close closes the board and database
func (b *BoltBoard) Close() error {
	return b.db.Close()
}
```

## Manager Implementation

```go
// pkg/kanban/manager.go
package kanban

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages multiple boards
type Manager struct {
	boards map[string]*BoltBoard
	mutex  sync.RWMutex
	events EventPublisher
}

// NewManager creates a new board manager
func NewManager(events EventPublisher) *Manager {
	return &Manager{
		boards: make(map[string]*BoltBoard),
		events: events,
	}
}

// CreateBoard creates a new board
func (m *Manager) CreateBoard(name, dbPath string) (*BoltBoard, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	board, err := NewBoltBoard(name, dbPath, m.events)
	if err != nil {
		return nil, err
	}

	m.boards[board.ID] = board
	return board, nil
}

// GetBoard retrieves a board by ID
func (m *Manager) GetBoard(id string) (*BoltBoard, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	board, ok := m.boards[id]
	if !ok {
		return nil, fmt.Errorf("board not found: %s", id)
	}

	return board, nil
}

// ListBoards returns all boards
func (m *Manager) ListBoards() []*BoltBoard {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	boards := make([]*BoltBoard, 0, len(m.boards))
	for _, board := range m.boards {
		boards = append(boards, board)
	}

	return boards
}

// CloseBoards closes all boards
func (m *Manager) CloseBoards() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastErr error
	for id, board := range m.boards {
		if err := board.Close(); err != nil {
			lastErr = err
		}
		delete(m.boards, id)
	}

	return lastErr
}
```

## Usage Examples

### Creating a Board

```go
// Create an event publisher
eventBus := orchestrator.NewEventBus("tcp://*:5555", "tcp://localhost:5555")

// Create a board manager
manager := kanban.NewManager(eventBus)

// Create a board
board, err := manager.CreateBoard("Development", "kanban.db")
if err != nil {
	log.Fatalf("Failed to create board: %v", err)
}
```

### Adding Tasks

```go
// Create a task
task := kanban.Task{
	Title:       "Implement user authentication",
	Description: "Create a login system with JWT",
	AgentID:     "agent-123",
	ObjectiveID: "obj-456",
	Status:      kanban.StatusToDo,
	Tags:        []string{"backend", "security"},
}

// Add task to board
ctx := context.Background()
err := board.Add(ctx, task)
if err != nil {
	log.Printf("Failed to add task: %v", err)
}
```

### Moving Tasks

```go
// Start working on a task
ctx := context.Background()
err := board.Move(ctx, "task-123", kanban.StatusInProgress)
if err != nil {
	log.Printf("Failed to move task: %v", err)
}

// Mark task as complete
err = board.Move(ctx, "task-123", kanban.StatusDone)
if err != nil {
	log.Printf("Failed to complete task: %v", err)
}
```

### Listing Tasks

```go
// Get all tasks
ctx := context.Background()
tasks, err := board.List(ctx, nil)
if err != nil {
	log.Printf("Failed to list tasks: %v", err)
}

// Get tasks by agent
agentTasks, err := board.GetByAgent(ctx, "agent-123")
if err != nil {
	log.Printf("Failed to get agent tasks: %v", err)
}

// Get tasks by status
todoTasks, err := board.GetByStatus(ctx, kanban.StatusToDo)
if err != nil {
	log.Printf("Failed to get todo tasks: %v", err)
}

// Get tasks by custom filter
filteredTasks, err := board.List(ctx, map[string]interface{}{
	"objective_id": "obj-456",
})
if err != nil {
	log.Printf("Failed to filter tasks: %v", err)
}
```

## Best Practices

1. **Transaction Usage**

   - Keep transactions short
   - Don't perform network operations in transactions
   - Handle transaction conflicts gracefully

2. **Performance Optimization**

   - Use indexes for common queries
   - Batch updates when possible
   - Implement pagination for large result sets

3. **Error Handling**
   - Retry operations on temporary errors
   - Use context for cancellation
   - Close database connections properly

## Related Documentation

- [BoltDB GitHub](https://github.com/boltdb/bolt)
- [../integration_guides/agent_task_events.md](../integration_guides/agent_task_events.md)
- [../architecture/task_execution_flow.md](../architecture/task_execution_flow.md)
