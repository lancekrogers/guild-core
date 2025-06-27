// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/storage"
)

// testRegistry implements kanban.ComponentRegistry for testing
type testRegistry struct {
	storageReg kanban.StorageRegistry
}

func (t *testRegistry) Storage() kanban.StorageRegistry {
	return t.storageReg
}

// testStorageRegistry implements kanban.StorageRegistry for testing
type testStorageRegistry struct {
	sqliteReg storage.StorageRegistry
	memStore  memory.Store // Use memory.Store directly
}

func (t *testStorageRegistry) GetKanbanCampaignRepository() kanban.CampaignRepository {
	return &testCampaignRepo{repo: t.sqliteReg.GetCampaignRepository()}
}

func (t *testStorageRegistry) GetKanbanCommissionRepository() kanban.CommissionRepository {
	return &testCommissionRepo{repo: t.sqliteReg.GetCommissionRepository()}
}

func (t *testStorageRegistry) GetBoardRepository() kanban.BoardRepository {
	return &testBoardRepo{repo: t.sqliteReg.GetBoardRepository()}
}

func (t *testStorageRegistry) GetKanbanTaskRepository() kanban.TaskRepository {
	return &testTaskRepo{repo: t.sqliteReg.GetTaskRepository()}
}

func (t *testStorageRegistry) GetMemoryStore() kanban.MemoryStore {
	// The kanban manager expects this to be castable to memory.Store
	// Since memory.Store implements all methods of kanban.MemoryStore, we can return it directly
	return t.memStore
}

// Repository adapters
type testCampaignRepo struct{ repo storage.CampaignRepository }

func (r *testCampaignRepo) CreateCampaign(ctx context.Context, campaign interface{}) error {
	if c, ok := campaign.(map[string]interface{}); ok {
		return r.repo.CreateCampaign(ctx, &storage.Campaign{
			ID:        c["ID"].(string),
			Name:      c["Name"].(string),
			Status:    c["Status"].(string),
			CreatedAt: c["CreatedAt"].(time.Time),
			UpdatedAt: c["UpdatedAt"].(time.Time),
		})
	}
	return nil
}

type testCommissionRepo struct{ repo storage.CommissionRepository }

func (r *testCommissionRepo) CreateCommission(ctx context.Context, commission interface{}) error {
	if c, ok := commission.(map[string]interface{}); ok {
		desc := c["Title"].(string)
		return r.repo.CreateCommission(ctx, &storage.Commission{
			ID:          c["ID"].(string),
			CampaignID:  c["CampaignID"].(string),
			Title:       c["Title"].(string),
			Description: &desc,
			Status:      c["Status"].(string),
			CreatedAt:   c["CreatedAt"].(time.Time),
		})
	}
	return nil
}

func (r *testCommissionRepo) GetCommission(ctx context.Context, id string) (interface{}, error) {
	return r.repo.GetCommission(ctx, id)
}

type testBoardRepo struct{ repo storage.BoardRepository }

func (r *testBoardRepo) CreateBoard(ctx context.Context, board interface{}) error {
	if b, ok := board.(*storage.Board); ok {
		return r.repo.CreateBoard(ctx, b)
	}
	return nil
}

func (r *testBoardRepo) GetBoard(ctx context.Context, id string) (interface{}, error) {
	return r.repo.GetBoard(ctx, id)
}

func (r *testBoardRepo) UpdateBoard(ctx context.Context, board interface{}) error {
	if b, ok := board.(*storage.Board); ok {
		return r.repo.UpdateBoard(ctx, b)
	}
	return nil
}

func (r *testBoardRepo) DeleteBoard(ctx context.Context, id string) error {
	return r.repo.DeleteBoard(ctx, id)
}

func (r *testBoardRepo) ListBoards(ctx context.Context) ([]interface{}, error) {
	boards, err := r.repo.ListBoards(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(boards))
	for i, b := range boards {
		result[i] = b
	}
	return result, nil
}

type testTaskRepo struct{ repo storage.TaskRepository }

func (r *testTaskRepo) CreateTask(ctx context.Context, task interface{}) error { return nil }

func (r *testTaskRepo) UpdateTask(ctx context.Context, task interface{}) error { return nil }

func (r *testTaskRepo) DeleteTask(ctx context.Context, id string) error { return nil }

func (r *testTaskRepo) ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (r *testTaskRepo) RecordTaskEvent(ctx context.Context, event interface{}) error { return nil }

