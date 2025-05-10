package objective

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blockhead-consulting/Guild/pkg/memory"
)

// Manager handles storage and retrieval of objectives
type Manager struct {
	store          memory.Store
	fsBasePath     string
	memoryBucket   string
	tagsIndex      map[string][]string // tag -> objective IDs
	objectiveCache map[string]*Objective
	mu             sync.RWMutex
}

// NewManager creates a new objective manager
func NewManager(store memory.Store, fsBasePath string) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	if fsBasePath == "" {
		// Default to current directory if not specified
		var err error
		fsBasePath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		fsBasePath = filepath.Join(fsBasePath, "objectives")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(fsBasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create objectives directory: %w", err)
	}

	return &Manager{
		store:          store,
		fsBasePath:     fsBasePath,
		memoryBucket:   "objectives",
		tagsIndex:      make(map[string][]string),
		objectiveCache: make(map[string]*Objective),
	}, nil
}

// Init initializes the manager by loading existing indices
func (m *Manager) Init(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing indices
	m.tagsIndex = make(map[string][]string)
	m.objectiveCache = make(map[string]*Objective)

	// List all objectives in the store
	objectiveIDs, err := m.store.List(ctx, m.memoryBucket)
	if err != nil {
		return fmt.Errorf("failed to list objectives: %w", err)
	}

	// Load each objective and rebuild indices
	for _, id := range objectiveIDs {
		obj, err := m.loadObjectiveFromStore(ctx, id)
		if err != nil {
			// Log the error but continue with other objectives
			fmt.Printf("Warning: failed to load objective %s: %v\n", id, err)
			continue
		}

		// Add to cache
		m.objectiveCache[id] = obj

		// Update the tags index
		for _, tag := range obj.Tags {
			m.tagsIndex[tag] = append(m.tagsIndex[tag], id)
		}
	}

	return nil
}

// SaveObjective stores an objective both in memory store and filesystem
func (m *Manager) SaveObjective(ctx context.Context, obj *Objective) error {
	if obj == nil {
		return fmt.Errorf("objective cannot be nil")
	}

	if obj.ID == "" {
		return fmt.Errorf("objective ID cannot be empty")
	}

	// Set updated time
	obj.UpdatedAt = time.Now().UTC()

	// Serialize the objective
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal objective: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Save to store
	if err := m.store.Put(ctx, m.memoryBucket, obj.ID, data); err != nil {
		return fmt.Errorf("failed to save objective to store: %w", err)
	}

	// Save to filesystem if it has a source
	if obj.Source != "" {
		// If the source is not within our base path, save a copy in our base path
		if !strings.HasPrefix(obj.Source, m.fsBasePath) {
			// Create a filename based on the title
			filename := sanitizeFilename(obj.Title) + ".md"
			newSource := filepath.Join(m.fsBasePath, filename)
			obj.Source = newSource
		}

		// Generate markdown content if it's not already set
		content := obj.Content
		if content == "" {
			content = objectiveToMarkdown(obj)
		}

		// Save to file
		if err := os.WriteFile(obj.Source, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to save objective to file: %w", err)
		}
	}

	// Update cache
	m.objectiveCache[obj.ID] = obj

	// Update tag index
	for _, tag := range obj.Tags {
		// Check if the objective is already indexed for this tag
		if !containsString(m.tagsIndex[tag], obj.ID) {
			m.tagsIndex[tag] = append(m.tagsIndex[tag], obj.ID)
		}
	}

	return nil
}

// GetObjective retrieves an objective by ID
func (m *Manager) GetObjective(ctx context.Context, id string) (*Objective, error) {
	m.mu.RLock()
	obj, exists := m.objectiveCache[id]
	m.mu.RUnlock()

	if exists {
		return obj, nil
	}

	// Not in cache, load from store
	obj, err := m.loadObjectiveFromStore(ctx, id)
	if err != nil {
		return nil, err
	}

	// Add to cache
	m.mu.Lock()
	m.objectiveCache[id] = obj
	m.mu.Unlock()

	return obj, nil
}

