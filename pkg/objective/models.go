package objective

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ObjectiveStatus represents the status of an objective
type ObjectiveStatus string

const (
	// ObjectiveStatusDraft indicates a draft objective
	ObjectiveStatusDraft ObjectiveStatus = "draft"
	
	// ObjectiveStatusActive indicates an active objective
	ObjectiveStatusActive ObjectiveStatus = "active"
	
	// ObjectiveStatusCompleted indicates a completed objective
	ObjectiveStatusCompleted ObjectiveStatus = "completed"
	
	// ObjectiveStatusCancelled indicates a cancelled objective
	ObjectiveStatusCancelled ObjectiveStatus = "cancelled"
)

// Objective represents a goal or task to be accomplished
type Objective struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      ObjectiveStatus   `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Context     []string          `json:"context,omitempty"` // References to context documents
	Parts       []*ObjectivePart  `json:"parts,omitempty"`
	Tasks       []*ObjectiveTask  `json:"tasks,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Assignees   []string          `json:"assignees,omitempty"`
	Priority    string            `json:"priority,omitempty"` // high, medium, low
	Source      string            `json:"source,omitempty"`   // Path to the source file
	Content     string            `json:"-"`                  // Original content, not stored in JSON
	FilePath    string            `json:"-"`                  // Path to the file
	FileName    string            `json:"-"`                  // Name of the file
	Goal        string            `json:"goal,omitempty"`     // The main goal of the objective
	Requirements []string         `json:"requirements,omitempty"` // Requirements for completion
	Related     []string          `json:"related,omitempty"`  // Related objectives
	AIDocs      []string          `json:"ai_docs,omitempty"`  // AI documentation paths
	Specs       []string          `json:"specs,omitempty"`    // Specification paths
	Completion  float64           `json:"completion"`         // Completion percentage (0.0-1.0)
	Iteration   int               `json:"iteration"`          // Current iteration count
}

// Format formats an objective as a markdown string
func (o *Objective) Format() string {
	var md strings.Builder

	// Title
	md.WriteString(fmt.Sprintf("# %s\n\n", o.Title))

	// Description
	md.WriteString(fmt.Sprintf("%s\n\n", o.Description))

	// Metadata
	md.WriteString("## Metadata\n\n")
	md.WriteString(fmt.Sprintf("- Status: %s\n", o.Status))
	md.WriteString(fmt.Sprintf("- Owner: %s\n", o.Owner))
	if len(o.Assignees) > 0 {
		md.WriteString(fmt.Sprintf("- Assignees: %s\n", strings.Join(o.Assignees, ", ")))
	}
	if o.Priority != "" {
		md.WriteString(fmt.Sprintf("- Priority: %s\n", o.Priority))
	}
	if len(o.Tags) > 0 {
		md.WriteString(fmt.Sprintf("- Tags: %s\n", strings.Join(o.Tags, ", ")))
	}
	md.WriteString("\n")

	// Parts
	for _, part := range o.Parts {
		md.WriteString(fmt.Sprintf("## %s\n\n", part.Title))
		md.WriteString(fmt.Sprintf("%s\n\n", part.Content))
	}

	// Tasks
	if len(o.Tasks) > 0 {
		md.WriteString("## Tasks\n\n")
		for _, task := range o.Tasks {
			status := ""
			if task.Status == "done" {
				status = "[x]"
			} else {
				status = "[ ]"
			}
			md.WriteString(fmt.Sprintf("- %s %s\n", status, task.Title))
		}
	}

	return md.String()
}

// ObjectivePart represents a section of an objective
type ObjectivePart struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Type        string            `json:"type"`     // context, goal, acceptance, implementation, etc.
	SortOrder   int               `json:"sort_order"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ObjectiveTask represents a task to complete an objective
type ObjectiveTask struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      string            `json:"status"`    // todo, in_progress, done, etc.
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Assignee    string            `json:"assignee,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty"` // IDs of tasks this depends on
	SortOrder   int               `json:"sort_order"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ParentID    string            `json:"parent_id,omitempty"` // For hierarchical tasks
}