func setupTestKanbanManager(t *testing.T) (*kanban.Manager, *kanban.Board, func()) {
	ctx := context.Background()

	// Initialize SQLite storage for tests
	storageReg, memoryStoreAdapter, err := storage.InitializeSQLiteStorageForTests(ctx)
	require.NoError(t, err)

	// Create default campaign
	campaignRepo := storageReg.GetCampaignRepository()
	defaultCampaign := &storage.Campaign{
		ID:        "test-campaign",
		Name:      "Test Campaign",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = campaignRepo.CreateCampaign(ctx, defaultCampaign)
	require.NoError(t, err)

	// Cast memory store adapter to memory.Store
	var memStore memory.Store
	if memoryStoreAdapter != nil {
		// Cast to memory.Store interface
		if memStoreImpl, ok := memoryStoreAdapter.(memory.Store); ok {
			memStore = memStoreImpl
		}
	}

	// Create test registry adapter
	testReg := &testRegistry{
		storageReg: &testStorageRegistry{
			sqliteReg: storageReg,
			memStore:  memStore,
		},
	}

	// Create manager with registry
	mgr, err := kanban.NewManagerWithRegistry(context.Background(), testReg)
	require.NoError(t, err)

	// Create test board
	board, err := mgr.CreateBoard(context.Background(), "Test Board", "Test Description")
	require.NoError(t, err)

	cleanup := func() {
		// SQLite in-memory DB will be cleaned up automatically
	}

	return mgr, board, cleanup
}

func TestScribeModel_New(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()

	model := New(context.Background(), mgr, board.ID)

	assert.NotNil(t, model)
	assert.Equal(t, board.ID, model.boardID)
	assert.Equal(t, 5, len(model.columns))
	assert.Equal(t, 0, model.viewport.FocusedColumn)
	assert.False(t, model.viewport.SearchMode)
}

func TestScribeModel_Init(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()

	model := New(context.Background(), mgr, board.ID)
	cmd := model.Init()

	assert.NotNil(t, cmd)
	// Should return a batch command with loadTasks and ticker
}

func TestScribeModel_ColumnNavigation(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()

	model := New(context.Background(), mgr, board.ID)

	tests := []struct {
		name     string
		key      string
		expected int
	}{
		{"Move right", "l", 1},
		{"Move left from 1", "h", 0},
		{"Jump to column 3", "3", 2},
		{"Jump to column 5", "5", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to column 1 for left movement test
			if tt.key == "h" {
				model.viewport.FocusedColumn = 1
			}

			updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			m := updatedModel.(*Model)
			assert.Equal(t, tt.expected, m.viewport.FocusedColumn)
		})
	}
}

func TestScribeModel_SearchMode(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()

	model := New(context.Background(), mgr, board.ID)

	// Enter search mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m := updatedModel.(*Model)
	assert.True(t, m.viewport.SearchMode)
	assert.Equal(t, "", m.viewport.SearchFilter)

	// Type search term
	searchTerm := "test"
	for _, ch := range searchTerm {
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = updatedModel.(*Model)
	}
	assert.Equal(t, searchTerm, m.viewport.SearchFilter)

	// Exit search mode
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updatedModel.(*Model)
	assert.False(t, m.viewport.SearchMode)
	assert.Equal(t, "", m.viewport.SearchFilter)
}

func TestScribeModel_TaskSorting(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()

	model := New(context.Background(), mgr, board.ID)

	// Create test tasks
	tasks := []*kanban.Task{
		{
			ID:        "1",
			Title:     "Low priority new task",
			Priority:  kanban.PriorityLow,
			CreatedAt: time.Now(),
		},
		{
			ID:        "2",
			Title:     "High priority task",
			Priority:  kanban.PriorityHigh,
			CreatedAt: time.Now().Add(-time.Hour),
		},
		{
			ID:        "3",
			Title:     "Medium priority old task",
			Priority:  kanban.PriorityMedium,
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
	}

	sorted := model.sortTasks(tasks)

	// Should be sorted by priority (high first), then by age
	assert.Equal(t, "2", sorted[0].ID) // High priority
	assert.Equal(t, "3", sorted[1].ID) // Medium priority, older
	assert.Equal(t, "1", sorted[2].ID) // Low priority
}

func TestScribeModel_ViewportScrolling(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()

	model := New(context.Background(), mgr, board.ID)
	model.viewport.VisibleRows = 5

	// Set up column with many tasks
	model.columns[0].TotalTasks = 20

	// Test scrolling down
	model.scrollColumn(1)
	assert.Equal(t, 1, model.columns[0].ScrollOffset)

	// Test page down
	model.scrollColumn(5)
	assert.Equal(t, 6, model.columns[0].ScrollOffset)

	// Test scrolling up
	model.scrollColumn(-2)
	assert.Equal(t, 4, model.columns[0].ScrollOffset)

	// Test boundary - can't scroll past 0
	model.scrollColumn(-10)
	assert.Equal(t, 0, model.columns[0].ScrollOffset)

	// Test boundary - can't scroll past max
	model.columns[0].ScrollOffset = 15
	model.scrollColumn(10)
	assert.Equal(t, 15, model.columns[0].ScrollOffset) // Max is 20-5=15
}

func TestScribeModel_WindowResize(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()

	model := New(context.Background(), mgr, board.ID)

	// Simulate window resize
	newWidth := 120
	newHeight := 40

	updatedModel, _ := model.Update(tea.WindowSizeMsg{
		Width:  newWidth,
		Height: newHeight,
	})
	m := updatedModel.(*Model)

	assert.Equal(t, newWidth, m.viewport.Width)
	assert.Equal(t, newHeight, m.viewport.Height)

	// Check visible rows calculation
	expectedRows := newHeight - 4 - 3 - 2 // header - columns - bottom
	assert.Equal(t, expectedRows, m.viewport.VisibleRows)
}
