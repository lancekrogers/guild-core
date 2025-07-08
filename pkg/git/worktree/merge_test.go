// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftMergeCoordinator tests creation of merge coordinator
func TestCraftMergeCoordinator(t *testing.T) {
	ctx := context.Background()
	mockManager := &mockWorktreeManager{}
	strategy := &SequentialMergeStrategy{}

	coordinator, err := NewMergeCoordinator(ctx, mockManager, strategy)
	require.NoError(t, err)
	assert.NotNil(t, coordinator)
	assert.NotNil(t, coordinator.manager)
	assert.NotNil(t, coordinator.strategy)
	assert.NotNil(t, coordinator.validator)
	assert.NotNil(t, coordinator.notifier)
}

// TestJourneymanMergeCoordinatorContextCancellation tests context cancellation
func TestJourneymanMergeCoordinatorContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockManager := &mockWorktreeManager{}
	strategy := &SequentialMergeStrategy{}

	coordinator, err := NewMergeCoordinator(ctx, mockManager, strategy)
	assert.Error(t, err)
	assert.Nil(t, coordinator)
}

// TestGuildMergeCoordinatorNilManager tests nil manager handling
func TestGuildMergeCoordinatorNilManager(t *testing.T) {
	ctx := context.Background()
	strategy := &SequentialMergeStrategy{}

	coordinator, err := NewMergeCoordinator(ctx, nil, strategy)
	assert.Error(t, err)
	assert.Nil(t, coordinator)
	assert.Contains(t, err.Error(), "manager is required")
}

// TestScribeMergePlanCreation tests merge plan creation
func TestScribeMergePlanCreation(t *testing.T) {
	ctx := context.Background()

	// Create test worktrees
	worktrees := []*Worktree{
		{
			ID:         "wt1",
			AgentID:    "agent1",
			TaskID:     "task1",
			Branch:     "agent/agent1/task1",
			BaseBranch: "main",
			Status:     WorktreeActive,
			CreatedAt:  time.Now().Add(-1 * time.Hour),
		},
		{
			ID:         "wt2",
			AgentID:    "agent2",
			TaskID:     "task2",
			Branch:     "agent/agent2/task2",
			BaseBranch: "main",
			Status:     WorktreeActive,
			CreatedAt:  time.Now().Add(-2 * time.Hour),
		},
	}

	strategy := &SequentialMergeStrategy{}
	plan := strategy.Plan(ctx, worktrees, "main")

	assert.NotEmpty(t, plan.ID)
	assert.Equal(t, "main", plan.Target)
	assert.Equal(t, "sequential", plan.Strategy)
	assert.Len(t, plan.Worktrees, 2)
	assert.Len(t, plan.Order, 2)
	assert.Greater(t, plan.EstimatedDuration, time.Duration(0))

	// Verify ordering (newer worktree should be first due to lower complexity score)
	assert.Equal(t, "wt1", plan.Order[0].WorktreeID) // Newer worktree (lower complexity)
	assert.Equal(t, "wt2", plan.Order[1].WorktreeID) // Older worktree (higher complexity)

	// Verify dependencies
	assert.Empty(t, plan.Order[0].Dependencies)
	assert.Contains(t, plan.Order[1].Dependencies, "wt1")
}

// TestCraftMergeActions tests merge action determination
func TestCraftMergeActions(t *testing.T) {
	actions := []MergeAction{
		MergeActionRebase,
		MergeActionMerge,
		MergeActionSquash,
		MergeActionCherry,
	}

	assert.Equal(t, MergeAction("rebase"), actions[0])
	assert.Equal(t, MergeAction("merge"), actions[1])
	assert.Equal(t, MergeAction("squash"), actions[2])
	assert.Equal(t, MergeAction("cherry-pick"), actions[3])
}

// TestJourneymanMergeStepValidation tests merge step validation
func TestJourneymanMergeStepValidation(t *testing.T) {
	step := MergeStep{
		WorktreeID:        "wt1",
		Action:            MergeActionRebase,
		Dependencies:      []string{"wt2"},
		Order:             1,
		EstimatedDuration: 30 * time.Second,
	}

	assert.Equal(t, "wt1", step.WorktreeID)
	assert.Equal(t, MergeActionRebase, step.Action)
	assert.Len(t, step.Dependencies, 1)
	assert.Contains(t, step.Dependencies, "wt2")
	assert.Equal(t, 1, step.Order)
	assert.Equal(t, 30*time.Second, step.EstimatedDuration)
}

// TestGuildPredictedConflicts tests conflict prediction
func TestGuildPredictedConflicts(t *testing.T) {
	conflict := PredictedConflict{
		File:        "main.go",
		Worktrees:   []string{"wt1", "wt2"},
		Type:        ConflictTypeContent,
		Severity:    SeverityMedium,
		Confidence:  0.7,
		Description: "Both worktrees modified main.go",
	}

	assert.Equal(t, "main.go", conflict.File)
	assert.Len(t, conflict.Worktrees, 2)
	assert.Equal(t, ConflictTypeContent, conflict.Type)
	assert.Equal(t, SeverityMedium, conflict.Severity)
	assert.Equal(t, 0.7, conflict.Confidence)
}

