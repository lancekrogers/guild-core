package kanban

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	// StatusBacklog indicates a task that hasn't been started
	StatusBacklog TaskStatus = "backlog"

	// StatusTodo indicates a task that is ready to be worked on
	StatusTodo TaskStatus = "todo"

	// StatusInProgress indicates a task that is currently being worked on
	StatusInProgress TaskStatus = "in_progress"

	// StatusBlocked indicates a task that is blocked by something
	StatusBlocked TaskStatus = "blocked"

	// StatusReadyForReview indicates a task that is ready for review
	StatusReadyForReview TaskStatus = "ready_for_review"

	// StatusDone indicates a completed task
	StatusDone TaskStatus = "done"

	// StatusCancelled indicates a cancelled task
	StatusCancelled TaskStatus = "cancelled"
)

// TaskPriority represents the priority of a task
type TaskPriority string

const (
	// PriorityHigh represents a high priority task
	PriorityHigh TaskPriority = "high"

	// PriorityMedium represents a medium priority task
	PriorityMedium TaskPriority = "medium"

	// PriorityLow represents a low priority task
	PriorityLow TaskPriority = "low"
)

// Task represents a task in the kanban board
type Task struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Status         TaskStatus        `json:"status"`
	Priority       TaskPriority      `json:"priority"`
	AssignedTo     string            `json:"assigned_to,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	DueDate        *time.Time        `json:"due_date,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ParentID       string            `json:"parent_id,omitempty"`
	Dependencies   []string          `json:"dependencies,omitempty"`
	Blockers       []string          `json:"blockers,omitempty"`
	Progress       int               `json:"progress"`
	EstimatedHours float64           `json:"estimated_hours,omitempty"`
	ActualHours    float64           `json:"actual_hours,omitempty"`
	Notes          []TaskNote        `json:"notes,omitempty"`
	History        []TaskHistory     `json:"history,omitempty"`
}

// TaskNote represents a note attached to a task
type TaskNote struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskHistory represents a historical change to a task
type TaskHistory struct {
	Timestamp    time.Time         `json:"timestamp"`
	ChangedBy    string            `json:"changed_by"`
	FromStatus   TaskStatus        `json:"from_status,omitempty"`
	ToStatus     TaskStatus        `json:"to_status,omitempty"`
	FromAssignee string            `json:"from_assignee,omitempty"`
	ToAssignee   string            `json:"to_assignee,omitempty"`
	Changes      map[string]string `json:"changes,omitempty"`
	Comment      string            `json:"comment,omitempty"`
}

// NewTask creates a new task with default values
func NewTask(title, description string) *Task {
	now := time.Now().UTC()
	return &Task{
		ID:           uuid.New().String(),
		Title:        title,
		Description:  description,
		Status:       StatusBacklog,
		Priority:     PriorityMedium,
		CreatedAt:    now,
		UpdatedAt:    now,
		Progress:     0,
		Metadata:     make(map[string]string),
		Tags:         []string{},
		Dependencies: []string{},
		Blockers:     []string{},
		Notes:        []TaskNote{},
		History:      []TaskHistory{},
	}
}

// AddNote adds a note to the task
func (t *Task) AddNote(content, createdBy string) {
	note := TaskNote{
		ID:        uuid.New().String(),
		Content:   content,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}
	t.Notes = append(t.Notes, note)
	t.UpdatedAt = time.Now().UTC()
}

// AddHistory adds a history entry to the task
func (t *Task) AddHistory(changedBy, comment string, changes map[string]string) {
	history := TaskHistory{
		Timestamp: time.Now().UTC(),
		ChangedBy: changedBy,
		Changes:   changes,
		Comment:   comment,
	}
	t.History = append(t.History, history)
}

// UpdateStatus updates the task's status and records the change in history
func (t *Task) UpdateStatus(newStatus TaskStatus, changedBy, comment string) error {
	if !IsValidStatus(newStatus) {
		return gerror.New(gerror.ErrCodeValidation, "invalid status", nil).
			WithComponent("kanban").
			WithOperation("UpdateStatus").
			WithDetails("attempted_status", string(newStatus))
	}

	oldStatus := t.Status
	t.Status = newStatus
	t.UpdatedAt = time.Now().UTC()

	// Update progress based on status
	switch newStatus {
	case StatusDone:
		t.Progress = 100
	case StatusCancelled:
		// Keep progress as is
	case StatusBacklog:
		t.Progress = 0
	}

	// Record the status change in history
	history := TaskHistory{
		Timestamp:  time.Now().UTC(),
		ChangedBy:  changedBy,
		FromStatus: oldStatus,
		ToStatus:   newStatus,
		Comment:    comment,
	}
	t.History = append(t.History, history)

	return nil
}

// UpdateAssignee updates the task's assignee and records the change in history
func (t *Task) UpdateAssignee(newAssignee, changedBy, comment string) {
	oldAssignee := t.AssignedTo
	t.AssignedTo = newAssignee
	t.UpdatedAt = time.Now().UTC()

	// Record the assignee change in history
	history := TaskHistory{
		Timestamp:    time.Now().UTC(),
		ChangedBy:    changedBy,
		FromAssignee: oldAssignee,
		ToAssignee:   newAssignee,
		Comment:      comment,
	}
	t.History = append(t.History, history)
}

