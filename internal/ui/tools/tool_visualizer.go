// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/internal/ui/progress"
)

// ToolVisualizer provides enhanced visualization for tool execution
type ToolVisualizer struct {
	style          lipgloss.Style
	errorStyle     lipgloss.Style
	successStyle   lipgloss.Style
	infoStyle      lipgloss.Style
	warningStyle   lipgloss.Style
	headerStyle    lipgloss.Style
	separatorStyle lipgloss.Style
	codeStyle      lipgloss.Style
	
	// Progress tracking
	activeOperations map[string]*ToolOperation
	mu               sync.RWMutex
	
	// Statistics
	stats toolStatsInternal
}

// ToolOperation represents an active tool operation
type ToolOperation struct {
	ID          string
	ToolName    string
	StartTime   time.Time
	EndTime     time.Time
	Status      OperationStatus
	Parameters  map[string]interface{}
	Result      interface{}
	Error       error
	Progress    *progress.Indicator
	Steps       []OperationStep
	Metadata    map[string]interface{}
}

// OperationStep represents a step in tool execution
type OperationStep struct {
	Name        string
	StartTime   time.Time
	EndTime     time.Time
	Status      OperationStatus
	Description string
	Result      interface{}
	Error       error
}

// OperationStatus represents the status of an operation
type OperationStatus int

const (
	OpStatusPending OperationStatus = iota
	OpStatusInProgress
	OpStatusCompleted
	OpStatusFailed
	OpStatusCancelled
)

// ToolStats tracks tool execution statistics (internal with mutex)
type toolStatsInternal struct {
	TotalExecutions     int
	SuccessfulOps       int
	FailedOps           int
	TotalDuration       time.Duration
	AverageDuration     time.Duration
	ToolUsageCounts     map[string]int
	ErrorCounts         map[string]int
	FileOperationCounts map[string]int
	mu                  sync.RWMutex
}

// ToolStats represents read-only tool execution statistics
type ToolStats struct {
	TotalExecutions     int               `json:"total_executions"`
	SuccessfulOps       int               `json:"successful_ops"`
	FailedOps           int               `json:"failed_ops"`
	TotalDuration       time.Duration     `json:"total_duration"`
	AverageDuration     time.Duration     `json:"average_duration"`
	ToolUsageCounts     map[string]int    `json:"tool_usage_counts"`
	ErrorCounts         map[string]int    `json:"error_counts"`
	FileOperationCounts map[string]int    `json:"file_operation_counts"`
}

// NewToolVisualizer creates a new tool visualizer
func NewToolVisualizer() *ToolVisualizer {
	return &ToolVisualizer{
		style: lipgloss.NewStyle().Margin(0, 1),
		
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
			
		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true),
			
		infoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")),
			
		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),
			
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Underline(true),
			
		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
			
		codeStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1).
			Margin(0, 1),
			
		activeOperations: make(map[string]*ToolOperation),
		stats: toolStatsInternal{
			ToolUsageCounts:     make(map[string]int),
			ErrorCounts:         make(map[string]int),
			FileOperationCounts: make(map[string]int),
		},
	}
}

// StartToolExecution starts visualizing a tool execution
func (v *ToolVisualizer) StartToolExecution(toolName string, params map[string]interface{}) string {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	opID := fmt.Sprintf("%s-%d", toolName, time.Now().UnixNano())
	
	operation := &ToolOperation{
		ID:         opID,
		ToolName:   toolName,
		StartTime:  time.Now(),
		Status:     OpStatusInProgress,
		Parameters: params,
		Progress:   progress.NewIndicator(),
		Steps:      []OperationStep{},
		Metadata:   make(map[string]interface{}),
	}
	
	v.activeOperations[opID] = operation
	v.stats.TotalExecutions++
	v.stats.ToolUsageCounts[toolName]++
	
	// Display initial visualization
	v.displayToolStart(operation)
	
	return opID
}

