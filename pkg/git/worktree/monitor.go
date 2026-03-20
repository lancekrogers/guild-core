// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// ConflictMonitor monitors worktrees for potential conflicts
type ConflictMonitor struct {
	manager  Manager
	detector *ConflictDetector
	notifier *ConflictNotifier
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
}

// NewConflictMonitor creates a new conflict monitor
func NewConflictMonitor(manager Manager) *ConflictMonitor {
	return &ConflictMonitor{
		manager:  manager,
		detector: NewConflictDetector(),
		notifier: NewConflictNotifier(),
		stopCh:   make(chan struct{}),
	}
}

// Start begins monitoring worktrees for conflicts
func (cm *ConflictMonitor) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree.monitor").
			WithOperation("Start")
	}

	cm.mu.Lock()
	if cm.running {
		cm.mu.Unlock()
		return gerror.New(gerror.ErrCodeConflict, "monitor already running", nil).
			WithComponent("git.worktree.monitor").
			WithOperation("Start")
	}
	cm.running = true
	cm.mu.Unlock()

	go cm.monitorWorktrees(ctx)
	return nil
}

// Stop stops the conflict monitor
func (cm *ConflictMonitor) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running {
		close(cm.stopCh)
		cm.running = false
	}
}

// monitorWorktrees continuously monitors for conflicts
func (cm *ConflictMonitor) monitorWorktrees(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.checkForConflicts(ctx)
		}
	}
}

// checkForConflicts checks all active worktrees for potential conflicts
func (cm *ConflictMonitor) checkForConflicts(ctx context.Context) {
	worktrees := cm.manager.GetActiveWorktrees()

	// Check each pair for potential conflicts
	for i, wt1 := range worktrees {
		for j := i + 1; j < len(worktrees); j++ {
			wt2 := worktrees[j]

			// Skip if same agent (less likely to conflict)
			if wt1.AgentID == wt2.AgentID {
				continue
			}

			if conflicts := cm.detector.DetectConflicts(ctx, wt1, wt2); len(conflicts) > 0 {
				cm.handleConflicts(ctx, wt1, wt2, conflicts)
			}
		}
	}
}

// handleConflicts processes detected conflicts
func (cm *ConflictMonitor) handleConflicts(ctx context.Context, wt1, wt2 *Worktree, conflicts []Conflict) {
	for _, conflict := range conflicts {
		cm.notifier.NotifyConflict(ConflictEvent{
			Type:      ConflictEventDetected,
			Conflict:  conflict,
			Worktree1: wt1.ID,
			Worktree2: wt2.ID,
			Timestamp: time.Now(),
		})
	}
}

