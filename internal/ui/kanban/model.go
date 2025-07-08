// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"

	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/kanban"
)

// Column represents a kanban board column
type Column struct {
	Status       kanban.TaskStatus
	Title        string
	Tasks        []*kanban.Task
	ScrollOffset int // Current scroll position
	TotalTasks   int // Total tasks in this column (may be > len(Tasks) due to viewport)
}

// ViewportState manages the visible portion of the board
type ViewportState struct {
	Width         int    // Terminal width
	Height        int    // Terminal height
	VisibleRows   int    // Number of task rows that fit
	FocusedColumn int    // Currently selected column (0-4)
	SearchFilter  string // Active search term
	SearchMode    bool   // Whether search mode is active
}

// Model represents the kanban board UI state
type Model struct {
	// Dependencies
	kanbanManager *kanban.Manager
	boardID       string
	ctx           context.Context

	// UI State
	columns       [5]Column
	viewportState ViewportState

	// Task cache
	taskCache  map[string][]*kanban.Task // Status -> Tasks
	lastUpdate time.Time
	loading    bool
	error      error

	// Interaction state
	selectedTaskID string
	showHelp       bool
	statusMessage  string

	// Performance tracking
	frameCount int
	lastRender time.Time
	fps        float64

	// Performance optimization components
	profiler       *KanbanProfiler
	viewport       *Viewport
	renderer       *OptimizedCardRenderer
	cardCache      *CompactCardCache
	virtualWindow  *VirtualWindow
	lowQualityMode bool

	// Event streaming
	eventClient   pb.EventServiceClient
	eventStream   pb.EventService_StreamEventsClient
	eventChan     chan tea.Msg
	streamContext context.Context
	streamCancel  context.CancelFunc
}

// New creates a new kanban board UI model
func New(ctx context.Context, kanbanManager *kanban.Manager, boardID string) *Model {
	m := &Model{
		ctx:           ctx,
		kanbanManager: kanbanManager,
		boardID:       boardID,
		taskCache:     make(map[string][]*kanban.Task),
		lastRender:    time.Now(),
		eventChan:     make(chan tea.Msg, 100),
		viewportState: ViewportState{
			Width:         80,
			Height:        24,
			VisibleRows:   10,
			FocusedColumn: 0,
		},
		// Initialize performance components
		profiler:      NewKanbanProfiler(60), // Target 60 FPS
		viewport:      NewViewport(80, 24),
		renderer:      NewOptimizedCardRenderer(),
		cardCache:     NewCompactCardCache(1000, 10*time.Minute),
		virtualWindow: NewVirtualWindow(100), // Window of 100 cards
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

// NewWithEventClient creates a new kanban board UI model with event streaming
func NewWithEventClient(ctx context.Context, kanbanManager *kanban.Manager, boardID string, conn *grpc.ClientConn) *Model {
	m := New(ctx, kanbanManager, boardID)

	if conn != nil {
		m.eventClient = pb.NewEventServiceClient(conn)
	}

	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.loadTasks(),
		tea.Every(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	}

	// Start event streaming if client is available
	if m.eventClient != nil {
		cmds = append(cmds, m.startEventStream())
	}

	// Listen for events from the event channel
	cmds = append(cmds, m.waitForEvent())

	return tea.Batch(cmds...)
}

// Message types
type tickMsg time.Time

type tasksLoadedMsg struct {
	tasks map[string][]*kanban.Task
}

type errorMsg struct{ err error }

type taskUpdatedMsg struct{ task *kanban.Task }

// Event stream messages
type eventStreamStartedMsg struct{}

type eventStreamStoppedMsg struct{ err error }

type taskEventMsg struct {
	eventType string
	taskID    string
	boardID   string
	data      map[string]interface{}
}

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

	// Start profiling if enabled
	var profilingDone func()
	if m.profiler != nil && m.profiler.IsEnabled() {
		profilingDone = m.profiler.StartFrame(m.ctx)
		defer profilingDone()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.viewportState.SearchMode {
			return m.handleSearchMode(msg)
		}
		return m.handleNormalMode(msg)

	case tea.WindowSizeMsg:
		m.viewportState.Width = msg.Width
		m.viewportState.Height = msg.Height

		// Update optimized viewport
		err := m.viewport.Resize(m.ctx, msg.Width, msg.Height)
		if err != nil {
			// Log error but continue
		}

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

	case eventStreamStartedMsg:
		m.statusMessage = "🟢 Connected to event stream"

	case eventStreamStoppedMsg:
		m.statusMessage = "🔴 Event stream disconnected"
		if msg.err != nil {
			m.error = msg.err
		}
		// Try to reconnect after a delay
		cmds = append(cmds, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return m.startEventStream()()
		}))

	case taskEventMsg:
		// Handle real-time task event
		m.handleTaskEvent(msg)
		// Continue listening for events
		cmds = append(cmds, m.waitForEvent())
	}

	return m, tea.Batch(cmds...)
}

