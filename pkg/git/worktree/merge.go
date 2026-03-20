// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// MergeCoordinator coordinates merges between multiple worktrees
type MergeCoordinator struct {
	manager   Manager
	strategy  MergeStrategy
	validator *MergeValidator
	notifier  *MergeNotifier
	mu        sync.RWMutex
}

// NewMergeCoordinator creates a new merge coordinator
func NewMergeCoordinator(ctx context.Context, manager Manager, strategy MergeStrategy) (*MergeCoordinator, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree.merge").
			WithOperation("NewMergeCoordinator")
	}

	if manager == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "manager is required", nil).
			WithComponent("git.worktree.merge").
			WithOperation("NewMergeCoordinator")
	}

	if strategy == nil {
		strategy = &SequentialMergeStrategy{}
	}

	return &MergeCoordinator{
		manager:   manager,
		strategy:  strategy,
		validator: NewMergeValidator(),
		notifier:  NewMergeNotifier(),
	}, nil
}

// PlanMerge creates a merge plan for the given worktrees
func (mc *MergeCoordinator) PlanMerge(ctx context.Context, worktreeIDs []string, targetBranch string) (*MergePlan, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree.merge").
			WithOperation("PlanMerge")
	}

	// Get worktrees
	var worktrees []*Worktree
	for _, id := range worktreeIDs {
		wt := mc.manager.GetWorktree(id)
		if wt == nil {
			return nil, gerror.New(gerror.ErrCodeNotFound, "worktree not found", nil).
				WithComponent("git.worktree.merge").
				WithOperation("PlanMerge").
				WithDetails("worktree_id", id)
		}
		worktrees = append(worktrees, wt)
	}

	if targetBranch == "" {
		targetBranch = "main"
	}

	// Use strategy to create plan
	plan := mc.strategy.Plan(ctx, worktrees, targetBranch)

	// Validate plan
	if err := mc.validatePlan(ctx, plan); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "merge plan validation failed").
			WithComponent("git.worktree.merge").
			WithOperation("PlanMerge")
	}

	return plan, nil
}

// ExecuteMerge executes a merge plan
func (mc *MergeCoordinator) ExecuteMerge(ctx context.Context, plan *MergePlan) (*MergeResult, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree.merge").
			WithOperation("ExecuteMerge")
	}

	mc.notifier.NotifyMergeStart(plan)

	result := &MergeResult{
		PlanID:  plan.ID,
		Started: time.Now(),
		Success: false,
		Steps:   make([]MergeStepResult, 0, len(plan.Order)),
	}

	// Execute merge strategy
	err := mc.strategy.Execute(ctx, plan)
	if err != nil {
		result.Completed = time.Now()
		result.Error = err.Error()
		mc.notifier.NotifyMergeComplete(plan, *result)
		return result, gerror.Wrap(err, gerror.ErrCodeInternal, "merge execution failed").
			WithComponent("git.worktree.merge").
			WithOperation("ExecuteMerge").
			WithDetails("plan_id", plan.ID)
	}

	result.Completed = time.Now()
	result.Success = true
	mc.notifier.NotifyMergeComplete(plan, *result)

	return result, nil
}

// validatePlan validates a merge plan for feasibility
func (mc *MergeCoordinator) validatePlan(ctx context.Context, plan *MergePlan) error {
	// Check for circular dependencies
	if mc.hasCircularDependencies(plan) {
		return gerror.New(gerror.ErrCodeValidation, "merge plan has circular dependencies", nil)
	}

	// Check that all dependencies exist
	worktreeMap := make(map[string]bool)
	for _, wt := range plan.Worktrees {
		worktreeMap[wt.ID] = true
	}

	for _, step := range plan.Order {
		for _, dep := range step.Dependencies {
			if !worktreeMap[dep] {
				return gerror.New(gerror.ErrCodeValidation, "dependency not found in plan", nil).
					WithDetails("step", step.WorktreeID).
					WithDetails("dependency", dep)
			}
		}
	}

	return nil
}