// ConflictDetector detects conflicts between worktrees
type ConflictDetector struct{}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectConflicts finds conflicts between two worktrees
func (cd *ConflictDetector) DetectConflicts(ctx context.Context, wt1, wt2 *Worktree) []Conflict {
	var conflicts []Conflict

	// Get changed files in both worktrees
	files1 := cd.getChangedFiles(ctx, wt1)
	files2 := cd.getChangedFiles(ctx, wt2)

	// Find overlapping files
	for file := range files1 {
		if _, exists := files2[file]; exists {
			// Both modified same file
			conflict := cd.analyzeFileConflict(ctx, wt1, wt2, file)
			if conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	// Check for semantic conflicts (different files, same functionality)
	semanticConflicts := cd.detectSemanticConflicts(ctx, wt1, wt2, files1, files2)
	conflicts = append(conflicts, semanticConflicts...)

	return conflicts
}

// getChangedFiles returns a map of changed files in the worktree
func (cd *ConflictDetector) getChangedFiles(ctx context.Context, wt *Worktree) map[string]struct{} {
	files := make(map[string]struct{})

	// Use git to get changed files relative to base branch
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", fmt.Sprintf("origin/%s...HEAD", wt.BaseBranch))
	cmd.Dir = wt.Path

	output, err := cmd.Output()
	if err != nil {
		return files
	}

	for _, file := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if file != "" {
			files[file] = struct{}{}
		}
	}

	return files
}

// analyzeFileConflict analyzes a specific file conflict between worktrees
func (cd *ConflictDetector) analyzeFileConflict(ctx context.Context, wt1, wt2 *Worktree, file string) *Conflict {
	// Get file content from both worktrees
	content1, err1 := cd.readFile(wt1, file)
	content2, err2 := cd.readFile(wt2, file)

	if err1 != nil || err2 != nil {
		return nil
	}

	// Get base content
	baseContent, _ := cd.getBaseContent(ctx, wt1, file)

	// Perform three-way diff analysis
	diff := cd.threeWayDiff(baseContent, content1, content2)

	if diff.HasConflicts() {
		return &Conflict{
			ID:         cd.generateConflictID(),
			File:       file,
			Worktree1:  wt1.ID,
			Worktree2:  wt2.ID,
			Type:       ConflictTypeContent,
			Diff:       diff,
			Severity:   cd.assessConflictSeverity(diff),
			DetectedAt: time.Now(),
		}
	}

	return nil
}

// detectSemanticConflicts detects conflicts in functionality even when different files are changed
func (cd *ConflictDetector) detectSemanticConflicts(ctx context.Context, wt1, wt2 *Worktree, files1, files2 map[string]struct{}) []Conflict {
	var conflicts []Conflict

	// Check for API changes that might conflict
	api1 := cd.extractAPIChanges(ctx, wt1, files1)
	api2 := cd.extractAPIChanges(ctx, wt2, files2)

	for _, apiChange1 := range api1 {
		for _, apiChange2 := range api2 {
			if cd.apiChangesConflict(apiChange1, apiChange2) {
				conflicts = append(conflicts, Conflict{
					ID:         cd.generateConflictID(),
					File:       fmt.Sprintf("%s / %s", apiChange1.File, apiChange2.File),
					Worktree1:  wt1.ID,
					Worktree2:  wt2.ID,
					Type:       ConflictTypeAPI,
					Severity:   SeverityHigh,
					DetectedAt: time.Now(),
					Metadata: map[string]interface{}{
						"api_change_1": apiChange1,
						"api_change_2": apiChange2,
					},
				})
			}
		}
	}

	return conflicts
}

// readFile reads a file from a worktree
func (cd *ConflictDetector) readFile(wt *Worktree, file string) (string, error) {
	filePath := filepath.Join(wt.Path, file)
	content, err := exec.Command("cat", filePath).Output()
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// getBaseContent gets the base version of a file
func (cd *ConflictDetector) getBaseContent(ctx context.Context, wt *Worktree, file string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "show", fmt.Sprintf("origin/%s:%s", wt.BaseBranch, file))
	cmd.Dir = wt.Path

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// threeWayDiff performs a three-way diff analysis
func (cd *ConflictDetector) threeWayDiff(base, content1, content2 string) *ThreeWayDiff {
	diff := &ThreeWayDiff{
		Base:     base,
		Content1: content1,
		Content2: content2,
	}

	// Simple conflict detection based on line differences
	baseLines := strings.Split(base, "\n")
	lines1 := strings.Split(content1, "\n")
	lines2 := strings.Split(content2, "\n")

	// Find lines that are different in both versions from base
	for i := 0; i < len(baseLines) && i < len(lines1) && i < len(lines2); i++ {
		if baseLines[i] != lines1[i] && baseLines[i] != lines2[i] && lines1[i] != lines2[i] {
			diff.ConflictLines = append(diff.ConflictLines, i+1)
		}
	}

	return diff
}

// extractAPIChanges extracts API-related changes from worktree files
func (cd *ConflictDetector) extractAPIChanges(ctx context.Context, wt *Worktree, files map[string]struct{}) []APIChange {
	var changes []APIChange

	for file := range files {
		// Check if file contains API definitions
		if cd.isAPIFile(file) {
			apiChanges := cd.analyzeAPIFile(ctx, wt, file)
			changes = append(changes, apiChanges...)
		}
	}

	return changes
}

// isAPIFile determines if a file likely contains API definitions
func (cd *ConflictDetector) isAPIFile(file string) bool {
	apiPatterns := []string{
		"interface",
		"api",
		"service",
		"controller",
		"handler",
		"route",
	}

	lowerFile := strings.ToLower(file)
	for _, pattern := range apiPatterns {
		if strings.Contains(lowerFile, pattern) {
			return true
		}
	}

	return false
}

// analyzeAPIFile analyzes a file for API changes
func (cd *ConflictDetector) analyzeAPIFile(ctx context.Context, wt *Worktree, file string) []APIChange {
	// Simplified API change detection
	// Real implementation would parse code and detect function signatures, etc.
	return []APIChange{}
}

// apiChangesConflict determines if two API changes conflict
func (cd *ConflictDetector) apiChangesConflict(api1, api2 APIChange) bool {
	// Simplified conflict detection
	return api1.Function == api2.Function && api1.Type != api2.Type
}

// assessConflictSeverity determines the severity of a conflict
func (cd *ConflictDetector) assessConflictSeverity(diff *ThreeWayDiff) ConflictSeverity {
	if len(diff.ConflictLines) == 0 {
		return SeverityLow
	}
	if len(diff.ConflictLines) < 5 {
		return SeverityMedium
	}
	return SeverityHigh
}

func (cd *ConflictDetector) generateConflictID() string {
	return fmt.Sprintf("conflict_%d", time.Now().UnixNano())
}

// ConflictNotifier handles conflict notifications
type ConflictNotifier struct {
	channel chan ConflictEvent
	mu      sync.RWMutex
}

// NewConflictNotifier creates a new conflict notifier
func NewConflictNotifier() *ConflictNotifier {
	return &ConflictNotifier{
		channel: make(chan ConflictEvent, 100),
	}
}

// NotifyConflict sends a conflict notification
func (cn *ConflictNotifier) NotifyConflict(event ConflictEvent) {
	select {
	case cn.channel <- event:
	default:
		// Channel full, drop event (or log error)
	}
}

// GetChannel returns the event channel for listening to conflicts
func (cn *ConflictNotifier) GetChannel() <-chan ConflictEvent {
	return cn.channel
}

// ConflictEvent represents a conflict event
type ConflictEvent struct {
	Type      ConflictEventType `json:"type"`
	Conflict  Conflict          `json:"conflict"`
	Worktree1 string            `json:"worktree_1"`
	Worktree2 string            `json:"worktree_2"`
	Timestamp time.Time         `json:"timestamp"`
}

// ConflictEventType represents the type of conflict event
type ConflictEventType string

const (
	ConflictEventDetected ConflictEventType = "detected"
	ConflictEventResolved ConflictEventType = "resolved"
	ConflictEventFailed   ConflictEventType = "failed"
)

// Conflict represents a detected conflict between worktrees
type Conflict struct {
	ID         string                 `json:"id"`
	File       string                 `json:"file"`
	Worktree1  string                 `json:"worktree_1"`
	Worktree2  string                 `json:"worktree_2"`
	Type       ConflictType           `json:"type"`
	Severity   ConflictSeverity       `json:"severity"`
	Diff       *ThreeWayDiff          `json:"diff,omitempty"`
	DetectedAt time.Time              `json:"detected_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConflictType represents the type of conflict
type ConflictType string

const (
	ConflictTypeContent ConflictType = "content"
	ConflictTypeAPI     ConflictType = "api"
	ConflictTypeSchema  ConflictType = "schema"
	ConflictTypeConfig  ConflictType = "config"
)

// ConflictSeverity represents the severity of a conflict
type ConflictSeverity string

const (
	SeverityLow    ConflictSeverity = "low"
	SeverityMedium ConflictSeverity = "medium"
	SeverityHigh   ConflictSeverity = "high"
)

// ThreeWayDiff represents a three-way diff result
type ThreeWayDiff struct {
	Base          string `json:"base"`
	Content1      string `json:"content_1"`
	Content2      string `json:"content_2"`
	ConflictLines []int  `json:"conflict_lines"`
}

// HasConflicts returns true if there are conflicts in the diff
func (twd *ThreeWayDiff) HasConflicts() bool {
	return len(twd.ConflictLines) > 0
}

// APIChange represents a change to an API
type APIChange struct {
	File        string `json:"file"`
	Function    string `json:"function"`
	Type        string `json:"type"` // added, removed, modified
	OldSig      string `json:"old_signature,omitempty"`
	NewSig      string `json:"new_signature,omitempty"`
	Description string `json:"description"`
}
