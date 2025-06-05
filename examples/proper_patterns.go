package examples

import (
	"context"
	"fmt"
	"time"
)

// Example of proper context passing, error wrapping, and interface usage

// TaskProcessor defines the interface for task processing
// This follows the registry pattern - define interfaces, not structs
type TaskProcessor interface {
	ProcessTask(ctx context.Context, taskID string) error
	ValidateTask(ctx context.Context, task Task) error
}

// Task represents a task to be processed
type Task struct {
	ID          string
	Title       string
	Description string
}

// taskProcessor implements TaskProcessor
type taskProcessor struct {
	repository TaskRepository
	validator  TaskValidator
	executor   TaskExecutor
}

// TaskRepository defines storage operations
type TaskRepository interface {
	GetTask(ctx context.Context, id string) (*Task, error)
	SaveTask(ctx context.Context, task *Task) error
}

// TaskValidator validates tasks
type TaskValidator interface {
	Validate(ctx context.Context, task Task) error
}

// TaskExecutor executes tasks
type TaskExecutor interface {
	Execute(ctx context.Context, task Task) error
}

// NewTaskProcessor creates a new task processor using the registry pattern
func NewTaskProcessor(repository TaskRepository, validator TaskValidator, executor TaskExecutor) TaskProcessor {
	return &taskProcessor{
		repository: repository,
		validator:  validator,
		executor:   executor,
	}
}

// ProcessTask demonstrates proper context passing and error wrapping
func (p *taskProcessor) ProcessTask(ctx context.Context, taskID string) error {
	// Always check context first
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before processing task %s: %w", taskID, err)
	}

	// Proper error wrapping with context
	task, err := p.repository.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task %s: %w", taskID, err)
	}

	// Validate with context
	if err := p.ValidateTask(ctx, *task); err != nil {
		return fmt.Errorf("validation failed for task %s: %w", taskID, err)
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	if err := p.executor.Execute(execCtx, *task); err != nil {
		return fmt.Errorf("failed to execute task %s: %w", taskID, err)
	}

	// Update task status
	task.Description = "Completed"
	if err := p.repository.SaveTask(ctx, task); err != nil {
		return fmt.Errorf("failed to save task %s after execution: %w", taskID, err)
	}

	return nil
}

// ValidateTask shows proper error context
func (p *taskProcessor) ValidateTask(ctx context.Context, task Task) error {
	// Check context
	select {
	case <-ctx.Done():
		return fmt.Errorf("validation cancelled: %w", ctx.Err())
	default:
		// Continue with validation
	}

	if task.ID == "" {
		return fmt.Errorf("task validation failed: ID cannot be empty")
	}

	if task.Title == "" {
		return fmt.Errorf("task validation failed: title cannot be empty for task %s", task.ID)
	}

	// Delegate to validator with context
	if err := p.validator.Validate(ctx, task); err != nil {
		return fmt.Errorf("task %s validation failed: %w", task.ID, err)
	}

	return nil
}

// Example of registry usage
type ProcessorRegistry interface {
	RegisterProcessor(name string, processor TaskProcessor) error
	GetProcessor(name string) (TaskProcessor, error)
}

// CreateProcessorFromRegistry shows how to use the registry pattern
func CreateProcessorFromRegistry(ctx context.Context, registry ProcessorRegistry, name string) (TaskProcessor, error) {
	processor, err := registry.GetProcessor(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get processor %s from registry: %w", name, err)
	}
	
	return processor, nil
}

// Example of custom error types for better error handling
type TaskError struct {
	TaskID    string
	Operation string
	Err       error
}

func (e *TaskError) Error() string {
	return fmt.Sprintf("task %s: %s failed: %v", e.TaskID, e.Operation, e.Err)
}

func (e *TaskError) Unwrap() error {
	return e.Err
}

// WrapTaskError creates a properly wrapped task error
func WrapTaskError(taskID, operation string, err error) error {
	if err == nil {
		return nil
	}
	return &TaskError{
		TaskID:    taskID,
		Operation: operation,
		Err:       err,
	}
}