// ObjectiveParser defines the interface for objective parsers
type ObjectiveParser interface {
	// Parse parses an objective from markdown content
	Parse(content, source string) (*Objective, error)
	
	// ParseFile parses an objective from a markdown file
	ParseFile(filepath string) (*Objective, error)
}

// SectionInfo represents information about a markdown section
type SectionInfo struct {
	Title    string
	Level    int
	Content  string
	Type     string
	MetaTags map[string]string
}

// ParseOptions contains options for parsing objectives
type ParseOptions struct {
	DefaultStatus  ObjectiveStatus
	DefaultPriority string
	DefaultOwner    string
	TagPrefixes     []string
	MetaPrefixes    []string
}

// DefaultParseOptions returns default parse options
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		DefaultStatus:   ObjectiveStatusDraft,
		DefaultPriority: "medium",
		TagPrefixes:     []string{"#", "@tag:"},
		MetaPrefixes:    []string{"@", "meta:"},
	}
}

// ObjectiveGenerator defines the interface for objective generators
type ObjectiveGenerator interface {
	// GenerateObjective generates a new objective from a description
	GenerateObjective(description string) (*Objective, error)
	
	// RefineObjective refines an existing objective
	RefineObjective(objective *Objective) (*Objective, error)
	
	// GenerateTasks generates tasks for an objective
	GenerateTasks(objective *Objective) ([]*ObjectiveTask, error)
}

// NewObjective creates a new objective with default values
func NewObjective(title, description string) *Objective {
	now := time.Now().UTC()
	return &Objective{
		ID:          GenerateID(),
		Title:       title,
		Description: description,
		Status:      ObjectiveStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        []string{},
		Metadata:    make(map[string]string),
		Context:     []string{},
		Parts:       []*ObjectivePart{},
		Tasks:       []*ObjectiveTask{},
		Priority:    "medium",
	}
}

// NewObjectivePart creates a new objective part
func NewObjectivePart(title, content, partType string, sortOrder int) *ObjectivePart {
	return &ObjectivePart{
		ID:        GenerateID(),
		Title:     title,
		Content:   content,
		Type:      partType,
		SortOrder: sortOrder,
		Metadata:  make(map[string]string),
	}
}

// NewObjectiveTask creates a new objective task
func NewObjectiveTask(title, description string, sortOrder int) *ObjectiveTask {
	now := time.Now().UTC()
	return &ObjectiveTask{
		ID:          GenerateID(),
		Title:       title,
		Description: description,
		Status:      "todo",
		CreatedAt:   now,
		UpdatedAt:   now,
		SortOrder:   sortOrder,
		Metadata:    make(map[string]string),
		Dependencies: []string{},
	}
}

// GenerateID generates a unique ID for objectives, parts, and tasks
func GenerateID() string {
	// Use timestamp + random characters for simplicity
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(6))
}

// randomString generates a random string of the specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// CalculateCompletion calculates and updates the completion percentage
func (o *Objective) CalculateCompletion() {
	// Simple implementation - could be enhanced based on tasks completion, etc.
	switch o.Status {
	case ObjectiveStatusDraft:
		o.Completion = 0.1
	case ObjectiveStatusActive:
		if len(o.Tasks) == 0 {
			o.Completion = 0.3
		} else {
			completed := 0
			for _, task := range o.Tasks {
				if task.Status == "done" {
					completed++
				}
			}
			o.Completion = 0.3 + 0.6*float64(completed)/float64(len(o.Tasks))
		}
	case ObjectiveStatusCompleted:
		o.Completion = 1.0
	default:
		o.Completion = 0.5 // Default for other statuses
	}
}

// IncrementIteration increments the iteration counter
func (o *Objective) IncrementIteration() {
	o.Iteration++
	o.UpdatedAt = time.Now()
}