// handleTaskEvent processes real-time task events
func (m *Model) handleTaskEvent(event taskEventMsg) {
	switch event.eventType {
	case "task.created":
		// Reload tasks for the affected column
		m.statusMessage = fmt.Sprintf("✨ New task created: %s", getStringFromMap(event.data, "title"))
		// Schedule a reload after a short delay to batch multiple events
		m.scheduleReload()

	case "task.moved":
		fromStatus := getStringFromMap(event.data, "from_status")
		toStatus := getStringFromMap(event.data, "to_status")
		m.statusMessage = fmt.Sprintf("📦 Task moved from %s to %s", fromStatus, toStatus)
		m.scheduleReload()

	case "task.updated":
		m.statusMessage = fmt.Sprintf("✏️ Task updated: %s", event.taskID)
		m.scheduleReload()

	case "task.completed":
		m.statusMessage = fmt.Sprintf("✅ Task completed: %s", event.taskID)
		m.scheduleReload()

	case "task.assigned":
		assignee := getStringFromMap(event.data, "assignee")
		m.statusMessage = fmt.Sprintf("👤 Task assigned to %s", assignee)
		m.scheduleReload()

	case "task.blocked":
		reason := getStringFromMap(event.data, "reason")
		m.statusMessage = fmt.Sprintf("🚫 Task blocked: %s", reason)
		m.scheduleReload()

	case "task.unblocked":
		m.statusMessage = fmt.Sprintf("✅ Task unblocked: %s", event.taskID)
		m.scheduleReload()
	}
}

// scheduleReload schedules a task reload if not already scheduled
func (m *Model) scheduleReload() {
	// Only reload if we haven't reloaded recently (debounce)
	if time.Since(m.lastUpdate) > 500*time.Millisecond {
		m.lastUpdate = time.Now()
		// Note: We should trigger loadTasks() here, but we can't return a Cmd from this method
		// Instead, we'll rely on the periodic refresh in tickMsg
	}
}

// handleNormalMode handles key presses in normal mode
func (m *Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.stopEventStream()
		return m, tea.Quit

	case "j", "down":
		m.scrollColumn(1)

	case "k", "up":
		m.scrollColumn(-1)

	case "J":
		m.scrollColumn(m.viewportState.VisibleRows) // Page down

	case "K":
		m.scrollColumn(-m.viewportState.VisibleRows) // Page up

	case "h", "left":
		if m.viewportState.FocusedColumn > 0 {
			m.viewportState.FocusedColumn--
		}

	case "l", "right":
		if m.viewportState.FocusedColumn < 4 {
			m.viewportState.FocusedColumn++
		}

	case "1", "2", "3", "4", "5":
		// Jump to column
		col := int(msg.String()[0] - '1')
		if col >= 0 && col < 5 {
			m.viewportState.FocusedColumn = col
		}

	case "/":
		m.viewportState.SearchMode = true
		m.viewportState.SearchFilter = ""

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
		m.viewportState.SearchMode = false
		m.viewportState.SearchFilter = ""
		m.updateColumns()

	case tea.KeyEnter:
		m.viewportState.SearchMode = false
		m.updateColumns()

	case tea.KeyBackspace:
		if len(m.viewportState.SearchFilter) > 0 {
			m.viewportState.SearchFilter = m.viewportState.SearchFilter[:len(m.viewportState.SearchFilter)-1]
			m.updateColumns()
		}

	default:
		if msg.Type == tea.KeyRunes {
			m.viewportState.SearchFilter += string(msg.Runes)
			m.updateColumns()
		}
	}

	return m, nil
}

