package commission

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// Manager handles storage and retrieval of commissions
type Manager struct {
	commissionRepo  storage.CommissionRepository
	fsBasePath      string
	tagsIndex       map[string][]string // tag -> commission IDs
	commissionCache map[string]*Commission
	mu              sync.RWMutex
}

// newManager creates a new commission manager (private constructor)
func newManager(commissionRepo storage.CommissionRepository, fsBasePath string) (*Manager, error) {
	if commissionRepo == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "commission repository cannot be nil", nil).
			WithComponent("CommissionManager").
			WithOperation("NewManager")
	}

	if fsBasePath == "" {
		// Default to current directory if not specified
		var err error
		fsBasePath, err = os.Getwd()
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get working directory").
				WithComponent("CommissionManager").
				WithOperation("newManager")
		}
		fsBasePath = filepath.Join(fsBasePath, "commissions")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(fsBasePath, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commissions directory").
			WithComponent("CommissionManager").
			WithOperation("NewManager").
			WithDetails("fs_base_path", fsBasePath)
	}

	return &Manager{
		commissionRepo:  commissionRepo,
		fsBasePath:      fsBasePath,
		tagsIndex:       make(map[string][]string),
		commissionCache: make(map[string]*Commission),
	}, nil
}

// DefaultCommissionManagerFactory creates a commission manager for registry use
func DefaultCommissionManagerFactory(commissionRepo storage.CommissionRepository, fsBasePath string) (CommissionManager, error) {
	return newManager(commissionRepo, fsBasePath)
}

// Init initializes the manager (currently a no-op due to repository limitations)
func (m *Manager) Init(ctx context.Context) error {
	// Note: The current CommissionRepository interface doesn't have a generic ListCommissions method
	// We would need to modify the interface to support loading all commissions for cache initialization
	// For now, the cache will be populated on-demand as commissions are accessed
	return nil
}

// SaveCommission stores a commission both in database and filesystem
func (m *Manager) SaveCommission(ctx context.Context, commission *Commission) error {
	if commission == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "commission cannot be nil", nil).
			WithComponent("CommissionManager").
			WithOperation("SaveCommission")
	}

	if commission.ID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "commission ID cannot be empty", nil).
			WithComponent("CommissionManager").
			WithOperation("SaveCommission")
	}

	// Set updated time
	commission.UpdatedAt = time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert to storage model and save
	storageCommission := commissionToStorageCommission(commission)
	if err := m.commissionRepo.CreateCommission(ctx, storageCommission); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission to database").
			WithComponent("CommissionManager").
			WithOperation("SaveCommission").
			WithDetails("commission_id", commission.ID)
	}

	// Save to filesystem if it has a source
	if commission.Source != "" {
		// If the source is not within our base path, save a copy in our base path
		if !strings.HasPrefix(commission.Source, m.fsBasePath) {
			// Create a filename based on the title
			filename := sanitizeFilename(commission.Title) + ".md"
			newSource := filepath.Join(m.fsBasePath, filename)
			commission.Source = newSource
		}

		// Generate markdown content if it's not already set
		content := commission.Content
		if content == "" {
			content = commissionToMarkdown(commission)
		}

		// Save to file
		if err := os.WriteFile(commission.Source, []byte(content), 0644); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission to file").
				WithComponent("CommissionManager").
				WithOperation("SaveCommission").
				WithDetails("source_file", commission.Source)
		}
	}

	// Update cache
	m.commissionCache[commission.ID] = commission

	// Update tag index
	for _, tag := range commission.Tags {
		// Check if the commission is already indexed for this tag
		if !containsString(m.tagsIndex[tag], commission.ID) {
			m.tagsIndex[tag] = append(m.tagsIndex[tag], commission.ID)
		}
	}

	return nil
}

