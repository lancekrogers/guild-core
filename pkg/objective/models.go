package objective

import (
	"fmt"
	"math/rand"
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