// LoadObjectiveFromFile loads an objective from a markdown file
func (m *Manager) LoadObjectiveFromFile(ctx context.Context, filePath string) (*Objective, error) {
	parser := NewMarkdownParser(DefaultParseOptions())
	obj, err := parser.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse objective file: %w", err)
	}

	// Save the objective to make it available through the manager
	if err := m.SaveObjective(ctx, obj); err != nil {
		return nil, fmt.Errorf("failed to save parsed objective: %w", err)
	}

	return obj, nil
}

// DeleteObjective removes an objective by ID
func (m *Manager) DeleteObjective(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get objective to retrieve file path
	obj, exists := m.objectiveCache[id]
	if !exists {
		// Try to load it first
		var err error
		obj, err = m.loadObjectiveFromStore(ctx, id)
		if err != nil {
			return fmt.Errorf("objective not found: %w", err)
		}
	}

	// Delete from store
	if err := m.store.Delete(ctx, m.memoryBucket, id); err != nil {
		return fmt.Errorf("failed to delete objective from store: %w", err)
	}

	// Delete from filesystem if it's within our base path
	if obj.Source != "" && strings.HasPrefix(obj.Source, m.fsBasePath) {
		if err := os.Remove(obj.Source); err != nil && !os.IsNotExist(err) {
			// Log but don't fail if the file doesn't exist
			fmt.Printf("Warning: failed to delete objective file %s: %v\n", obj.Source, err)
		}
	}

	// Remove from cache
	delete(m.objectiveCache, id)

	// Update tag index
	for _, tag := range obj.Tags {
		// Remove objective ID from the tag's list
		m.tagsIndex[tag] = removeString(m.tagsIndex[tag], id)
	}

	return nil
}

// ListObjectives returns all objectives
func (m *Manager) ListObjectives(ctx context.Context) ([]*Objective, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var objectives []*Objective
	for _, obj := range m.objectiveCache {
		objectives = append(objectives, obj)
	}

	return objectives, nil
}

// FindObjectivesByTags finds objectives with the given tags
func (m *Manager) FindObjectivesByTags(ctx context.Context, tags []string) ([]*Objective, error) {
	if len(tags) == 0 {
		return nil, fmt.Errorf("at least one tag must be provided")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get IDs from the first tag
	objectiveIDs, exists := m.tagsIndex[tags[0]]
	if !exists {
		return nil, nil // No objectives with the first tag
	}

	// Filter by additional tags if provided
	for i := 1; i < len(tags); i++ {
		tag := tags[i]
		tagIDs, exists := m.tagsIndex[tag]
		if !exists {
			return nil, nil // No objectives with this tag
		}

		// Keep only IDs that are in both lists
		objectiveIDs = intersect(objectiveIDs, tagIDs)
		if len(objectiveIDs) == 0 {
			return nil, nil // No objectives with all tags
		}
	}

	// Retrieve the objectives
	var objectives []*Objective
	for _, id := range objectiveIDs {
		obj, exists := m.objectiveCache[id]
		if exists {
			objectives = append(objectives, obj)
		}
	}

	return objectives, nil
}

// AddTask adds a task to an objective
func (m *Manager) AddTask(ctx context.Context, objectiveID string, task *ObjectiveTask) error {
	if objectiveID == "" {
		return fmt.Errorf("objective ID cannot be empty")
	}

	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the objective
	obj, exists := m.objectiveCache[objectiveID]
	if !exists {
		// Try to load it
		var err error
		obj, err = m.loadObjectiveFromStore(ctx, objectiveID)
		if err != nil {
			return fmt.Errorf("objective not found: %w", err)
		}
	}

	// Add the task to the objective
	obj.Tasks = append(obj.Tasks, task)
	obj.UpdatedAt = time.Now().UTC()

	// Save the updated objective
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal objective: %w", err)
	}

	if err := m.store.Put(ctx, m.memoryBucket, obj.ID, data); err != nil {
		return fmt.Errorf("failed to save objective to store: %w", err)
	}

	// Update cache
	m.objectiveCache[obj.ID] = obj

	return nil
}