// UpdateToolProgress updates the progress of a tool execution
func (v *ToolVisualizer) UpdateToolProgress(opID string, step string, progress float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	operation, exists := v.activeOperations[opID]
	if !exists {
		return
	}
	
	// Add or update step
	stepIndex := v.findOrCreateStep(operation, step)
	operation.Steps[stepIndex].Status = OpStatusInProgress
	
	v.displayProgressUpdate(operation, step, progress)
}

// CompleteToolExecution completes a tool execution
func (v *ToolVisualizer) CompleteToolExecution(opID string, success bool, result interface{}, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	operation, exists := v.activeOperations[opID]
	if !exists {
		return
	}
	
	operation.EndTime = time.Now()
	operation.Result = result
	operation.Error = err
	
	if success {
		operation.Status = OpStatusCompleted
		v.stats.SuccessfulOps++
	} else {
		operation.Status = OpStatusFailed
		v.stats.FailedOps++
		if err != nil {
			v.stats.ErrorCounts[err.Error()]++
		}
	}
	
	// Update statistics
	duration := operation.EndTime.Sub(operation.StartTime)
	v.stats.TotalDuration += duration
	v.stats.AverageDuration = v.stats.TotalDuration / time.Duration(v.stats.TotalExecutions)
	
	// Display completion
	v.displayToolCompletion(operation)
	
	// Clean up
	delete(v.activeOperations, opID)
}

// ShowFileOperation displays a file operation with enhanced visualization
func (v *ToolVisualizer) ShowFileOperation(op string, file string, details map[string]interface{}) {
	v.stats.mu.Lock()
	v.stats.FileOperationCounts[op]++
	v.stats.mu.Unlock()
	
	icon := v.getFileOpIcon(op)
	color := v.getFileOpColor(op)
	
	// Header
	header := fmt.Sprintf("%s %s: %s", 
		icon, 
		strings.Title(op), 
		v.style.Copy().Foreground(color).Underline(true).Render(file))
	
	fmt.Println(v.headerStyle.Render(header))
	
	// Show file details
	v.displayFileDetails(op, file, details)
	
	// Show changes if available
	if changes, ok := details["changes"].([]string); ok && len(changes) > 0 {
		v.displayFileChanges(changes)
	}
	
	// Show before/after if available
	if before, ok := details["before"].(string); ok {
		if after, ok := details["after"].(string); ok {
			v.displayBeforeAfter(before, after)
		}
	}
}

// ShowCommandExecution displays command execution with live output
func (v *ToolVisualizer) ShowCommandExecution(cmd string) *CommandVisualizer {
	fmt.Println(v.headerStyle.Render(fmt.Sprintf("💻 Executing: %s", cmd)))
	
	return &CommandVisualizer{
		command:   cmd,
		startTime: time.Now(),
		style:     v.style,
		lines:     []CommandLine{},
	}
}

// CommandVisualizer handles command execution visualization
type CommandVisualizer struct {
	command   string
	startTime time.Time
	style     lipgloss.Style
	lines     []CommandLine
	mu        sync.Mutex
}

// CommandLine represents a line of command output
type CommandLine struct {
	Content   string
	Timestamp time.Time
	Type      LineType
}

// LineType represents the type of command output line
type LineType int

const (
	LineTypeStdout LineType = iota
	LineTypeStderr
	LineTypeInfo
	LineTypeError
)

// StreamOutput streams command output
func (c *CommandVisualizer) StreamOutput(line string, lineType LineType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	cmdLine := CommandLine{
		Content:   line,
		Timestamp: time.Now(),
		Type:      lineType,
	}
	
	c.lines = append(c.lines, cmdLine)
	
	// Display with appropriate styling
	prefix := "   │ "
	switch lineType {
	case LineTypeStderr, LineTypeError:
		fmt.Printf("%s%s\n", prefix, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(line))
	case LineTypeInfo:
		fmt.Printf("%s%s\n", prefix, lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(line))
	default:
		fmt.Printf("%s%s\n", prefix, line)
	}
}

