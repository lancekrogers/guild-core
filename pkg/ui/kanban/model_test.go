package kanban

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestKanbanManager(t *testing.T) (*kanban.Manager, *kanban.Board, func()) {
	// Create temporary store
	tempDB := "/tmp/test-kanban-" + time.Now().Format("20060102-150405") + ".db"
	store, err := boltdb.NewStore(tempDB)
	require.NoError(t, err)
	
	// Create manager
	mgr, err := kanban.NewManager(store)
	require.NoError(t, err)
	
	// Create test board
	board, err := mgr.CreateBoard(context.Background(), "Test Board", "Test Description")
	require.NoError(t, err)
	
	cleanup := func() {
		store.Close()
	}
	
	return mgr, board, cleanup
}

func TestScribeModel_New(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()
	
	model := New(mgr, board.ID)
	
	assert.NotNil(t, model)
	assert.Equal(t, board.ID, model.boardID)
	assert.Equal(t, 5, len(model.columns))
	assert.Equal(t, 0, model.viewport.FocusedColumn)
	assert.False(t, model.viewport.SearchMode)
}

func TestScribeModel_Init(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()
	
	model := New(mgr, board.ID)
	cmd := model.Init()
	
	assert.NotNil(t, cmd)
	// Should return a batch command with loadTasks and ticker
}

func TestScribeModel_ColumnNavigation(t *testing.T) {
	mgr, board, cleanup := setupTestKanbanManager(t)
	defer cleanup()
	
	model := New(mgr, board.ID)
	
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
	
	model := New(mgr, board.ID)
	
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
	
	model := New(mgr, board.ID)
	
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
	
	model := New(mgr, board.ID)
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
	
	model := New(mgr, board.ID)
	
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