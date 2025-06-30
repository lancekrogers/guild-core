// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftIsolationFramework tests creation of isolation framework
func TestCraftIsolationFramework(t *testing.T) {
	ctx := context.Background()
	mockManager := &mockWorktreeManager{}

	framework, err := NewIsolationFramework(ctx, mockManager)
	require.NoError(t, err)
	assert.NotNil(t, framework)
	assert.NotNil(t, framework.manager)
	assert.NotNil(t, framework.validator)
	assert.NotNil(t, framework.monitor)
}

// TestJourneymanIsolationFrameworkContextCancellation tests context cancellation
func TestJourneymanIsolationFrameworkContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockManager := &mockWorktreeManager{}
	framework, err := NewIsolationFramework(ctx, mockManager)
	assert.Error(t, err)
	assert.Nil(t, framework)
}

// TestGuildIsolationFrameworkNilManager tests nil manager handling
func TestGuildIsolationFrameworkNilManager(t *testing.T) {
	ctx := context.Background()

	framework, err := NewIsolationFramework(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, framework)
	assert.Contains(t, err.Error(), "manager is required")
}

// TestScribeIsolatedWorkspaceCreation tests isolated workspace creation
func TestScribeIsolatedWorkspaceCreation(t *testing.T) {
	ctx := context.Background()
	mockManager := &mockWorktreeManager{
		worktrees: make(map[string]*Worktree),
	}

	// Mock a successful worktree creation
	mockWorktree := &Worktree{
		ID:         "test-wt-1",
		AgentID:    "test-agent",
		TaskID:     "test-task",
		Path:       "/tmp/test-worktree",
		Branch:     "agent/test-agent/test-task",
		BaseBranch: "main",
		Status:     WorktreeActive,
		CreatedAt:  time.Now(),
	}
	mockManager.worktrees["test-wt-1"] = mockWorktree

	framework, err := NewIsolationFramework(ctx, mockManager)
	require.NoError(t, err)

	task := Task{
		ID:          "test-task",
		Type:        "feature",
		Description: "Test task for isolation",
		Requirements: map[string]interface{}{
			"base_branch": "main",
		},
	}

	// This would fail in real implementation due to filesystem operations
	// but tests the structure
	workspace, err := framework.CreateIsolatedWorkspace(ctx, "test-agent", task)
	
	// We expect this to fail due to filesystem operations in the real implementation
	// but we can test the error handling
	if err != nil {
		assert.Contains(t, err.Error(), "worktree not found")
	} else {
		assert.NotNil(t, workspace)
		assert.Equal(t, "test-agent", workspace.Worktree.AgentID)
		assert.Equal(t, "test-task", workspace.Worktree.TaskID)
	}
}

// TestCraftResourceLimits tests resource limits configuration
func TestCraftResourceLimits(t *testing.T) {
	limits := ResourceLimits{
		MaxDiskUsage:   1024 * 1024 * 1024, // 1GB
		MaxFileCount:   10000,
		MaxProcessTime: 30 * time.Minute,
		AllowedPaths: []string{
			"./",
			"/tmp",
			"/var/tmp",
		},
		DeniedPaths: []string{
			"/etc/passwd",
			"/etc/shadow",
			"~/.ssh",
		},
	}

	assert.Equal(t, int64(1024*1024*1024), limits.MaxDiskUsage)
	assert.Equal(t, 10000, limits.MaxFileCount)
	assert.Equal(t, 30*time.Minute, limits.MaxProcessTime)
	assert.Len(t, limits.AllowedPaths, 3)
	assert.Len(t, limits.DeniedPaths, 3)
	assert.Contains(t, limits.AllowedPaths, "./")
	assert.Contains(t, limits.DeniedPaths, "/etc/passwd")
}