// scrollColumn scrolls the focused column
func (m *Model) scrollColumn(delta int) {
	col := &m.columns[m.viewportState.FocusedColumn]
	newOffset := col.ScrollOffset + delta

	// Clamp to valid range
	maxOffset := col.TotalTasks - m.viewportState.VisibleRows
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
		if m.viewportState.SearchFilter != "" {
			filter := strings.ToLower(m.viewportState.SearchFilter)
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
		end := start + m.viewportState.VisibleRows
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
	headerHeight := 4 // Campaign header + separator
	columnHeight := 3 // Column headers + separator
	bottomHeight := 2 // Help line + border

	availableHeight := m.viewportState.Height - headerHeight - columnHeight - bottomHeight
	if availableHeight < 1 {
		availableHeight = 1
	}

	m.viewportState.VisibleRows = availableHeight
}

// getSelectedTask returns the currently highlighted task
func (m *Model) getSelectedTask() *kanban.Task {
	col := &m.columns[m.viewportState.FocusedColumn]
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

// Event streaming methods

// startEventStream starts the gRPC event stream
func (m *Model) startEventStream() tea.Cmd {
	return func() tea.Msg {
		if m.eventClient == nil {
			return nil
		}

		// Create stream context
		m.streamContext, m.streamCancel = context.WithCancel(m.ctx)

		// Create event stream request
		req := &pb.StreamEventsRequest{
			EventTypes: []string{
				"task.created",
				"task.moved",
				"task.updated",
				"task.completed",
				"task.assigned",
				"task.blocked",
				"task.unblocked",
			},
		}

		// Start streaming
		stream, err := m.eventClient.StreamEvents(m.streamContext, req)
		if err != nil {
			return errorMsg{gerror.Wrap(err, gerror.ErrCodeConnection, "failed to start event stream").
				WithComponent("kanban").
				WithOperation("startEventStream")}
		}

		m.eventStream = stream

		// Start goroutine to receive events
		go m.receiveEvents()

		m.statusMessage = "🟢 Connected to event stream"
		return eventStreamStartedMsg{}
	}
}

// receiveEvents receives events from the stream
func (m *Model) receiveEvents() {
	for {
		event, err := m.eventStream.Recv()
		if err != nil {
			// Stream ended
			m.eventChan <- eventStreamStoppedMsg{err: err}
			return
		}

		// Parse event and send to channel
		if event.Type == "task.created" ||
			event.Type == "task.moved" ||
			event.Type == "task.updated" ||
			event.Type == "task.completed" ||
			event.Type == "task.assigned" ||
			event.Type == "task.blocked" ||
			event.Type == "task.unblocked" {

			data := event.Data.AsMap()

			msg := taskEventMsg{
				eventType: event.Type,
				boardID:   getStringFromMap(data, "board_id"),
			}

			// Extract task ID based on event type
			if taskID := getStringFromMap(data, "task_id"); taskID != "" {
				msg.taskID = taskID
			}

			msg.data = data

			// Only send events for our board
			if msg.boardID == m.boardID || msg.boardID == "" {
				m.eventChan <- msg
			}
		}
	}
}

// waitForEvent waits for events from the event channel
func (m *Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		return <-m.eventChan
	}
}

// stopEventStream stops the event stream
func (m *Model) stopEventStream() {
	if m.streamCancel != nil {
		m.streamCancel()
	}
}

// Helper function to safely get string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Performance optimization methods

// EnableProfiling enables performance profiling
func (m *Model) EnableProfiling() {
	if m.profiler != nil {
		m.profiler.Enable()
	}
}

// DisableProfiling disables performance profiling
func (m *Model) DisableProfiling() {
	if m.profiler != nil {
		m.profiler.Disable()
	}
}

// GetPerformanceReport returns current performance metrics
func (m *Model) GetPerformanceReport() (*ProfileReport, error) {
	if m.profiler == nil || !m.profiler.IsEnabled() {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "profiler not available or disabled", nil).
			WithComponent("kanban.model").
			WithOperation("GetPerformanceReport")
	}

	return m.profiler.GenerateReport(m.ctx)
}