// TestScribeMergeResult tests merge result creation
func TestScribeMergeResult(t *testing.T) {
	started := time.Now()
	completed := started.Add(5 * time.Minute)

	result := &MergeResult{
		PlanID:    "plan123",
		Success:   true,
		Started:   started,
		Completed: completed,
		Steps: []MergeStepResult{
			{
				WorktreeID: "wt1",
				Action:     MergeActionRebase,
				Success:    true,
				Started:    started,
				Completed:  started.Add(2 * time.Minute),
			},
		},
	}

	assert.Equal(t, "plan123", result.PlanID)
	assert.True(t, result.Success)
	assert.Equal(t, started, result.Started)
	assert.Equal(t, completed, result.Completed)
	assert.Len(t, result.Steps, 1)
	assert.Equal(t, "wt1", result.Steps[0].WorktreeID)
	assert.True(t, result.Steps[0].Success)
}

// TestCraftSequentialMergeStrategy tests sequential merge strategy
func TestCraftSequentialMergeStrategy(t *testing.T) {
	ctx := context.Background()
	strategy := &SequentialMergeStrategy{}

	worktrees := []*Worktree{
		{
			ID:         "wt1",
			AgentID:    "agent1",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			Path:       "/tmp/wt1",
			BaseBranch: "main",
		},
		{
			ID:         "wt2",
			AgentID:    "agent2",
			CreatedAt:  time.Now().Add(-3 * time.Hour), // Older
			Path:       "/tmp/wt2",
			BaseBranch: "main",
		},
	}

	// Test ordering (should be by complexity/age)
	ordered := strategy.orderWorktrees(ctx, worktrees)
	assert.Len(t, ordered, 2)

	// Test complexity calculation
	score1 := strategy.calculateComplexityScore(ctx, worktrees[0])
	score2 := strategy.calculateComplexityScore(ctx, worktrees[1])

	// Older worktree should have higher complexity score
	assert.Greater(t, score2, score1)

	// Test action determination
	action := strategy.determineAction(ctx, worktrees[0])
	assert.Contains(t, []MergeAction{MergeActionRebase, MergeActionMerge, MergeActionSquash}, action)

	// Test duration estimation
	duration := strategy.estimateDuration(ctx, worktrees[0])
	assert.Greater(t, duration, time.Duration(0))
}

// TestJourneymanParallelMergeStrategy tests parallel merge strategy
func TestJourneymanParallelMergeStrategy(t *testing.T) {
	ctx := context.Background()
	strategy := &ParallelMergeStrategy{}

	worktrees := []*Worktree{
		{ID: "wt1", AgentID: "agent1"},
		{ID: "wt2", AgentID: "agent2"},
		{ID: "wt3", AgentID: "agent3"},
	}

	plan := strategy.Plan(ctx, worktrees, "main")

	assert.Equal(t, "parallel", plan.Strategy)
	assert.Equal(t, "main", plan.Target)
	assert.Len(t, plan.Worktrees, 3)

	// Test grouping
	groups := strategy.groupNonConflicting(ctx, worktrees)
	assert.Greater(t, len(groups), 0)
}

// TestGuildMergeValidator tests merge validation
func TestGuildMergeValidator(t *testing.T) {
	validator := NewMergeValidator()
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.testRunner)
	assert.NotNil(t, validator.codeQuality)
}

// TestScribeMergeNotifier tests merge notifications
func TestScribeMergeNotifier(t *testing.T) {
	notifier := NewMergeNotifier()
	assert.NotNil(t, notifier)
	assert.NotNil(t, notifier.channel)

	// Test channel access
	ch := notifier.GetChannel()
	assert.NotNil(t, ch)

	// Test notification
	plan := &MergePlan{
		ID:        "test-plan",
		Target:    "main",
		Worktrees: []*Worktree{{ID: "wt1"}},
	}

	notifier.NotifyMergeStart(plan)

	// Check if event was sent (non-blocking read)
	select {
	case event := <-ch:
		assert.Equal(t, MergeEventStarted, event.Type)
		assert.Equal(t, "test-plan", event.PlanID)
		assert.Equal(t, "main", event.Target)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected merge start event")
	}
}

// TestCraftMergeEventTypes tests merge event types
func TestCraftMergeEventTypes(t *testing.T) {
	events := []MergeEventType{
		MergeEventStarted,
		MergeEventCompleted,
		MergeEventFailed,
		MergeEventConflict,
	}

	assert.Equal(t, MergeEventType("started"), events[0])
	assert.Equal(t, MergeEventType("completed"), events[1])
	assert.Equal(t, MergeEventType("failed"), events[2])
	assert.Equal(t, MergeEventType("conflict"), events[3])
}