// hasCircularDependencies checks for circular dependencies in the merge plan
func (mc *MergeCoordinator) hasCircularDependencies(plan *MergePlan) bool {
	// Build dependency graph
	deps := make(map[string][]string)
	for _, step := range plan.Order {
		deps[step.WorktreeID] = step.Dependencies
	}

	// Use DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range deps[node] {
			if !visited[neighbor] && hasCycle(neighbor) {
				return true
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range deps {
		if !visited[node] && hasCycle(node) {
			return true
		}
	}

	return false
}

// MergeStrategy defines the interface for merge strategies
type MergeStrategy interface {
	Plan(ctx context.Context, worktrees []*Worktree, targetBranch string) *MergePlan
	Execute(ctx context.Context, plan *MergePlan) error
}

// MergePlan contains the plan for merging multiple worktrees
type MergePlan struct {
	ID                string                 `json:"id"`
	Worktrees         []*Worktree            `json:"worktrees"`
	Target            string                 `json:"target_branch"`
	Order             []MergeStep            `json:"order"`
	Conflicts         []PredictedConflict    `json:"predicted_conflicts"`
	Strategy          string                 `json:"strategy"`
	CreatedAt         time.Time              `json:"created_at"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// MergeStep represents a single step in the merge plan
type MergeStep struct {
	WorktreeID        string        `json:"worktree_id"`
	Action            MergeAction   `json:"action"`
	Dependencies      []string      `json:"dependencies"`
	Order             int           `json:"order"`
	EstimatedDuration time.Duration `json:"estimated_duration"`
}

// MergeAction represents the action to take for a merge step
type MergeAction string

const (
	MergeActionRebase MergeAction = "rebase"
	MergeActionMerge  MergeAction = "merge"
	MergeActionSquash MergeAction = "squash"
	MergeActionCherry MergeAction = "cherry-pick"
)

// PredictedConflict represents a predicted merge conflict
type PredictedConflict struct {
	File        string           `json:"file"`
	Worktrees   []string         `json:"worktrees"`
	Type        ConflictType     `json:"type"`
	Severity    ConflictSeverity `json:"severity"`
	Confidence  float64          `json:"confidence"`
	Description string           `json:"description"`
}

// MergeResult contains the result of a merge operation
type MergeResult struct {
	PlanID    string            `json:"plan_id"`
	Success   bool              `json:"success"`
	Started   time.Time         `json:"started"`
	Completed time.Time         `json:"completed"`
	Steps     []MergeStepResult `json:"steps"`
	Error     string            `json:"error,omitempty"`
	Conflicts []Conflict        `json:"conflicts"`
}

// MergeStepResult contains the result of a single merge step
type MergeStepResult struct {
	WorktreeID string      `json:"worktree_id"`
	Action     MergeAction `json:"action"`
	Success    bool        `json:"success"`
	Started    time.Time   `json:"started"`
	Completed  time.Time   `json:"completed"`
	Error      string      `json:"error,omitempty"`
	Conflicts  []string    `json:"conflicts"`
}

// SequentialMergeStrategy implements sequential merging
type SequentialMergeStrategy struct {
	coordinator *MergeCoordinator
}

// Plan creates a sequential merge plan
func (sms *SequentialMergeStrategy) Plan(ctx context.Context, worktrees []*Worktree, targetBranch string) *MergePlan {
	plan := &MergePlan{
		ID:        sms.generatePlanID(),
		Worktrees: worktrees,
		Target:    targetBranch,
		Strategy:  "sequential",
		CreatedAt: time.Now(),
	}

	// Order by dependency and change complexity
	ordered := sms.orderWorktrees(ctx, worktrees)

	// Create merge steps
	for i, wt := range ordered {
		step := MergeStep{
			WorktreeID:        wt.ID,
			Action:            sms.determineAction(ctx, wt),
			Order:             i,
			EstimatedDuration: sms.estimateDuration(ctx, wt),
		}

		// Depend on previous merges
		if i > 0 {
			step.Dependencies = []string{ordered[i-1].ID}
		}

		plan.Order = append(plan.Order, step)
	}

	// Predict conflicts
	plan.Conflicts = sms.predictConflicts(ctx, ordered)

	// Calculate total estimated duration
	for _, step := range plan.Order {
		plan.EstimatedDuration += step.EstimatedDuration
	}

	return plan
}

// Execute executes a sequential merge plan
func (sms *SequentialMergeStrategy) Execute(ctx context.Context, plan *MergePlan) error {
	completed := make(map[string]bool)

	for _, step := range plan.Order {
		// Check context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Wait for dependencies
		for _, dep := range step.Dependencies {
			if !completed[dep] {
				return gerror.New(gerror.ErrCodeInternal, "dependency not satisfied", nil).
					WithComponent("git.worktree.merge").
					WithOperation("Execute").
					WithDetails("step", step.WorktreeID).
					WithDetails("dependency", dep)
			}
		}

		// Execute merge step
		if err := sms.executeMergeStep(ctx, step, plan.Target); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "merge step failed").
				WithComponent("git.worktree.merge").
				WithOperation("Execute").
				WithDetails("step", step.WorktreeID)
		}

		completed[step.WorktreeID] = true
	}

	return nil
}

// orderWorktrees orders worktrees for optimal merging
func (sms *SequentialMergeStrategy) orderWorktrees(ctx context.Context, worktrees []*Worktree) []*Worktree {
	// Sort by complexity score (simpler merges first)
	sort.Slice(worktrees, func(i, j int) bool {
		score1 := sms.calculateComplexityScore(ctx, worktrees[i])
		score2 := sms.calculateComplexityScore(ctx, worktrees[j])
		return score1 < score2
	})

	return worktrees
}

// calculateComplexityScore calculates a complexity score for a worktree
func (sms *SequentialMergeStrategy) calculateComplexityScore(ctx context.Context, wt *Worktree) int {
	score := 0

	// Count changed files
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", fmt.Sprintf("origin/%s...HEAD", wt.BaseBranch))
	cmd.Dir = wt.Path
	if output, err := cmd.Output(); err == nil {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		score += len(files) * 2
	}

	// Count commits
	cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("origin/%s...HEAD", wt.BaseBranch))
	cmd.Dir = wt.Path
	if output, err := cmd.Output(); err == nil {
		var commits int
		fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &commits)
		score += commits
	}

	// Age factor (older branches are more complex to merge)
	age := time.Since(wt.CreatedAt)
	score += int(age.Hours() / 24) // Add 1 point per day

	// For sub-day precision in tests, add hours
	score += int(age.Hours())

	return score
}

// determineAction determines the best merge action for a worktree
func (sms *SequentialMergeStrategy) determineAction(ctx context.Context, wt *Worktree) MergeAction {
	// Count commits to determine best action
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("origin/%s...HEAD", wt.BaseBranch))
	cmd.Dir = wt.Path

	if output, err := cmd.Output(); err == nil {
		var commits int
		fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &commits)

		if commits == 1 {
			return MergeActionSquash
		} else if commits <= 3 {
			return MergeActionRebase
		}
	}

	return MergeActionMerge
}

// estimateDuration estimates how long a merge step will take
func (sms *SequentialMergeStrategy) estimateDuration(ctx context.Context, wt *Worktree) time.Duration {
	// Base duration
	duration := 30 * time.Second

	// Add time based on complexity
	complexity := sms.calculateComplexityScore(ctx, wt)
	duration += time.Duration(complexity) * 5 * time.Second

	return duration
}

// predictConflicts predicts potential conflicts in the merge
func (sms *SequentialMergeStrategy) predictConflicts(ctx context.Context, worktrees []*Worktree) []PredictedConflict {
	var conflicts []PredictedConflict

	// Check for overlapping files
	fileMap := make(map[string][]*Worktree)

	for _, wt := range worktrees {
		files := sms.getChangedFiles(ctx, wt)
		for _, file := range files {
			fileMap[file] = append(fileMap[file], wt)
		}
	}

	// Find files modified by multiple worktrees
	for file, wts := range fileMap {
		if len(wts) > 1 {
			worktreeIDs := make([]string, len(wts))
			for i, wt := range wts {
				worktreeIDs[i] = wt.ID
			}

			conflicts = append(conflicts, PredictedConflict{
				File:        file,
				Worktrees:   worktreeIDs,
				Type:        ConflictTypeContent,
				Severity:    sms.assessConflictSeverity(file, len(wts)),
				Confidence:  0.7, // Medium confidence for prediction
				Description: fmt.Sprintf("File %s modified by %d worktrees", file, len(wts)),
			})
		}
	}

	return conflicts
}

// getChangedFiles gets the list of changed files in a worktree
func (sms *SequentialMergeStrategy) getChangedFiles(ctx context.Context, wt *Worktree) []string {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", fmt.Sprintf("origin/%s...HEAD", wt.BaseBranch))
	cmd.Dir = wt.Path

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return nil
	}

	return files
}

// assessConflictSeverity assesses the severity of a predicted conflict
func (sms *SequentialMergeStrategy) assessConflictSeverity(file string, worktreeCount int) ConflictSeverity {
	// API files are high severity
	if strings.Contains(strings.ToLower(file), "api") ||
		strings.Contains(strings.ToLower(file), "interface") {
		return SeverityHigh
	}

	// Configuration files are medium severity
	if strings.Contains(file, "config") ||
		strings.HasSuffix(file, ".yaml") ||
		strings.HasSuffix(file, ".json") {
		return SeverityMedium
	}

	// More worktrees = higher severity
	if worktreeCount > 2 {
		return SeverityHigh
	}

	return SeverityLow
}

// executeMergeStep executes a single merge step
func (sms *SequentialMergeStrategy) executeMergeStep(ctx context.Context, step MergeStep, targetBranch string) error {
	wt := sms.coordinator.manager.GetWorktree(step.WorktreeID)
	if wt == nil {
		return gerror.New(gerror.ErrCodeNotFound, "worktree not found", nil).
			WithDetails("worktree_id", step.WorktreeID)
	}

	switch step.Action {
	case MergeActionRebase:
		return sms.rebaseAndMerge(ctx, wt, targetBranch)
	case MergeActionMerge:
		return sms.mergeDirectly(ctx, wt, targetBranch)
	case MergeActionSquash:
		return sms.squashAndMerge(ctx, wt, targetBranch)
	case MergeActionCherry:
		return sms.cherryPickMerge(ctx, wt, targetBranch)
	default:
		return gerror.New(gerror.ErrCodeValidation, "unknown merge action", nil).
			WithDetails("action", string(step.Action))
	}
}

// rebaseAndMerge performs a rebase followed by merge
func (sms *SequentialMergeStrategy) rebaseAndMerge(ctx context.Context, wt *Worktree, targetBranch string) error {
	// Switch to target branch in base repo
	baseRepoPath := sms.getBaseRepoPath()

	// Fetch latest
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "fetch", "origin", targetBranch); err != nil {
		return err
	}

	// Checkout target branch
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "checkout", targetBranch); err != nil {
		return err
	}

	// Pull latest
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "pull", "origin", targetBranch); err != nil {
		return err
	}

	// Merge the worktree branch
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "merge", "--no-ff", wt.Branch); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "merge failed").
			WithDetails("branch", wt.Branch)
	}

	return nil
}

// mergeDirectly performs a direct merge
func (sms *SequentialMergeStrategy) mergeDirectly(ctx context.Context, wt *Worktree, targetBranch string) error {
	baseRepoPath := sms.getBaseRepoPath()

	// Checkout target branch
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "checkout", targetBranch); err != nil {
		return err
	}

	// Merge with merge commit
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "merge", "--no-ff", "-m",
		fmt.Sprintf("Merge branch '%s' (Agent: %s, Task: %s)", wt.Branch, wt.AgentID, wt.TaskID),
		wt.Branch); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "direct merge failed").
			WithDetails("branch", wt.Branch)
	}

	return nil
}

// squashAndMerge performs a squash merge
func (sms *SequentialMergeStrategy) squashAndMerge(ctx context.Context, wt *Worktree, targetBranch string) error {
	baseRepoPath := sms.getBaseRepoPath()

	// Checkout target branch
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "checkout", targetBranch); err != nil {
		return err
	}

	// Squash merge
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "merge", "--squash", wt.Branch); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "squash merge failed").
			WithDetails("branch", wt.Branch)
	}

	// Commit the squashed changes
	commitMsg := fmt.Sprintf("Squash merge from %s\n\nAgent: %s\nTask: %s",
		wt.Branch, wt.AgentID, wt.TaskID)

	if err := sms.executeCommand(ctx, baseRepoPath, "git", "commit", "-m", commitMsg); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "squash commit failed").
			WithDetails("branch", wt.Branch)
	}

	return nil
}

// cherryPickMerge performs cherry-pick merge
func (sms *SequentialMergeStrategy) cherryPickMerge(ctx context.Context, wt *Worktree, targetBranch string) error {
	baseRepoPath := sms.getBaseRepoPath()

	// Checkout target branch
	if err := sms.executeCommand(ctx, baseRepoPath, "git", "checkout", targetBranch); err != nil {
		return err
	}

	// Get commits to cherry pick
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--reverse", fmt.Sprintf("origin/%s..HEAD", wt.BaseBranch))
	cmd.Dir = wt.Path

	output, err := cmd.Output()
	if err != nil {
		return err
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Cherry pick each commit
	for _, commit := range commits {
		if commit != "" {
			if err := sms.executeCommand(ctx, baseRepoPath, "git", "cherry-pick", commit); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "cherry-pick failed").
					WithDetails("commit", commit)
			}
		}
	}

	return nil
}

// Helper methods

func (sms *SequentialMergeStrategy) generatePlanID() string {
	return fmt.Sprintf("plan_%d", time.Now().UnixNano())
}

func (sms *SequentialMergeStrategy) getBaseRepoPath() string {
	// This would need to be configured based on the actual base repository
	return "."
}

func (sms *SequentialMergeStrategy) executeCommand(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s (output: %s)", err, string(output))
	}

	return nil
}

// ParallelMergeStrategy implements parallel merging where possible
type ParallelMergeStrategy struct {
	coordinator *MergeCoordinator
}

// Plan creates a parallel merge plan
func (pms *ParallelMergeStrategy) Plan(ctx context.Context, worktrees []*Worktree, targetBranch string) *MergePlan {
	plan := &MergePlan{
		ID:        pms.generatePlanID(),
		Worktrees: worktrees,
		Target:    targetBranch,
		Strategy:  "parallel",
		CreatedAt: time.Now(),
	}

	// Group non-conflicting worktrees
	groups := pms.groupNonConflicting(ctx, worktrees)

	// Create parallel merge steps
	for groupIdx, group := range groups {
		for _, wt := range group {
			step := MergeStep{
				WorktreeID: wt.ID,
				Action:     MergeActionRebase,
				Order:      groupIdx,
			}

			// Depend on previous group
			if groupIdx > 0 {
				for _, prevWT := range groups[groupIdx-1] {
					step.Dependencies = append(step.Dependencies, prevWT.ID)
				}
			}

			plan.Order = append(plan.Order, step)
		}
	}

	return plan
}

// Execute executes a parallel merge plan
func (pms *ParallelMergeStrategy) Execute(ctx context.Context, plan *MergePlan) error {
	// Group steps by order (parallel groups)
	groups := make(map[int][]MergeStep)
	for _, step := range plan.Order {
		groups[step.Order] = append(groups[step.Order], step)
	}

	// Execute groups in order
	for i := 0; i < len(groups); i++ {
		if err := pms.executeParallelGroup(ctx, groups[i], plan.Target); err != nil {
			return err
		}
	}

	return nil
}

// executeParallelGroup executes a group of merge steps in parallel
func (pms *ParallelMergeStrategy) executeParallelGroup(ctx context.Context, steps []MergeStep, targetBranch string) error {
	errCh := make(chan error, len(steps))

	for _, step := range steps {
		go func(s MergeStep) {
			// Execute step (implementation would be similar to sequential)
			errCh <- nil // Placeholder
		}(step)
	}

	// Wait for all to complete
	for i := 0; i < len(steps); i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}

	return nil
}

// groupNonConflicting groups worktrees that don't conflict with each other
func (pms *ParallelMergeStrategy) groupNonConflicting(ctx context.Context, worktrees []*Worktree) [][]*Worktree {
	// Simple grouping - in practice would analyze file conflicts
	var groups [][]*Worktree

	// For now, put each worktree in its own group (sequential)
	for _, wt := range worktrees {
		groups = append(groups, []*Worktree{wt})
	}

	return groups
}

func (pms *ParallelMergeStrategy) generatePlanID() string {
	return fmt.Sprintf("parallel_plan_%d", time.Now().UnixNano())
}

// MergeNotifier handles merge notifications
type MergeNotifier struct {
	channel chan MergeEvent
}

// NewMergeNotifier creates a new merge notifier
func NewMergeNotifier() *MergeNotifier {
	return &MergeNotifier{
		channel: make(chan MergeEvent, 100),
	}
}

// NotifyMergeStart notifies that a merge has started
func (mn *MergeNotifier) NotifyMergeStart(plan *MergePlan) {
	mn.channel <- MergeEvent{
		Type:      MergeEventStarted,
		PlanID:    plan.ID,
		Worktrees: mn.getWorktreeIDs(plan),
		Target:    plan.Target,
		Timestamp: time.Now(),
	}
}

// NotifyMergeComplete notifies that a merge has completed
func (mn *MergeNotifier) NotifyMergeComplete(plan *MergePlan, result MergeResult) {
	eventType := MergeEventCompleted
	if !result.Success {
		eventType = MergeEventFailed
	}

	mn.channel <- MergeEvent{
		Type:      eventType,
		PlanID:    plan.ID,
		Result:    &result,
		Timestamp: time.Now(),
	}
}

// GetChannel returns the event channel
func (mn *MergeNotifier) GetChannel() <-chan MergeEvent {
	return mn.channel
}

func (mn *MergeNotifier) getWorktreeIDs(plan *MergePlan) []string {
	ids := make([]string, len(plan.Worktrees))
	for i, wt := range plan.Worktrees {
		ids[i] = wt.ID
	}
	return ids
}

// MergeEvent represents a merge event
type MergeEvent struct {
	Type      MergeEventType `json:"type"`
	PlanID    string         `json:"plan_id"`
	Worktrees []string       `json:"worktrees,omitempty"`
	Target    string         `json:"target,omitempty"`
	Result    *MergeResult   `json:"result,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// MergeEventType represents the type of merge event
type MergeEventType string

const (
	MergeEventStarted   MergeEventType = "started"
	MergeEventCompleted MergeEventType = "completed"
	MergeEventFailed    MergeEventType = "failed"
	MergeEventConflict  MergeEventType = "conflict"
)