// UpdateCardCache updates the compact card cache with current tasks
func (m *Model) updateCardCache() error {
	if m.cardCache == nil {
		return nil
	}

	for _, tasks := range m.taskCache {
		for _, task := range tasks {
			if err := m.cardCache.AddCard(m.ctx, task); err != nil {
				// Log error but continue processing other cards
				continue
			}
		}
	}

	return nil
}

// SetLowQualityMode enables or disables low quality rendering mode
func (m *Model) SetLowQualityMode(enabled bool) error {
	m.lowQualityMode = enabled

	if m.renderer != nil {
		return m.renderer.SetLowQualityMode(m.ctx, enabled)
	}

	return nil
}

// GetMemoryUsage returns estimated memory usage
func (m *Model) GetMemoryUsage() (int64, error) {
	var totalUsage int64

	// Viewport memory usage
	if m.viewport != nil {
		columns := make([]*Column, len(m.columns))
		for i := range m.columns {
			columns[i] = &m.columns[i]
		}

		usage, err := m.viewport.EstimateMemoryUsage(m.ctx, columns)
		if err != nil {
			return 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to estimate viewport memory usage").
				WithComponent("kanban.model").
				WithOperation("GetMemoryUsage")
		}
		totalUsage += usage
	}

	// Virtual window memory usage
	if m.virtualWindow != nil {
		totalUsage += m.virtualWindow.GetMemoryUsage()
	}

	// Task cache rough estimation
	taskCount := 0
	for _, tasks := range m.taskCache {
		taskCount += len(tasks)
	}
	totalUsage += int64(taskCount) * 200 // ~200 bytes per task estimate

	return totalUsage, nil
}

// GetDebugInfo returns debug information for performance analysis
func (m *Model) GetDebugInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Basic model info
	info["columns"] = len(m.columns)
	info["viewport_width"] = m.viewportState.Width
	info["viewport_height"] = m.viewportState.Height
	info["focused_column"] = m.viewportState.FocusedColumn
	info["low_quality_mode"] = m.lowQualityMode

	// Task counts
	totalTasks := 0
	for status, tasks := range m.taskCache {
		count := len(tasks)
		info[fmt.Sprintf("tasks_%s", status)] = count
		totalTasks += count
	}
	info["total_tasks"] = totalTasks

	// Performance component info
	if m.profiler != nil {
		info["profiler"] = m.profiler.GetDebugInfo(m.ctx)
	}

	if m.viewport != nil {
		columns := make([]*Column, len(m.columns))
		for i := range m.columns {
			columns[i] = &m.columns[i]
		}

		if stats, err := m.viewport.GetViewportStats(m.ctx, columns); err == nil {
			info["viewport"] = stats
		}
	}

	if m.renderer != nil {
		info["renderer"] = m.renderer.GetDebugInfo(m.ctx)
	}

	if m.cardCache != nil {
		info["card_cache"] = m.cardCache.GetStats()
	}

	// Memory usage
	if memUsage, err := m.GetMemoryUsage(); err == nil {
		info["memory_usage_mb"] = float64(memUsage) / 1024 / 1024
	}

	return info
}

// Cleanup performs cleanup operations when the model is being destroyed
func (m *Model) Cleanup() error {
	// Stop event streaming
	m.stopEventStream()

	// Cleanup performance components
	if m.profiler != nil {
		m.profiler.Disable()
	}

	if m.renderer != nil {
		if err := m.renderer.Cleanup(m.ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to cleanup renderer").
				WithComponent("kanban.model").
				WithOperation("Cleanup")
		}
	}

	if m.cardCache != nil {
		m.cardCache.Clear()
	}

	return nil
}