// CreateCommission creates a new commission
func (m *Manager) CreateCommission(ctx context.Context, commission Commission) (*Commission, error) {
	// Generate ID if not provided
	if commission.ID == "" {
		commission.ID = generateCommissionID(commission.Title)
	}

	// Set timestamps
	now := time.Now().UTC()
	commission.CreatedAt = now
	commission.UpdatedAt = now

	// Set default status if not provided
	if commission.Status == "" {
		commission.Status = StatusDraft
	}

	// Save the commission
	if err := m.SaveCommission(ctx, &commission); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commission").
			WithComponent("CommissionManager").
			WithOperation("CreateCommission")
	}

	return &commission, nil
}

// GetCommission retrieves a commission by ID
func (m *Manager) GetCommission(ctx context.Context, id string) (*Commission, error) {
	m.mu.RLock()
	commission, exists := m.commissionCache[id]
	m.mu.RUnlock()

	if exists {
		return commission, nil
	}

	// Not in cache, load from database
	storageCommission, err := m.commissionRepo.GetCommission(ctx, id)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get commission from database").
			WithComponent("CommissionManager").
			WithOperation("GetCommission").
			WithDetails("commission_id", id)
	}

	commission = storageCommissionToCommission(storageCommission)

	// Add to cache
	m.mu.Lock()
	m.commissionCache[id] = commission
	m.mu.Unlock()

	return commission, nil
}

// LoadCommissionFromFile loads a commission from a markdown file
func (m *Manager) LoadCommissionFromFile(ctx context.Context, filePath string) (*Commission, error) {
	parser := NewMarkdownParser(DefaultParseOptions())
	commission, err := parser.ParseFile(filePath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to parse commission file").
			WithComponent("CommissionManager").
			WithOperation("LoadCommissionFromFile").
			WithDetails("file_path", filePath)
	}

	// Save the commission to make it available through the manager
	if err := m.SaveCommission(ctx, commission); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save parsed commission").
			WithComponent("CommissionManager").
			WithOperation("LoadCommissionFromFile")
	}

	return commission, nil
}

// DeleteCommission removes a commission by ID
func (m *Manager) DeleteCommission(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get commission to retrieve file path
	commission, exists := m.commissionCache[id]
	if !exists {
		// Try to load it first
		dbCommission, err := m.commissionRepo.GetCommission(ctx, id)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "commission not found").
				WithComponent("CommissionManager").
				WithOperation("DeleteCommission").
				WithDetails("commission_id", id)
		}
		commission = storageCommissionToCommission(dbCommission)
	}

	// Delete from database
	if err := m.commissionRepo.DeleteCommission(ctx, id); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete commission from database").
			WithComponent("CommissionManager").
			WithOperation("DeleteCommission").
			WithDetails("commission_id", id)
	}

	// Delete from filesystem if it's within our base path
	if commission.Source != "" && strings.HasPrefix(commission.Source, m.fsBasePath) {
		if err := os.Remove(commission.Source); err != nil && !os.IsNotExist(err) {
			// Log but don't fail if the file doesn't exist
			fmt.Printf("Warning: failed to delete commission file %s: %v\n", commission.Source, err)
		}
	}

	// Remove from cache
	delete(m.commissionCache, id)

	// Update tag index
	for _, tag := range commission.Tags {
		// Remove commission ID from the tag's list
		m.tagsIndex[tag] = removeString(m.tagsIndex[tag], id)
	}

	return nil
}

// ListCommissions returns all commissions
func (m *Manager) ListCommissions(ctx context.Context) ([]*Commission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var commissions []*Commission
	for _, commission := range m.commissionCache {
		commissions = append(commissions, commission)
	}

	return commissions, nil
}

// FindCommissionsByTags finds commissions with the given tags
func (m *Manager) FindCommissionsByTags(ctx context.Context, tags []string) ([]*Commission, error) {
	if len(tags) == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "at least one tag must be provided", nil).
			WithComponent("CommissionManager").
			WithOperation("FindCommissionsByTags")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get IDs from the first tag
	commissionIDs, exists := m.tagsIndex[tags[0]]
	if !exists {
		return nil, nil // No commissions with the first tag
	}

	// Filter by additional tags if provided
	for i := 1; i < len(tags); i++ {
		tag := tags[i]
		tagIDs, exists := m.tagsIndex[tag]
		if !exists {
			return nil, nil // No commissions with this tag
		}

		// Keep only IDs that are in both lists
		commissionIDs = intersect(commissionIDs, tagIDs)
		if len(commissionIDs) == 0 {
			return nil, nil // No commissions with all tags
		}
	}

	// Retrieve the commissions
	var commissions []*Commission
	for _, id := range commissionIDs {
		commission, exists := m.commissionCache[id]
		if exists {
			commissions = append(commissions, commission)
		}
	}

	return commissions, nil
}

