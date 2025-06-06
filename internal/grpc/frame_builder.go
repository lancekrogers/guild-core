package grpc

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/internal/kanban"
	"github.com/guild-ventures/guild-core/internal/commission"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m"
	
	bgRed    = "\033[41m"
	bgGreen  = "\033[42m"
	bgYellow = "\033[43m"
	bgBlue   = "\033[44m"
)

// FrameMetadata contains metadata about the frame
type FrameMetadata struct {
	Width          int
	Height         int
	ActiveAgents   int
	TotalTasks     int
	CompletedTasks int
	FPS            float64
}

// FrameBuilder creates ASCII board representations
type FrameBuilder struct {
	width, height int
	campaignMgr   campaign.Manager
	commissionMgr *commission.Manager
	kanbanMgr     *kanban.Manager
	agentReg      registry.AgentRegistry
	
	// Rendering state
	lastRender    time.Time
	frameCount    int
	fps           float64
}

// NewFrameBuilder creates a new frame builder
func NewFrameBuilder(
	campaignMgr campaign.Manager,
	commissionMgr *commission.Manager,
	kanbanMgr *kanban.Manager,
	agentReg registry.AgentRegistry,
) *FrameBuilder {
	return &FrameBuilder{
		width:         80,
		height:        24,
		campaignMgr:   campaignMgr,
		commissionMgr: commissionMgr,
		kanbanMgr:     kanbanMgr,
		agentReg:      agentReg,
		lastRender:   time.Now(),
	}
}

// SetSize sets the frame size
func (fb *FrameBuilder) SetSize(width, height int) {
	fb.width = width
	fb.height = height
}

// BuildFrame creates an ASCII board representation
func (fb *FrameBuilder) BuildFrame(ctx context.Context, campaign *campaign.Campaign, options watchOptions) ([]byte, FrameMetadata) {
	// Update FPS calculation
	now := time.Now()
	elapsed := now.Sub(fb.lastRender).Seconds()
	if elapsed > 0 {
		fb.fps = 1.0 / elapsed
	}
	fb.lastRender = now
	fb.frameCount++

	// Create buffer for frame
	var buf bytes.Buffer

	// Build sections based on options
	header := fb.buildHeader(campaign)
	buf.Write(header)

	// Calculate remaining height
	headerLines := bytes.Count(header, []byte{'\n'})
	remainingHeight := fb.height - headerLines - 1 // -1 for bottom border

	// Build content sections
	if options.includeAgents && options.includeKanban {
		// Split screen between agents and kanban
		agentHeight := remainingHeight / 3
		kanbanHeight := remainingHeight - agentHeight
		
		agentGrid := fb.buildAgentGrid(agentHeight)
		buf.Write(agentGrid)
		
		kanbanBoard := fb.buildKanbanBoard(ctx, campaign, kanbanHeight)
		buf.Write(kanbanBoard)
	} else if options.includeAgents {
		agentGrid := fb.buildAgentGrid(remainingHeight)
		buf.Write(agentGrid)
	} else if options.includeKanban {
		kanbanBoard := fb.buildKanbanBoard(ctx, campaign, remainingHeight)
		buf.Write(kanbanBoard)
	} else {
		// Just show progress
		progress := fb.buildProgressDetails(campaign, remainingHeight)
		buf.Write(progress)
	}

	// Add bottom border
	buf.Write(fb.buildBottomBorder())

	// Collect metadata
	metadata := FrameMetadata{
		Width:          fb.width,
		Height:         fb.height,
		ActiveAgents:   fb.getActiveAgentCount(),
		TotalTasks:     fb.getTotalTaskCount(),
		CompletedTasks: fb.getCompletedTaskCount(),
		FPS:            fb.fps,
	}

	return buf.Bytes(), metadata
}

// buildHeader creates the header section
func (fb *FrameBuilder) buildHeader(campaign *campaign.Campaign) []byte {
	var buf bytes.Buffer

	// Top border
	buf.WriteString(colorBlue + "╔" + strings.Repeat("═", fb.width-2) + "╗" + colorReset + "\n")

	// Campaign title
	title := fmt.Sprintf("Campaign: %s", campaign.Name)
	titleLine := fb.centerText(title, fb.width-2)
	buf.WriteString(colorBlue + "║" + colorBold + titleLine + colorReset + colorBlue + "║" + colorReset + "\n")

	// Status line
	statusColor := fb.getStatusColor(campaign.Status)
	statusLine := fmt.Sprintf("Status: %s%s%s | Progress: %.1f%%", 
		statusColor, campaign.Status, colorReset, campaign.Progress*100)
	buf.WriteString(colorBlue + "║" + colorReset + " " + statusLine)
	buf.WriteString(strings.Repeat(" ", fb.width-len(stripANSI(statusLine))-4))
	buf.WriteString(colorBlue + "║" + colorReset + "\n")

	// Separator
	buf.WriteString(colorBlue + "╠" + strings.Repeat("═", fb.width-2) + "╣" + colorReset + "\n")

	return buf.Bytes()
}

