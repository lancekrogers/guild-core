package commission

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// CommissionStatus represents the status of a commission
type CommissionStatus string

const (
	// CommissionStatusDraft indicates a draft commission
	CommissionStatusDraft CommissionStatus = "draft"
	
	// CommissionStatusActive indicates an active commission
	CommissionStatusActive CommissionStatus = "active"
	
	// CommissionStatusCompleted indicates a completed commission
	CommissionStatusCompleted CommissionStatus = "completed"
	
	// CommissionStatusCancelled indicates a cancelled commission
	CommissionStatusCancelled CommissionStatus = "cancelled"
)

// Commission represents a goal or task to be accomplished
type Commission struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      CommissionStatus  `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Context     []string          `json:"context,omitempty"` // References to context documents
	Parts       []*CommissionPart `json:"parts,omitempty"`
	Tasks       []*CommissionTask `json:"tasks,omitempty"`
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
	CampaignID  string            `json:"campaign_id,omitempty"` // Optional campaign association
}

// Format formats a commission as a markdown string
func (c *Commission) Format() string {
	var md strings.Builder

	// Title
	md.WriteString(fmt.Sprintf("# %s\n\n", c.Title))

	// Description
	md.WriteString(fmt.Sprintf("%s\n\n", c.Description))

	// Metadata
	md.WriteString("## Metadata\n\n")
	md.WriteString(fmt.Sprintf("- Status: %s\n", c.Status))
	md.WriteString(fmt.Sprintf("- Owner: %s\n", c.Owner))
	if len(c.Assignees) > 0 {
		md.WriteString(fmt.Sprintf("- Assignees: %s\n", strings.Join(c.Assignees, ", ")))
	}
	if c.Priority != "" {
		md.WriteString(fmt.Sprintf("- Priority: %s\n", c.Priority))
	}
	if len(c.Tags) > 0 {
		md.WriteString(fmt.Sprintf("- Tags: %s\n", strings.Join(c.Tags, ", ")))
	}
	md.WriteString("\n")

	// Parts
	for _, part := range c.Parts {
		md.WriteString(fmt.Sprintf("## %s\n\n", part.Title))
		md.WriteString(fmt.Sprintf("%s\n\n", part.Content))
	}

	// Tasks
	if len(c.Tasks) > 0 {
		md.WriteString("## Tasks\n\n")
		for _, task := range c.Tasks {
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

// CommissionPart represents a section of a commission
type CommissionPart struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Type        string            `json:"type"`     // context, goal, acceptance, implementation, etc.
	SortOrder   int               `json:"sort_order"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CommissionTask represents a task to complete a commission
type CommissionTask struct {
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

// CommissionParser defines the interface for commission parsers
type CommissionParser interface {
	// Parse parses a commission from markdown content
	Parse(content, source string) (*Commission, error)
	
	// ParseFile parses a commission from a markdown file
	ParseFile(filepath string) (*Commission, error)
}

// SectionInfo represents information about a markdown section
type SectionInfo struct {
	Title    string
	Level    int
	Content  string
	Type     string
	MetaTags map[string]string
}

// ParseOptions contains options for parsing commissions
type ParseOptions struct {
	DefaultStatus  CommissionStatus
	DefaultPriority string
	DefaultOwner    string
	TagPrefixes     []string
	MetaPrefixes    []string
}

// DefaultParseOptions returns default parse options
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		DefaultStatus:   CommissionStatusDraft,
		DefaultPriority: "medium",
		TagPrefixes:     []string{"#", "@tag:"},
		MetaPrefixes:    []string{"@", "meta:"},
	}
}

// CommissionGenerator defines the interface for commission generators
type CommissionGenerator interface {
	// GenerateCommission generates a new commission from a description
	GenerateCommission(description string) (*Commission, error)
	
	// RefineCommission refines an existing commission
	RefineCommission(commission *Commission) (*Commission, error)
	
	// GenerateTasks generates tasks for a commission
	GenerateTasks(commission *Commission) ([]*CommissionTask, error)
}

// NewCommission creates a new commission with default values
func NewCommission(title, description string) *Commission {
	now := time.Now().UTC()
	return &Commission{
		ID:          GenerateID(),
		Title:       title,
		Description: description,
		Status:      CommissionStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        []string{},
		Metadata:    make(map[string]string),
		Context:     []string{},
		Parts:       []*CommissionPart{},
		Tasks:       []*CommissionTask{},
		Priority:    "medium",
	}
}

// NewCommissionPart creates a new commission part
func NewCommissionPart(title, content, partType string, sortOrder int) *CommissionPart {
	return &CommissionPart{
		ID:        GenerateID(),
		Title:     title,
		Content:   content,
		Type:      partType,
		SortOrder: sortOrder,
		Metadata:  make(map[string]string),
	}
}

// NewCommissionTask creates a new commission task
func NewCommissionTask(title, description string, sortOrder int) *CommissionTask {
	now := time.Now().UTC()
	return &CommissionTask{
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

// GenerateID generates a unique ID for commissions, parts, and tasks
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
func (c *Commission) CalculateCompletion() {
	// Simple implementation - could be enhanced based on tasks completion, etc.
	switch c.Status {
	case CommissionStatusDraft:
		c.Completion = 0.1
	case CommissionStatusActive:
		if len(c.Tasks) == 0 {
			c.Completion = 0.3
		} else {
			completed := 0
			for _, task := range c.Tasks {
				if task.Status == "done" {
					completed++
				}
			}
			c.Completion = 0.3 + 0.6*float64(completed)/float64(len(c.Tasks))
		}
	case CommissionStatusCompleted:
		c.Completion = 1.0
	default:
		c.Completion = 0.5 // Default for other statuses
	}
}

// IncrementIteration increments the iteration counter
func (c *Commission) IncrementIteration() {
	c.Iteration++
	c.UpdatedAt = time.Now()
}