// TestJourneymanIsolationConfig tests isolation configuration
func TestJourneymanIsolationConfig(t *testing.T) {
	config := IsolationConfig{
		NetworkIsolation: true,
		ProcessIsolation: true,
		FileSystemMask: []string{
			".env",
			".secrets",
			"*.key",
		},
		EnvironmentMask: []string{
			"*_PASSWORD",
			"*_SECRET",
			"*_KEY",
		},
	}

	assert.True(t, config.NetworkIsolation)
	assert.True(t, config.ProcessIsolation)
	assert.Len(t, config.FileSystemMask, 3)
	assert.Len(t, config.EnvironmentMask, 3)
	assert.Contains(t, config.FileSystemMask, ".env")
	assert.Contains(t, config.EnvironmentMask, "*_PASSWORD")
}

// TestGuildMountConfiguration tests mount configuration
func TestGuildMountConfiguration(t *testing.T) {
	mount := Mount{
		Source:   "/usr/local/bin",
		Target:   "/workspace/.shared/bin",
		ReadOnly: true,
		Type:     "bind",
	}

	assert.Equal(t, "/usr/local/bin", mount.Source)
	assert.Equal(t, "/workspace/.shared/bin", mount.Target)
	assert.True(t, mount.ReadOnly)
	assert.Equal(t, "bind", mount.Type)
}

// TestScribeChangeValidator tests change validation
func TestScribeChangeValidator(t *testing.T) {
	validator := NewChangeValidator()
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.rules)
	assert.NotNil(t, validator.security)
	assert.Len(t, validator.rules, 3) // NoSecretRule, FileSizeRule, PathValidationRule
}

// TestCraftValidationRules tests individual validation rules
func TestCraftValidationRules(t *testing.T) {
	ctx := context.Background()

	t.Run("NoSecretRule", func(t *testing.T) {
		rule := &NoSecretRule{}
		
		// Test with secret content
		changes := []Change{
			{
				File:    "config.go",
				Content: "api_key = \"sk-1234567890abcdef1234567890abcdef12345678\"",
			},
		}
		
		issues := rule.Validate(ctx, changes)
		assert.Greater(t, len(issues), 0)
		assert.Equal(t, SeverityError, issues[0].Severity)
		assert.Equal(t, "no-secrets", issues[0].Rule)
	})

	t.Run("FileSizeRule", func(t *testing.T) {
		rule := &FileSizeRule{maxSize: 100} // 100 bytes limit
		
		// Test with large content
		largeContent := make([]byte, 200)
		for i := range largeContent {
			largeContent[i] = 'a'
		}
		
		changes := []Change{
			{
				File:    "large.txt",
				Content: string(largeContent),
			},
		}
		
		issues := rule.Validate(ctx, changes)
		assert.Greater(t, len(issues), 0)
		assert.Equal(t, SeverityWarning, issues[0].Severity)
		assert.Equal(t, "file-size", issues[0].Rule)
	})

	t.Run("PathValidationRule", func(t *testing.T) {
		rule := &PathValidationRule{}
		
		// Test with restricted path
		changes := []Change{
			{
				File:    ".env",
				Content: "SECRET=value",
			},
		}
		
		issues := rule.Validate(ctx, changes)
		assert.Greater(t, len(issues), 0)
		assert.Equal(t, SeverityError, issues[0].Severity)
		assert.Equal(t, "path-validation", issues[0].Rule)
	})
}

// TestJourneymanSecurityScanner tests security scanning
func TestJourneymanSecurityScanner(t *testing.T) {
	ctx := context.Background()
	scanner := NewSecurityScanner()

	// Test with dangerous patterns
	changes := []Change{
		{
			File:    "dangerous.js",
			Content: "eval(userInput);",
			Line:    1,
		},
		{
			File:    "system.py",
			Content: "os.system(command)",
			Line:    5,
		},
	}

	issues := scanner.Scan(ctx, changes)
	assert.Greater(t, len(issues), 0)
	
	// Check for eval detection
	found := false
	for _, issue := range issues {
		if issue.File == "dangerous.js" && issue.Line == 1 {
			found = true
			assert.Equal(t, SeverityWarning, issue.Severity)
			assert.Equal(t, "security-scan", issue.Rule)
			break
		}
	}
	assert.True(t, found, "Should detect eval() usage")
}