// AddTask adds a task to a commission
func (m *Manager) AddTask(ctx context.Context, commissionID string, task *CommissionTask) error {
	if commissionID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "commission ID cannot be empty", nil).
			WithComponent("CommissionManager").
			WithOperation("AddTask")
	}

	if task == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "task cannot be nil", nil).
			WithComponent("CommissionManager").
			WithOperation("AddTask")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the commission
	commission, exists := m.commissionCache[commissionID]
	if !exists {
		// Try to load it
		dbCommission, err := m.commissionRepo.GetCommission(ctx, commissionID)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "commission not found").
				WithComponent("CommissionManager").
				WithOperation("AddTask").
				WithDetails("commission_id", commissionID)
		}
		commission = storageCommissionToCommission(dbCommission)
	}

	// Add the task to the commission
	commission.Tasks = append(commission.Tasks, task)
	commission.UpdatedAt = time.Now().UTC()

	// Save the updated commission (note: using CreateCommission as a workaround since UpdateCommission doesn't exist)
	storageCommission := commissionToStorageCommission(commission)
	if err := m.commissionRepo.CreateCommission(ctx, storageCommission); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission to database").
			WithComponent("CommissionManager").
			WithOperation("AddTask").
			WithDetails("commission_id", commissionID)
	}

	// Update cache
	m.commissionCache[commission.ID] = commission

	return nil
}

// UpdateTaskStatus updates the status of a task
func (m *Manager) UpdateTaskStatus(ctx context.Context, commissionID, taskID, status string) error {
	if commissionID == "" || taskID == "" || status == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "commission ID, task ID, and status cannot be empty", nil).
			WithComponent("CommissionManager").
			WithOperation("UpdateTaskStatus")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the commission
	commission, exists := m.commissionCache[commissionID]
	if !exists {
		// Try to load it
		dbCommission, err := m.commissionRepo.GetCommission(ctx, commissionID)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "commission not found").
				WithComponent("CommissionManager").
				WithOperation("UpdateTaskStatus").
				WithDetails("commission_id", commissionID)
		}
		commission = storageCommissionToCommission(dbCommission)
	}

	// Find the task
	found := false
	for i, task := range commission.Tasks {
		if task.ID == taskID {
			commission.Tasks[i].Status = status
			commission.Tasks[i].UpdatedAt = time.Now().UTC()

			// Set completed time if status is "done"
			if status == "done" {
				now := time.Now().UTC()
				commission.Tasks[i].CompletedAt = &now
			} else {
				commission.Tasks[i].CompletedAt = nil
			}

			found = true
			break
		}
	}

	if !found {
		return gerror.New(gerror.ErrCodeNotFound, "task not found in commission", nil).
			WithComponent("CommissionManager").
			WithOperation("UpdateTaskStatus").
			WithDetails("commission_id", commissionID).
			WithDetails("task_id", taskID)
	}

	// Save the updated commission (note: using CreateCommission as a workaround since UpdateCommission doesn't exist)
	commission.UpdatedAt = time.Now().UTC()
	storageCommission := commissionToStorageCommission(commission)
	if err := m.commissionRepo.CreateCommission(ctx, storageCommission); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission to database").
			WithComponent("CommissionManager").
			WithOperation("UpdateTaskStatus").
			WithDetails("commission_id", commissionID)
	}

	// Update cache
	m.commissionCache[commission.ID] = commission

	return nil
}