// UpdateProgress updates the task's progress percentage
func (t *Task) UpdateProgress(progress int, changedBy, comment string) error {
	if progress < 0 || progress > 100 {
		return gerror.New(gerror.ErrCodeValidation, "progress must be between 0 and 100", nil).
			WithComponent("kanban").
			WithOperation("UpdateProgress").
			WithDetails("attempted_progress", fmt.Sprintf("%d", progress))
	}

	oldProgress := t.Progress
	t.Progress = progress
	t.UpdatedAt = time.Now().UTC()

	// Record the progress change in history
	changes := map[string]string{
		"progress": fmt.Sprintf("%d -> %d", oldProgress, progress),
	}
	t.AddHistory(changedBy, comment, changes)

	return nil
}

// AddDependency adds a dependency to the task
func (t *Task) AddDependency(dependencyID string) {
	// Check if dependency already exists
	for _, dep := range t.Dependencies {
		if dep == dependencyID {
			return // Already exists
		}
	}

	t.Dependencies = append(t.Dependencies, dependencyID)
	t.UpdatedAt = time.Now().UTC()
}

// RemoveDependency removes a dependency from the task
func (t *Task) RemoveDependency(dependencyID string) {
	var newDependencies []string
	for _, dep := range t.Dependencies {
		if dep != dependencyID {
			newDependencies = append(newDependencies, dep)
		}
	}
	t.Dependencies = newDependencies
	t.UpdatedAt = time.Now().UTC()
}

// AddBlocker adds a blocker to the task
func (t *Task) AddBlocker(blockerID string, changedBy, comment string) {
	// Check if blocker already exists
	for _, b := range t.Blockers {
		if b == blockerID {
			return // Already exists
		}
	}

	t.Blockers = append(t.Blockers, blockerID)
	t.UpdatedAt = time.Now().UTC()

	// If we're adding a blocker, automatically set status to blocked
	if t.Status != StatusBlocked && len(t.Blockers) == 1 {
		t.UpdateStatus(StatusBlocked, changedBy, "Task blocked")
	}

	// Record the blocker addition in history
	changes := map[string]string{
		"blocker_added": blockerID,
	}
	t.AddHistory(changedBy, comment, changes)
}

// RemoveBlocker removes a blocker from the task
func (t *Task) RemoveBlocker(blockerID string, changedBy, comment string) {
	var newBlockers []string
	removed := false

	for _, b := range t.Blockers {
		if b != blockerID {
			newBlockers = append(newBlockers, b)
		} else {
			removed = true
		}
	}

	if !removed {
		return // Blocker not found
	}

	t.Blockers = newBlockers
	t.UpdatedAt = time.Now().UTC()

	// If we removed the last blocker, change status from blocked to todo
	if len(t.Blockers) == 0 && t.Status == StatusBlocked {
		t.UpdateStatus(StatusTodo, changedBy, "Task unblocked")
	}

	// Record the blocker removal in history
	changes := map[string]string{
		"blocker_removed": blockerID,
	}
	t.AddHistory(changedBy, comment, changes)
}

// IsValidStatus checks if a status is valid
func IsValidStatus(status TaskStatus) bool {
	switch status {
	case StatusBacklog, StatusTodo, StatusInProgress, StatusBlocked, StatusReadyForReview, StatusDone, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsValidPriority checks if a priority is valid
func IsValidPriority(priority TaskPriority) bool {
	switch priority {
	case PriorityHigh, PriorityMedium, PriorityLow:
		return true
	default:
		return false
	}
}

// IsBlocked checks if a task is blocked
func (t *Task) IsBlocked() bool {
	return len(t.Blockers) > 0
}

// TaskFilter is a function that filters tasks
type TaskFilter func(*Task) bool

// FilterByStatus creates a filter for tasks with the given status
func FilterByStatus(status TaskStatus) TaskFilter {
	return func(t *Task) bool {
		return t.Status == status
	}
}

// FilterByAssignee creates a filter for tasks assigned to the given user
func FilterByAssignee(assignee string) TaskFilter {
	return func(t *Task) bool {
		return t.AssignedTo == assignee
	}
}

// FilterByPriority creates a filter for tasks with the given priority
func FilterByPriority(priority TaskPriority) TaskFilter {
	return func(t *Task) bool {
		return t.Priority == priority
	}
}

// FilterByTag creates a filter for tasks with the given tag
func FilterByTag(tag string) TaskFilter {
	return func(t *Task) bool {
		for _, t := range t.Tags {
			if t == tag {
				return true
			}
		}
		return false
	}
}

// CombineFilters combines multiple filters with AND logic
func CombineFilters(filters ...TaskFilter) TaskFilter {
	return func(t *Task) bool {
		for _, filter := range filters {
			if !filter(t) {
				return false
			}
		}
		return true
	}
}