// Complete completes the command execution
func (c *CommandVisualizer) Complete(exitCode int, output *CommandOutput) {
	duration := time.Since(c.startTime)
	
	// Summary line
	if exitCode == 0 {
		summary := fmt.Sprintf("✅ Completed in %v", duration.Round(time.Millisecond))
		fmt.Printf("   └─ %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(summary))
	} else {
		summary := fmt.Sprintf("❌ Failed with exit code %d (took %v)", exitCode, duration.Round(time.Millisecond))
		fmt.Printf("   └─ %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(summary))
	}
	
	// Show output summary if available
	if output != nil {
		c.displayOutputSummary(output)
	}
}

// CommandOutput represents command execution output
type CommandOutput struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	Duration  time.Duration
	FilesRead []string
	FilesWritten []string
	Errors    []string
}

// displayOutputSummary displays a summary of command output
func (c *CommandVisualizer) displayOutputSummary(output *CommandOutput) {
	if len(output.FilesWritten) > 0 {
		fmt.Printf("   📝 Files modified: %s\n", strings.Join(output.FilesWritten, ", "))
	}
	
	if len(output.FilesRead) > 0 {
		fmt.Printf("   👁️  Files read: %s\n", strings.Join(output.FilesRead, ", "))
	}
	
	if len(output.Errors) > 0 {
		fmt.Printf("   ⚠️  Warnings: %d\n", len(output.Errors))
	}
}

// Display methods

// displayToolStart displays the start of tool execution
func (v *ToolVisualizer) displayToolStart(op *ToolOperation) {
	header := fmt.Sprintf("🔧 %s", op.ToolName)
	fmt.Println(v.headerStyle.Render(header))
	
	if len(op.Parameters) > 0 {
		fmt.Println(v.infoStyle.Render("📋 Parameters:"))
		v.displayParameters(op.Parameters, 1)
	}
	
	fmt.Println(v.separatorStyle.Render(strings.Repeat("─", 50)))
}

// displayProgressUpdate displays progress updates
func (v *ToolVisualizer) displayProgressUpdate(op *ToolOperation, step string, progress float64) {
	progressBar := v.generateProgressBar(progress, 30)
	status := fmt.Sprintf("🔄 %s: %s %.0f%%", step, progressBar, progress*100)
	fmt.Printf("\r%s", v.infoStyle.Render(status))
}

// displayToolCompletion displays tool completion
func (v *ToolVisualizer) displayToolCompletion(op *ToolOperation) {
	fmt.Print("\r") // Clear progress line
	
	duration := op.EndTime.Sub(op.StartTime)
	
	if op.Status == OpStatusCompleted {
		header := fmt.Sprintf("✅ %s completed in %v", op.ToolName, duration.Round(time.Millisecond))
		fmt.Println(v.successStyle.Render(header))
		
		if op.Result != nil {
			v.displayResult(op.Result)
		}
	} else {
		header := fmt.Sprintf("❌ %s failed after %v", op.ToolName, duration.Round(time.Millisecond))
		fmt.Println(v.errorStyle.Render(header))
		
		if op.Error != nil {
			fmt.Println(v.errorStyle.Render(fmt.Sprintf("Error: %v", op.Error)))
		}
	}
	
	fmt.Println(v.separatorStyle.Render(strings.Repeat("─", 50)))
}

// displayFileDetails displays file operation details
func (v *ToolVisualizer) displayFileDetails(op, file string, details map[string]interface{}) {
	switch op {
	case "create":
		if lines, ok := details["lines"].(int); ok {
			fmt.Printf("   📝 Created with %d lines\n", lines)
		}
		if size, ok := details["size"].(int64); ok {
			fmt.Printf("   📏 Size: %s\n", v.formatBytes(size))
		}
		
	case "edit", "modify":
		if edits, ok := details["edits"].(int); ok {
			fmt.Printf("   ✏️  Applied %d edits\n", edits)
		}
		if added, ok := details["lines_added"].(int); ok {
			if removed, ok := details["lines_removed"].(int); ok {
				fmt.Printf("   📊 Lines: +%d -%d\n", added, removed)
			}
		}
		
	case "delete", "remove":
		fmt.Printf("   🗑️  File removed\n")
		if backup, ok := details["backup"].(string); ok {
			fmt.Printf("   💾 Backup: %s\n", backup)
		}
		
	case "move", "rename":
		if dest, ok := details["destination"].(string); ok {
			fmt.Printf("   📦 Moved to: %s\n", dest)
		}
		
	case "copy":
		if dest, ok := details["destination"].(string); ok {
			fmt.Printf("   📋 Copied to: %s\n", dest)
		}
	}
}

// displayFileChanges displays file changes
func (v *ToolVisualizer) displayFileChanges(changes []string) {
	if len(changes) == 0 {
		return
	}
	
	fmt.Println(v.infoStyle.Render("   📝 Changes:"))
	
	for _, change := range changes {
		if strings.HasPrefix(change, "+") {
			fmt.Printf("     %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(change))
		} else if strings.HasPrefix(change, "-") {
			fmt.Printf("     %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(change))
		} else {
			fmt.Printf("     %s\n", change)
		}
	}
}

// displayBeforeAfter displays before/after content comparison
func (v *ToolVisualizer) displayBeforeAfter(before, after string) {
	fmt.Println(v.infoStyle.Render("   🔍 Changes:"))
	
	// Simple diff visualization
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	
	maxLines := max(len(beforeLines), len(afterLines))
	if maxLines > 10 {
		maxLines = 10 // Limit display
	}
	
	for i := 0; i < maxLines; i++ {
		var beforeLine, afterLine string
		
		if i < len(beforeLines) {
			beforeLine = beforeLines[i]
		}
		if i < len(afterLines) {
			afterLine = afterLines[i]
		}
		
		if beforeLine != afterLine {
			if beforeLine != "" {
				fmt.Printf("     %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("- "+beforeLine))
			}
			if afterLine != "" {
				fmt.Printf("     %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("+ "+afterLine))
			}
		}
	}
}

// displayParameters displays tool parameters in a tree format
func (v *ToolVisualizer) displayParameters(params map[string]interface{}, indent int) {
	// Sort keys for consistent display
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	prefix := strings.Repeat("  ", indent)
	
	for _, key := range keys {
		value := params[key]
		
		switch val := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s• %s:\n", prefix, key)
			v.displayParameters(val, indent+1)
		case []interface{}:
			fmt.Printf("%s• %s: [%d items]\n", prefix, key, len(val))
		case string:
			if len(val) > 100 {
				fmt.Printf("%s• %s: %s...\n", prefix, key, val[:100])
			} else {
				fmt.Printf("%s• %s: %s\n", prefix, key, val)
			}
		default:
			fmt.Printf("%s• %s: %v\n", prefix, key, value)
		}
	}
}

// displayResult displays tool execution result
func (v *ToolVisualizer) displayResult(result interface{}) {
	fmt.Println(v.infoStyle.Render("📄 Result:"))
	
	switch r := result.(type) {
	case string:
		if len(r) > 500 {
			fmt.Printf("   %s...\n", r[:500])
		} else {
			fmt.Printf("   %s\n", r)
		}
	case map[string]interface{}:
		// Pretty print JSON
		if jsonBytes, err := json.MarshalIndent(r, "   ", "  "); err == nil {
			fmt.Printf("   %s\n", string(jsonBytes))
		} else {
			fmt.Printf("   %v\n", r)
		}
	default:
		fmt.Printf("   %v\n", result)
	}
}

// Helper methods

// getFileOpIcon returns the appropriate icon for file operations
func (v *ToolVisualizer) getFileOpIcon(op string) string {
	icons := map[string]string{
		"create": "📄",
		"edit":   "✏️",
		"modify": "✏️",
		"delete": "🗑️",
		"remove": "🗑️",
		"read":   "👁️",
		"move":   "📦",
		"rename": "📦",
		"copy":   "📋",
	}
	
	if icon, ok := icons[op]; ok {
		return icon
	}
	return "📄"
}

// getFileOpColor returns the appropriate color for file operations
func (v *ToolVisualizer) getFileOpColor(op string) lipgloss.Color {
	colors := map[string]lipgloss.Color{
		"create": lipgloss.Color("42"),  // Green
		"edit":   lipgloss.Color("33"),  // Yellow
		"modify": lipgloss.Color("33"),  // Yellow
		"delete": lipgloss.Color("196"), // Red
		"remove": lipgloss.Color("196"), // Red
		"read":   lipgloss.Color("33"),  // Blue
		"move":   lipgloss.Color("99"),  // Purple
		"rename": lipgloss.Color("99"),  // Purple
		"copy":   lipgloss.Color("45"),  // Cyan
	}
	
	if color, ok := colors[op]; ok {
		return color
	}
	return lipgloss.Color("252")
}

// generateProgressBar generates a visual progress bar
func (v *ToolVisualizer) generateProgressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	
	filled := int(progress * float64(width))
	empty := width - filled
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("[%s]", bar)
}

// formatBytes formats byte size in human readable format
func (v *ToolVisualizer) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	suffixes := []string{"B", "KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), suffixes[exp])
}

// findOrCreateStep finds or creates a step in the operation
func (v *ToolVisualizer) findOrCreateStep(op *ToolOperation, stepName string) int {
	for i, step := range op.Steps {
		if step.Name == stepName {
			return i
		}
	}
	
	// Create new step
	step := OperationStep{
		Name:      stepName,
		StartTime: time.Now(),
		Status:    OpStatusInProgress,
	}
	
	op.Steps = append(op.Steps, step)
	return len(op.Steps) - 1
}

// GetStats returns current tool execution statistics
func (v *ToolVisualizer) GetStats() ToolStats {
	v.stats.mu.RLock()
	defer v.stats.mu.RUnlock()
	
	// Create a copy to avoid race conditions (read-only struct, no mutex)
	statsCopy := ToolStats{
		TotalExecutions:     v.stats.TotalExecutions,
		SuccessfulOps:       v.stats.SuccessfulOps,
		FailedOps:           v.stats.FailedOps,
		TotalDuration:       v.stats.TotalDuration,
		AverageDuration:     v.stats.AverageDuration,
		ToolUsageCounts:     make(map[string]int),
		ErrorCounts:         make(map[string]int),
		FileOperationCounts: make(map[string]int),
	}
	
	for k, v := range v.stats.ToolUsageCounts {
		statsCopy.ToolUsageCounts[k] = v
	}
	for k, v := range v.stats.ErrorCounts {
		statsCopy.ErrorCounts[k] = v
	}
	for k, v := range v.stats.FileOperationCounts {
		statsCopy.FileOperationCounts[k] = v
	}
	
	return statsCopy
}

// DisplaySummary displays a summary of tool execution statistics
func (v *ToolVisualizer) DisplaySummary() {
	stats := v.GetStats()
	
	fmt.Println(v.headerStyle.Render("📊 Tool Execution Summary"))
	
	successRate := float64(stats.SuccessfulOps) / float64(stats.TotalExecutions) * 100
	
	fmt.Printf("Total Executions: %d\n", stats.TotalExecutions)
	fmt.Printf("Success Rate: %.1f%% (%d/%d)\n", successRate, stats.SuccessfulOps, stats.TotalExecutions)
	fmt.Printf("Average Duration: %v\n", stats.AverageDuration.Round(time.Millisecond))
	
	if len(stats.ToolUsageCounts) > 0 {
		fmt.Println("\nMost Used Tools:")
		v.displayTopCounts(stats.ToolUsageCounts, 5)
	}
	
	if len(stats.FileOperationCounts) > 0 {
		fmt.Println("\nFile Operations:")
		v.displayTopCounts(stats.FileOperationCounts, 5)
	}
}

// displayTopCounts displays the top N items from a count map
func (v *ToolVisualizer) displayTopCounts(counts map[string]int, n int) {
	type countItem struct {
		name  string
		count int
	}
	
	var items []countItem
	for name, count := range counts {
		items = append(items, countItem{name, count})
	}
	
	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})
	
	for i, item := range items {
		if i >= n {
			break
		}
		fmt.Printf("  %d. %s: %d\n", i+1, item.name, item.count)
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}