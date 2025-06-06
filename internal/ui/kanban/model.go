package kanban

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// Column represents a kanban board column
type Column struct {
	Status      kanban.TaskStatus
	Title       string
	Tasks       []*kanban.Task
	ScrollOffset int  // Current scroll position
	TotalTasks   int  // Total tasks in this column (may be > len(Tasks) due to viewport)
}

// ViewportState manages the visible portion of the board
type ViewportState struct {
	Width           int      // Terminal width
	Height          int      // Terminal height
	VisibleRows     int      // Number of task rows that fit
	FocusedColumn   int      // Currently selected column (0-4)
	SearchFilter    string   // Active search term
	SearchMode      bool     // Whether search mode is active
}

// Model represents the kanban board UI state
type Model struct {
	// Dependencies
	kanbanManager *kanban.Manager
	boardID       string
	ctx           context.Context
	
	// UI State
	columns       [5]Column
	viewport      ViewportState
	
	// Task cache
	taskCache     map[string][]*kanban.Task // Status -> Tasks
	lastUpdate    time.Time
	loading       bool
	error         error
	
	// Interaction state
	selectedTaskID string
	showHelp       bool
	statusMessage  string
	
	// Performance tracking
	frameCount     int
	lastRender     time.Time
	fps            float64
}

// New creates a new kanban board UI model
func New(ctx context.Context, kanbanManager *kanban.Manager, boardID string) *Model {
	m := &Model{
		ctx:           ctx,
		kanbanManager: kanbanManager,
		boardID:       boardID,
		taskCache:     make(map[string][]*kanban.Task),
		lastRender:    time.Now(),
		viewport: ViewportState{
			Width:        80,
			Height:       24,
			VisibleRows:  10,
			FocusedColumn: 0,
		},
	}
	
	// Initialize columns
	m.columns = [5]Column{
		{Status: kanban.StatusTodo, Title: "TODO"},
		{Status: kanban.StatusInProgress, Title: "IN PROGRESS"},
		{Status: kanban.StatusBlocked, Title: "BLOCKED"},
		{Status: kanban.StatusReadyForReview, Title: "READY FOR REVIEW"},
		{Status: kanban.StatusDone, Title: "DONE"},
	}
	
	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTasks(),
		tea.Every(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

// Message types
type tickMsg time.Time
type tasksLoadedMsg struct {
	tasks map[string][]*kanban.Task
}
type errorMsg struct{ err error }
type taskUpdatedMsg struct{ task *kanban.Task }

// loadTasks loads tasks from the kanban manager
func (m *Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		board, err := m.kanbanManager.GetBoard(m.ctx, m.boardID)
		if err != nil {
			return errorMsg{err}
		}
		
		// Load tasks for each column status
		taskCache := make(map[string][]*kanban.Task)
		for _, col := range m.columns {
			tasks, err := board.GetTasksByStatus(m.ctx, col.Status)
			if err != nil {
				return errorMsg{err}
			}
			
			// Sort tasks by priority and age
			tasks = m.sortTasks(tasks)
			taskCache[string(col.Status)] = tasks
		}
		
		return tasksLoadedMsg{tasks: taskCache}
	}
}

// sortTasks sorts tasks by priority and age
func (m *Model) sortTasks(tasks []*kanban.Task) []*kanban.Task {
	// TODO: Implement smart prioritization as per design doc:
	// 1. Blocking urgency - Tasks blocking other work
	// 2. Priority level - High/Medium/Low
	// 3. Age - Older tasks bubble up
	// 4. Agent availability - Tasks with available agents
	
	// For now, simple priority sort
	sorted := make([]*kanban.Task, len(tasks))
	copy(sorted, tasks)
	
	// Sort by priority (high first), then by age (older first)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			shouldSwap := false
			
			// Compare priorities
			if sorted[i].Priority == kanban.PriorityLow && sorted[j].Priority != kanban.PriorityLow {
				shouldSwap = true
			} else if sorted[i].Priority == kanban.PriorityMedium && sorted[j].Priority == kanban.PriorityHigh {
				shouldSwap = true
			} else if sorted[i].Priority == sorted[j].Priority {
				// Same priority, sort by age
				if sorted[i].CreatedAt.After(sorted[j].CreatedAt) {
					shouldSwap = true
				}
			}
			
			if shouldSwap {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.viewport.SearchMode {
			return m.handleSearchMode(msg)
		}
		return m.handleNormalMode(msg)
		
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		m.calculateVisibleRows()
		
	case tickMsg:
		// Update FPS
		now := time.Now()
		elapsed := now.Sub(m.lastRender).Seconds()
		if elapsed > 0 {
			m.fps = 1.0 / elapsed
		}
		m.lastRender = now
		m.frameCount++
		
		// Periodic refresh
		if time.Since(m.lastUpdate) > 5*time.Second {
			cmds = append(cmds, m.loadTasks())
		}
		
	case tasksLoadedMsg:
		m.loading = false
		m.taskCache = msg.tasks
		m.lastUpdate = time.Now()
		m.updateColumns()
		
	case errorMsg:
		m.loading = false
		m.error = msg.err
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
	}
	
	return m, tea.Batch(cmds...)
}

// handleNormalMode handles key presses in normal mode
func (m *Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
		
	case "j", "down":
		m.scrollColumn(1)
		
	case "k", "up":
		m.scrollColumn(-1)
		
	case "J":
		m.scrollColumn(m.viewport.VisibleRows) // Page down
		
	case "K":
		m.scrollColumn(-m.viewport.VisibleRows) // Page up
		
	case "h", "left":
		if m.viewport.FocusedColumn > 0 {
			m.viewport.FocusedColumn--
		}
		
	case "l", "right":
		if m.viewport.FocusedColumn < 4 {
			m.viewport.FocusedColumn++
		}
		
	case "1", "2", "3", "4", "5":
		// Jump to column
		col := int(msg.String()[0] - '1')
		if col >= 0 && col < 5 {
			m.viewport.FocusedColumn = col
		}
		
	case "/":
		m.viewport.SearchMode = true
		m.viewport.SearchFilter = ""
		
	case "?":
		m.showHelp = !m.showHelp
		
	case "r", "R":
		m.statusMessage = "Refreshing..."
		return m, m.loadTasks()
		
	case "enter":
		// Select task for detailed view or action
		if task := m.getSelectedTask(); task != nil {
			m.selectedTaskID = task.ID
			// TODO: Open task detail view
		}
	}
	
	return m, nil
}