// buildAgentGrid creates the agent status grid
func (fb *FrameBuilder) buildAgentGrid(height int) []byte {
	var buf bytes.Buffer

	// Title
	buf.WriteString(colorBlue + "║" + colorBold + " Artisans (Agents)" + colorReset)
	buf.WriteString(strings.Repeat(" ", fb.width-20))
	buf.WriteString(colorBlue + "║" + colorReset + "\n")

	// Get agents
	agents := []string{}
	if fb.agentReg != nil {
		agents = fb.agentReg.ListAgentTypes()
	}

	// Display agents in grid
	contentHeight := height - 2 // -1 for title, -1 for bottom separator
	for i := 0; i < contentHeight; i++ {
		if i < len(agents) {
			agentLine := fmt.Sprintf("  • %s: %sActive%s", agents[i], colorGreen, colorReset)
			buf.WriteString(colorBlue + "║" + colorReset + agentLine)
			buf.WriteString(strings.Repeat(" ", fb.width-len(stripANSI(agentLine))-2))
			buf.WriteString(colorBlue + "║" + colorReset + "\n")
		} else {
			// Empty line
			buf.WriteString(colorBlue + "║" + strings.Repeat(" ", fb.width-2) + "║" + colorReset + "\n")
		}
	}

	// Separator
	buf.WriteString(colorBlue + "╠" + strings.Repeat("═", fb.width-2) + "╣" + colorReset + "\n")

	return buf.Bytes()
}

// buildKanbanBoard creates the kanban board section
func (fb *FrameBuilder) buildKanbanBoard(ctx context.Context, campaign *campaign.Campaign, height int) []byte {
	var buf bytes.Buffer

	// Title
	buf.WriteString(colorBlue + "║" + colorBold + " Workshop Board (Kanban)" + colorReset)
	buf.WriteString(strings.Repeat(" ", fb.width-27))
	buf.WriteString(colorBlue + "║" + colorReset + "\n")

	// Column headers
	colWidth := (fb.width - 2) / 3
	headers := []string{"Todo", "In Progress", "Done"}
	colors := []string{colorYellow, colorCyan, colorGreen}
	
	buf.WriteString(colorBlue + "║" + colorReset)
	for i, header := range headers {
		headerText := colors[i] + fb.centerText(header, colWidth) + colorReset
		buf.WriteString(headerText)
	}
	buf.WriteString(colorBlue + "║" + colorReset + "\n")

	// Separator line
	buf.WriteString(colorBlue + "║" + colorGray + strings.Repeat("─", fb.width-2) + colorReset + colorBlue + "║" + colorReset + "\n")

	// Get tasks
	var board *kanban.Board
	if fb.kanbanMgr != nil {
		boards, _ := fb.kanbanMgr.ListBoards(ctx)
		if len(boards) > 0 {
			board = boards[0] // Use first board for now
		}
	}

	// Display tasks
	contentHeight := height - 4 // -1 title, -1 headers, -1 separator, -1 for bottom
	todoTasks := []string{}
	inProgressTasks := []string{}
	doneTasks := []string{}

	if board != nil {
		// TODO: Implement proper task retrieval from board store
		// For now, just show placeholder data
		todoTasks = append(todoTasks, "[Tasks would be loaded from store]")
		inProgressTasks = append(inProgressTasks, "[Tasks would be loaded from store]")
		doneTasks = append(doneTasks, "[Tasks would be loaded from store]")
	}

	// Render task rows
	for i := 0; i < contentHeight; i++ {
		buf.WriteString(colorBlue + "║" + colorReset)
		
		// Todo column
		if i < len(todoTasks) {
			task := truncateText(todoTasks[i], colWidth-2)
			buf.WriteString(" " + task + strings.Repeat(" ", colWidth-len(task)-1))
		} else {
			buf.WriteString(strings.Repeat(" ", colWidth))
		}
		
		// In Progress column
		if i < len(inProgressTasks) {
			task := truncateText(inProgressTasks[i], colWidth-2)
			buf.WriteString(" " + task + strings.Repeat(" ", colWidth-len(task)-1))
		} else {
			buf.WriteString(strings.Repeat(" ", colWidth))
		}
		
		// Done column
		if i < len(doneTasks) {
			task := truncateText(doneTasks[i], colWidth-2)
			buf.WriteString(" " + task + strings.Repeat(" ", colWidth-len(task)-1))
		} else {
			buf.WriteString(strings.Repeat(" ", colWidth))
		}
		
		buf.WriteString(colorBlue + "║" + colorReset + "\n")
	}

	return buf.Bytes()
}