// UpdateTaskStatus updates the status of a task
func (m *Manager) UpdateTaskStatus(ctx context.Context, objectiveID, taskID, status string) error {
	if objectiveID == "" || taskID == "" || status == "" {
		return fmt.Errorf("objective ID, task ID, and status cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the objective
	obj, exists := m.objectiveCache[objectiveID]
	if !exists {
		// Try to load it
		var err error
		obj, err = m.loadObjectiveFromStore(ctx, objectiveID)
		if err != nil {
			return fmt.Errorf("objective not found: %w", err)
		}
	}

	// Find the task
	found := false
	for i, task := range obj.Tasks {
		if task.ID == taskID {
			obj.Tasks[i].Status = status
			obj.Tasks[i].UpdatedAt = time.Now().UTC()
			
			// Set completed time if status is "done"
			if status == "done" {
				now := time.Now().UTC()
				obj.Tasks[i].CompletedAt = &now
			} else {
				obj.Tasks[i].CompletedAt = nil
			}
			
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task not found in objective")
	}

	// Save the updated objective
	obj.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal objective: %w", err)
	}

	if err := m.store.Put(ctx, m.memoryBucket, obj.ID, data); err != nil {
		return fmt.Errorf("failed to save objective to store: %w", err)
	}

	// Update cache
	m.objectiveCache[obj.ID] = obj

	return nil
}

// loadObjectiveFromStore loads an objective from the store
func (m *Manager) loadObjectiveFromStore(ctx context.Context, id string) (*Objective, error) {
	data, err := m.store.Get(ctx, m.memoryBucket, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get objective from store: %w", err)
	}

	var obj Objective
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal objective: %w", err)
	}

	return &obj, nil
}

// objectiveToMarkdown converts an objective to markdown format
func objectiveToMarkdown(obj *Objective) string {
	var sb strings.Builder

	// Add title
	sb.WriteString("# " + obj.Title + "\n\n")

	// Add metadata tags
	if obj.Priority != "" {
		sb.WriteString("@priority: " + obj.Priority + "\n")
	}
	if obj.Owner != "" {
		sb.WriteString("@owner: " + obj.Owner + "\n")
	}
	for _, tag := range obj.Tags {
		sb.WriteString("@tag:" + tag + " ")
	}
	if len(obj.Tags) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Add description
	sb.WriteString(obj.Description + "\n\n")

	// Add parts
	for _, part := range obj.Parts {
		sb.WriteString("## " + part.Title + "\n\n")
		sb.WriteString(part.Content + "\n\n")
	}

	// Add tasks if they don't exist in parts
	hasTaskSection := false
	for _, part := range obj.Parts {
		if part.Type == "tasks" || part.Type == "implementation" {
			hasTaskSection = true
			break
		}
	}

	if !hasTaskSection && len(obj.Tasks) > 0 {
		sb.WriteString("## Tasks\n\n")
		for _, task := range obj.Tasks {
			status := " "
			if task.Status == "done" {
				status = "x"
			}
			sb.WriteString("- [" + status + "] " + task.Title + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// sanitizeFilename removes characters that are invalid in filenames
func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"/", "\\", "?", "%", "*", ":", "|", "\"", "<", ">"}
	result := filename

	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Convert spaces to underscores
	result = strings.ReplaceAll(result, " ", "_")

	// Ensure the filename is not too long
	if len(result) > 100 {
		result = result[:100]
	}

	return result
}

// intersect returns the intersection of two string slices
func intersect(a, b []string) []string {
	set := make(map[string]bool)
	for _, item := range a {
		set[item] = true
	}

	var result []string
	for _, item := range b {
		if set[item] {
			result = append(result, item)
		}
	}

	return result
}

// removeString removes a string from a slice
func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}