// TestJourneymanCircularDependencyDetection tests circular dependency detection
func TestJourneymanCircularDependencyDetection(t *testing.T) {
	mockManager := &mockWorktreeManager{}
	coordinator := &MergeCoordinator{manager: mockManager}

	// Test plan with circular dependencies
	circularPlan := &MergePlan{
		Order: []MergeStep{
			{
				WorktreeID:   "wt1",
				Dependencies: []string{"wt2"},
			},
			{
				WorktreeID:   "wt2",
				Dependencies: []string{"wt3"},
			},
			{
				WorktreeID:   "wt3",
				Dependencies: []string{"wt1"}, // Circular!
			},
		},
		Worktrees: []*Worktree{
			{ID: "wt1"},
			{ID: "wt2"},
			{ID: "wt3"},
		},
	}

	hasCircular := coordinator.hasCircularDependencies(circularPlan)
	assert.True(t, hasCircular)

	// Test plan without circular dependencies
	linearPlan := &MergePlan{
		Order: []MergeStep{
			{
				WorktreeID:   "wt1",
				Dependencies: []string{},
			},
			{
				WorktreeID:   "wt2",
				Dependencies: []string{"wt1"},
			},
			{
				WorktreeID:   "wt3",
				Dependencies: []string{"wt2"},
			},
		},
		Worktrees: []*Worktree{
			{ID: "wt1"},
			{ID: "wt2"},
			{ID: "wt3"},
		},
	}

	hasCircular = coordinator.hasCircularDependencies(linearPlan)
	assert.False(t, hasCircular)
}

// TestGuildMergePlanValidation tests merge plan validation
func TestGuildMergePlanValidation(t *testing.T) {
	ctx := context.Background()
	mockManager := &mockWorktreeManager{}
	coordinator := &MergeCoordinator{manager: mockManager}

	// Valid plan
	validPlan := &MergePlan{
		Order: []MergeStep{
			{
				WorktreeID:   "wt1",
				Dependencies: []string{},
			},
			{
				WorktreeID:   "wt2",
				Dependencies: []string{"wt1"},
			},
		},
		Worktrees: []*Worktree{
			{ID: "wt1"},
			{ID: "wt2"},
		},
	}

	err := coordinator.validatePlan(ctx, validPlan)
	assert.NoError(t, err)

	// Invalid plan with missing dependency
	invalidPlan := &MergePlan{
		Order: []MergeStep{
			{
				WorktreeID:   "wt1",
				Dependencies: []string{"wt_missing"}, // Dependency not in worktrees
			},
		},
		Worktrees: []*Worktree{
			{ID: "wt1"},
		},
	}

	err = coordinator.validatePlan(ctx, invalidPlan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency not found")
}

// TestScribeConflictSeverityAssessment tests conflict severity assessment
func TestScribeConflictSeverityAssessment(t *testing.T) {
	strategy := &SequentialMergeStrategy{}

	tests := []struct {
		name             string
		file             string
		worktreeCount    int
		expectedSeverity ConflictSeverity
	}{
		{
			name:             "API file high severity",
			file:             "api/user.go",
			worktreeCount:    2,
			expectedSeverity: SeverityHigh,
		},
		{
			name:             "Interface file high severity",
			file:             "interfaces/service.go",
			worktreeCount:    2,
			expectedSeverity: SeverityHigh,
		},
		{
			name:             "Config file medium severity",
			file:             "config.yaml",
			worktreeCount:    2,
			expectedSeverity: SeverityMedium,
		},
		{
			name:             "Many worktrees high severity",
			file:             "normal.go",
			worktreeCount:    3,
			expectedSeverity: SeverityHigh,
		},
		{
			name:             "Normal file low severity",
			file:             "util.go",
			worktreeCount:    2,
			expectedSeverity: SeverityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := strategy.assessConflictSeverity(tt.file, tt.worktreeCount)
			assert.Equal(t, tt.expectedSeverity, severity)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkMergePlanCreation(b *testing.B) {
	ctx := context.Background()
	strategy := &SequentialMergeStrategy{}

	worktrees := make([]*Worktree, 10)
	for i := 0; i < 10; i++ {
		worktrees[i] = &Worktree{
			ID:         fmt.Sprintf("wt%d", i),
			AgentID:    fmt.Sprintf("agent%d", i),
			CreatedAt:  time.Now().Add(-time.Duration(i) * time.Hour),
			BaseBranch: "main",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.Plan(ctx, worktrees, "main")
	}
}

func BenchmarkCircularDependencyDetection(b *testing.B) {
	coordinator := &MergeCoordinator{}

	plan := &MergePlan{
		Order:     make([]MergeStep, 20),
		Worktrees: make([]*Worktree, 20),
	}

	// Create linear dependencies
	for i := 0; i < 20; i++ {
		plan.Worktrees[i] = &Worktree{ID: fmt.Sprintf("wt%d", i)}
		plan.Order[i] = MergeStep{
			WorktreeID: fmt.Sprintf("wt%d", i),
		}
		if i > 0 {
			plan.Order[i].Dependencies = []string{fmt.Sprintf("wt%d", i-1)}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coordinator.hasCircularDependencies(plan)
	}
}