// buildProgressDetails creates a detailed progress view
func (fb *FrameBuilder) buildProgressDetails(campaign *campaign.Campaign, height int) []byte {
	var buf bytes.Buffer

	// Title
	buf.WriteString(colorBlue + "║" + colorBold + " Campaign Progress" + colorReset)
	buf.WriteString(strings.Repeat(" ", fb.width-21))
	buf.WriteString(colorBlue + "║" + colorReset + "\n")

	// Progress bar
	progressWidth := fb.width - 4
	filled := int(campaign.Progress * float64(progressWidth))
	progressBar := colorGreen + strings.Repeat("█", filled) + colorGray + strings.Repeat("░", progressWidth-filled) + colorReset
	
	buf.WriteString(colorBlue + "║ " + progressBar + " ║" + colorReset + "\n")

	// Stats
	stats := []string{
		fmt.Sprintf("Total Objectives: %d", campaign.TotalObjectives),
		fmt.Sprintf("Completed: %d", campaign.CompletedObjectives),
		fmt.Sprintf("Progress: %.1f%%", campaign.Progress*100),
		fmt.Sprintf("Status: %s", campaign.Status),
	}

	for _, stat := range stats {
		buf.WriteString(colorBlue + "║" + colorReset + "  " + stat)
		buf.WriteString(strings.Repeat(" ", fb.width-len(stat)-4))
		buf.WriteString(colorBlue + "║" + colorReset + "\n")
	}

	// Fill remaining height
	usedHeight := 2 + len(stats) // title + progress bar + stats
	for i := usedHeight; i < height; i++ {
		buf.WriteString(colorBlue + "║" + strings.Repeat(" ", fb.width-2) + "║" + colorReset + "\n")
	}

	return buf.Bytes()
}

// buildBottomBorder creates the bottom border
func (fb *FrameBuilder) buildBottomBorder() []byte {
	var buf bytes.Buffer
	
	// FPS counter in bottom right
	fpsText := fmt.Sprintf(" FPS: %.1f ", fb.fps)
	borderWidth := fb.width - len(fpsText) - 2
	
	buf.WriteString(colorBlue + "╚" + strings.Repeat("═", borderWidth) + colorReset)
	buf.WriteString(colorGray + fpsText + colorReset)
	buf.WriteString(colorBlue + "╝" + colorReset + "\n")
	
	return buf.Bytes()
}

// Helper methods

// centerText centers text within a given width
func (fb *FrameBuilder) centerText(text string, width int) string {
	textLen := len(stripANSI(text))
	if textLen >= width {
		return text
	}
	
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-textLen-padding)
}

// getStatusColor returns the color for a campaign status
func (fb *FrameBuilder) getStatusColor(status campaign.CampaignStatus) string {
	switch status {
	case campaign.CampaignStatusDream:
		return colorPurple
	case campaign.CampaignStatusPlanning:
		return colorCyan
	case campaign.CampaignStatusReady:
		return colorYellow
	case campaign.CampaignStatusActive:
		return colorGreen
	case campaign.CampaignStatusPaused:
		return colorYellow
	case campaign.CampaignStatusCompleted:
		return colorBlue
	case campaign.CampaignStatusCancelled:
		return colorRed
	default:
		return colorGray
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	// Simple implementation - in production use a proper ANSI stripping library
	result := str
	for _, code := range []string{
		colorReset, colorBold, colorRed, colorGreen, colorYellow,
		colorBlue, colorPurple, colorCyan, colorWhite, colorGray,
		bgRed, bgGreen, bgYellow, bgBlue,
	} {
		result = strings.ReplaceAll(result, code, "")
	}
	return result
}

// truncateText truncates text to fit within a given width
func truncateText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}
	if maxWidth <= 3 {
		return text[:maxWidth]
	}
	return text[:maxWidth-3] + "..."
}

// Stub methods for metadata - these would be implemented based on actual data

func (fb *FrameBuilder) getActiveAgentCount() int {
	if fb.agentReg != nil {
		return len(fb.agentReg.ListAgentTypes())
	}
	return 0
}

func (fb *FrameBuilder) getTotalTaskCount() int {
	// This would query the kanban manager for actual task counts
	return 0
}

func (fb *FrameBuilder) getCompletedTaskCount() int {
	// This would query the kanban manager for completed task counts
	return 0
}