// GetCommissionsByTag retrieves all commissions that have a specific tag
func (m *Manager) GetCommissionsByTag(ctx context.Context, tag string) ([]*Commission, error) {
	if tag == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "tag cannot be empty", nil).
			WithComponent("CommissionManager").
			WithOperation("GetCommissionsByTag")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get commission IDs with this tag
	commissionIDs, exists := m.tagsIndex[tag]
	if !exists || len(commissionIDs) == 0 {
		return []*Commission{}, nil
	}

	// Retrieve the commissions
	var commissions []*Commission
	for _, id := range commissionIDs {
		commission, exists := m.commissionCache[id]
		if exists {
			commissions = append(commissions, commission)
		} else {
			// Try to load from database if not in cache
			dbCommission, err := m.commissionRepo.GetCommission(ctx, id)
			if err == nil {
				commission = storageCommissionToCommission(dbCommission)
				m.commissionCache[id] = commission
				commissions = append(commissions, commission)
			}
		}
	}

	return commissions, nil
}

// UpdateCommission updates an existing commission
func (m *Manager) UpdateCommission(ctx context.Context, commission Commission) error {
	if commission.ID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "commission ID cannot be empty", nil).
			WithComponent("CommissionManager").
			WithOperation("UpdateCommission")
	}

	// Set updated time
	commission.UpdatedAt = time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if commission exists
	_, exists := m.commissionCache[commission.ID]
	if !exists {
		// Try to load it to verify existence
		_, err := m.commissionRepo.GetCommission(ctx, commission.ID)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeNotFound, "commission not found").
				WithComponent("CommissionManager").
				WithOperation("UpdateCommission").
				WithDetails("commission_id", commission.ID)
		}
	}

	// Convert to storage model and update
	storageCommission := commissionToStorageCommission(&commission)
	// Note: Using CreateCommission as UpdateCommission is not available in the interface
	// This will upsert the commission
	if err := m.commissionRepo.CreateCommission(ctx, storageCommission); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update commission in database").
			WithComponent("CommissionManager").
			WithOperation("UpdateCommission").
			WithDetails("commission_id", commission.ID)
	}

	// Update filesystem if source exists
	if commission.Source != "" && strings.HasPrefix(commission.Source, m.fsBasePath) {
		// Generate markdown content if it's not already set
		content := commission.Content
		if content == "" {
			content = commissionToMarkdown(&commission)
		}

		// Save to file
		if err := os.WriteFile(commission.Source, []byte(content), 0644); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update commission file").
				WithComponent("CommissionManager").
				WithOperation("UpdateCommission").
				WithDetails("source_file", commission.Source)
		}
	}

	// Update cache
	m.commissionCache[commission.ID] = &commission

	// Update tag index
	for _, tag := range commission.Tags {
		// Check if the commission is already indexed for this tag
		if !containsString(m.tagsIndex[tag], commission.ID) {
			m.tagsIndex[tag] = append(m.tagsIndex[tag], commission.ID)
		}
	}

	return nil
}

// SetCommission updates the current active commission
func (m *Manager) SetCommission(ctx context.Context, commissionID string) error {
	if commissionID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "commission ID cannot be empty", nil).
			WithComponent("CommissionManager").
			WithOperation("SetCommission")
	}

	// Verify the commission exists
	_, err := m.GetCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "commission not found").
			WithComponent("CommissionManager").
			WithOperation("SetCommission").
			WithDetails("commission_id", commissionID)
	}

	// In a real implementation, this might update a "current commission" state
	// For now, we just verify it exists
	return nil
}

// Conversion functions between Commission and database models