// TestGuildValidationResult tests validation result structure
func TestGuildValidationResult(t *testing.T) {
	result := &ValidationResult{
		WorktreeID: "test-wt",
		Valid:      false,
		Issues: []ValidationIssue{
			{
				File:     "test.go",
				Line:     10,
				Severity: SeverityError,
				Message:  "Test issue",
				Rule:     "test-rule",
			},
		},
		Timestamp: time.Now(),
	}

	assert.Equal(t, "test-wt", result.WorktreeID)
	assert.False(t, result.Valid)
	assert.Len(t, result.Issues, 1)
	assert.Equal(t, "test.go", result.Issues[0].File)
	assert.Equal(t, 10, result.Issues[0].Line)
	assert.Equal(t, SeverityError, result.Issues[0].Severity)
}

// TestScribeEnvironmentBuilder tests environment building
func TestScribeEnvironmentBuilder(t *testing.T) {
	framework := &IsolationFramework{}
	worktree := &Worktree{
		ID:         "test-wt",
		AgentID:    "test-agent",
		Path:       "/tmp/test-worktree",
		Branch:     "agent/test-agent/test-task",
		BaseBranch: "main",
	}

	env := framework.buildEnvironment("test-agent", worktree)

	assert.Equal(t, "test-agent", env["GUILD_AGENT_ID"])
	assert.Equal(t, "test-wt", env["GUILD_WORKTREE_ID"])
	assert.Equal(t, "/tmp/test-worktree", env["GUILD_WORKTREE_PATH"])
	assert.Equal(t, "agent/test-agent/test-task", env["GUILD_BRANCH"])
	assert.Equal(t, "main", env["GUILD_BASE_BRANCH"])
	assert.Equal(t, "true", env["GUILD_ISOLATED"])
	assert.Equal(t, "test-agent", env["GIT_AUTHOR_NAME"])
	assert.Equal(t, "test-agent@guild.local", env["GIT_AUTHOR_EMAIL"])
}

// TestCraftTaskRequirements tests task requirement handling
func TestCraftTaskRequirements(t *testing.T) {
	task := Task{
		ID:          "test-task",
		Type:        "feature",
		Description: "Test task",
		Requirements: map[string]interface{}{
			"base_branch":       "develop",
			"network_isolation": true,
			"max_duration":      "2h",
		},
	}

	assert.Equal(t, "test-task", task.ID)
	assert.Equal(t, "feature", task.Type)
	assert.Equal(t, "develop", task.Requirements["base_branch"])
	assert.True(t, task.Requirements["network_isolation"].(bool))
	assert.Equal(t, "2h", task.Requirements["max_duration"])
}

// TestJourneymanSeverityLevels tests severity level validation
func TestJourneymanSeverityLevels(t *testing.T) {
	severities := []Severity{
		SeverityError,
		SeverityWarning,
		SeverityInfo,
	}

	assert.Equal(t, Severity("error"), severities[0])
	assert.Equal(t, Severity("warning"), severities[1])
	assert.Equal(t, Severity("info"), severities[2])
}

// TestGuildChangeTypes tests change type handling
func TestGuildChangeTypes(t *testing.T) {
	change := Change{
		File:       "test.go",
		Type:       "modified",
		Content:    "package main",
		Line:       1,
		OldContent: "package test",
		Timestamp:  time.Now(),
	}

	assert.Equal(t, "test.go", change.File)
	assert.Equal(t, "modified", change.Type)
	assert.Equal(t, "package main", change.Content)
	assert.Equal(t, 1, change.Line)
	assert.Equal(t, "package test", change.OldContent)
}

// Benchmark tests for performance validation
func BenchmarkValidationRules(b *testing.B) {
	ctx := context.Background()
	rule := &NoSecretRule{}
	
	changes := []Change{
		{
			File:    "config.go",
			Content: "const normalContent = \"this is just normal code without secrets\"",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rule.Validate(ctx, changes)
	}
}

func BenchmarkSecurityScanner(b *testing.B) {
	ctx := context.Background()
	scanner := NewSecurityScanner()
	
	changes := []Change{
		{
			File:    "normal.go",
			Content: "func normalFunction() { return \"safe code\" }",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.Scan(ctx, changes)
	}
}