// handleSearchMode handles key presses in search mode
func (m *Model) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.viewport.SearchMode = false
		m.viewport.SearchFilter = ""
		m.updateColumns()
		
	case tea.KeyEnter:
		m.viewport.SearchMode = false
		m.updateColumns()
		
	case tea.KeyBackspace:
		if len(m.viewport.SearchFilter) > 0 {
			m.viewport.SearchFilter = m.viewport.SearchFilter[:len(m.viewport.SearchFilter)-1]
			m.updateColumns()
		}
		
	default:
		if msg.Type == tea.KeyRunes {
			m.viewport.SearchFilter += string(msg.Runes)
			m.updateColumns()
		}
	}
	
	return m, nil
}

// scrollColumn scrolls the focused column
func (m *Model) scrollColumn(delta int) {
	col := &m.columns[m.viewport.FocusedColumn]
	newOffset := col.ScrollOffset + delta
	
	// Clamp to valid range
	maxOffset := col.TotalTasks - m.viewport.VisibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	
	if newOffset < 0 {
		newOffset = 0
	} else if newOffset > maxOffset {
		newOffset = maxOffset
	}
	
	col.ScrollOffset = newOffset
}

// updateColumns updates column data from the task cache
func (m *Model) updateColumns() {
	for i := range m.columns {
		status := string(m.columns[i].Status)
		allTasks := m.taskCache[status]
		
		// Apply search filter
		var filteredTasks []*kanban.Task
		if m.viewport.SearchFilter != "" {
			filter := strings.ToLower(m.viewport.SearchFilter)
			for _, task := range allTasks {
				if strings.Contains(strings.ToLower(task.Title), filter) ||
				   strings.Contains(strings.ToLower(task.Description), filter) ||
				   strings.Contains(strings.ToLower(task.AssignedTo), filter) {
					filteredTasks = append(filteredTasks, task)
				}
			}
		} else {
			filteredTasks = allTasks
		}
		
		m.columns[i].TotalTasks = len(filteredTasks)
		
		// Get visible tasks based on scroll offset
		start := m.columns[i].ScrollOffset
		end := start + m.viewport.VisibleRows
		if end > len(filteredTasks) {
			end = len(filteredTasks)
		}
		
		if start < len(filteredTasks) {
			m.columns[i].Tasks = filteredTasks[start:end]
		} else {
			m.columns[i].Tasks = nil
		}
	}
}

// calculateVisibleRows calculates how many task rows fit in the viewport
func (m *Model) calculateVisibleRows() {
	// Account for header, column titles, borders, help line
	headerHeight := 4  // Campaign header + separator
	columnHeight := 3  // Column headers + separator
	bottomHeight := 2  // Help line + border
	
	availableHeight := m.viewport.Height - headerHeight - columnHeight - bottomHeight
	if availableHeight < 1 {
		availableHeight = 1
	}
	
	m.viewport.VisibleRows = availableHeight
}

// getSelectedTask returns the currently highlighted task
func (m *Model) getSelectedTask() *kanban.Task {
	col := &m.columns[m.viewport.FocusedColumn]
	if len(col.Tasks) > 0 {
		// For now, return the first visible task
		// TODO: Add intra-column selection
		return col.Tasks[0]
	}
	return nil
}


// Styles for the kanban board
var (
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)
		
	columnHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Width(20).
		Align(lipgloss.Center)
		
	taskStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Width(20).
		MaxHeight(3)
		
	selectedTaskStyle = taskStyle.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12"))
		
	scrollIndicatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)
)