// storageCommissionToCommission converts a storage commission to a Commission object
func storageCommissionToCommission(storageCommission *storage.Commission) *Commission {
	commission := &Commission{
		ID:         storageCommission.ID,
		Title:      storageCommission.Title,
		Status:     CommissionStatus(storageCommission.Status),
		CampaignID: storageCommission.CampaignID,
		Tags:       []string{},
		Metadata:   make(map[string]string),
		Context:    []string{},
		Parts:      []*CommissionPart{},
		Tasks:      []*CommissionTask{},
		Priority:   "medium",
	}

	if storageCommission.Description != nil {
		commission.Description = *storageCommission.Description
	}
	if storageCommission.Domain != nil {
		commission.Goal = *storageCommission.Domain
	}
	commission.CreatedAt = storageCommission.CreatedAt

	// Parse context if it exists
	if storageCommission.Context != nil {
		// Context is already map[string]interface{} in storage layer
		contextData := storageCommission.Context
		// Extract tags, metadata, parts, tasks from context
		if tags, ok := contextData["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					commission.Tags = append(commission.Tags, tagStr)
				}
			}
		}
		if priority, ok := contextData["priority"].(string); ok {
			commission.Priority = priority
		}
		if owner, ok := contextData["owner"].(string); ok {
			commission.Owner = owner
		}
	}

	return commission
}

// commissionToStorageCommission converts a Commission object to a storage commission
func commissionToStorageCommission(commission *Commission) *storage.Commission {
	storageCommission := &storage.Commission{
		ID:         commission.ID,
		CampaignID: commission.CampaignID,
		Title:      commission.Title,
		Status:     string(commission.Status),
	}

	if commission.Description != "" {
		storageCommission.Description = &commission.Description
	}
	if commission.Goal != "" {
		storageCommission.Domain = &commission.Goal
	}
	storageCommission.CreatedAt = commission.CreatedAt

	// Create context with tags, metadata, parts, tasks
	contextData := map[string]interface{}{
		"tags":     commission.Tags,
		"priority": commission.Priority,
		"owner":    commission.Owner,
		"metadata": commission.Metadata,
		"parts":    commission.Parts,
		"tasks":    commission.Tasks,
	}
	storageCommission.Context = contextData

	return storageCommission
}

// Legacy method names for backward compatibility
func (m *Manager) SaveObjective(ctx context.Context, obj *Commission) error {
	return m.SaveCommission(ctx, obj)
}

func (m *Manager) GetObjective(ctx context.Context, id string) (*Commission, error) {
	return m.GetCommission(ctx, id)
}

func (m *Manager) LoadObjectiveFromFile(ctx context.Context, filePath string) (*Commission, error) {
	return m.LoadCommissionFromFile(ctx, filePath)
}

func (m *Manager) DeleteObjective(ctx context.Context, id string) error {
	return m.DeleteCommission(ctx, id)
}

func (m *Manager) ListObjectives(ctx context.Context) ([]*Commission, error) {
	return m.ListCommissions(ctx)
}

func (m *Manager) FindObjectivesByTags(ctx context.Context, tags []string) ([]*Commission, error) {
	return m.FindCommissionsByTags(ctx, tags)
}

// commissionToMarkdown converts a commission to markdown format
func commissionToMarkdown(commission *Commission) string {
	var sb strings.Builder

	// Add title
	sb.WriteString("# " + commission.Title + "\n\n")

	// Add metadata tags
	if commission.Priority != "" {
		sb.WriteString("@priority: " + commission.Priority + "\n")
	}
	if commission.Owner != "" {
		sb.WriteString("@owner: " + commission.Owner + "\n")
	}
	for _, tag := range commission.Tags {
		sb.WriteString("@tag:" + tag + " ")
	}
	if len(commission.Tags) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Add description
	sb.WriteString(commission.Description + "\n\n")

	// Add parts
	for _, part := range commission.Parts {
		sb.WriteString("## " + part.Title + "\n\n")
		sb.WriteString(part.Content + "\n\n")
	}

	// Add tasks if they don't exist in parts
	hasTaskSection := false
	for _, part := range commission.Parts {
		if part.Type == "tasks" || part.Type == "implementation" {
			hasTaskSection = true
			break
		}
	}

	if !hasTaskSection && len(commission.Tasks) > 0 {
		sb.WriteString("## Tasks\n\n")
		for _, task := range commission.Tasks {
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

// containsString checks if a string is in a slice
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// generateCommissionID generates a unique ID for a commission based on its title
func generateCommissionID(title string) string {
	// Use sanitized title with timestamp for uniqueness
	base := sanitizeFilename(title)
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d", base, timestamp)
}
