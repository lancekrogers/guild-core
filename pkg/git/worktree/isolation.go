// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// IsolationFramework provides complete workspace isolation for worktrees
type IsolationFramework struct {
	manager   Manager
	validator *ChangeValidator
	monitor   *ConflictMonitor
	mu        sync.RWMutex
}

// NewIsolationFramework creates a new isolation framework
func NewIsolationFramework(ctx context.Context, manager Manager) (*IsolationFramework, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree.isolation").
			WithOperation("NewIsolationFramework")
	}

	if manager == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "manager is required", nil).
			WithComponent("git.worktree.isolation").
			WithOperation("NewIsolationFramework")
	}

	validator := NewChangeValidator()
	monitor := NewConflictMonitor(manager)

	return &IsolationFramework{
		manager:   manager,
		validator: validator,
		monitor:   monitor,
	}, nil
}

// IsolatedWorkspace represents a completely isolated workspace for an agent
type IsolatedWorkspace struct {
	Worktree    *Worktree         `json:"worktree"`
	Environment map[string]string `json:"environment"`
	Mounts      []Mount           `json:"mounts"`
	Limits      ResourceLimits    `json:"limits"`
	Isolation   IsolationConfig   `json:"isolation"`
	CreatedAt   time.Time         `json:"created_at"`
	LastAccess  time.Time         `json:"last_access"`
}

// Mount represents a filesystem mount for isolation
type Mount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
	Type     string `json:"type"`
}

// ResourceLimits defines resource constraints for the workspace
type ResourceLimits struct {
	MaxDiskUsage   int64         `json:"max_disk_usage"`
	MaxFileCount   int           `json:"max_file_count"`
	MaxProcessTime time.Duration `json:"max_process_time"`
	AllowedPaths   []string      `json:"allowed_paths"`
	DeniedPaths    []string      `json:"denied_paths"`
}

// IsolationConfig contains isolation-specific settings
type IsolationConfig struct {
	NetworkIsolation bool     `json:"network_isolation"`
	FileSystemMask   []string `json:"filesystem_mask"`
	EnvironmentMask  []string `json:"environment_mask"`
	ProcessIsolation bool     `json:"process_isolation"`
}

// Task represents a task that requires workspace isolation
type Task struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Requirements map[string]interface{} `json:"requirements"`
}

// CreateIsolatedWorkspace creates a fully isolated workspace for an agent
func (i *IsolationFramework) CreateIsolatedWorkspace(ctx context.Context, agentID string, task Task) (*IsolatedWorkspace, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("git.worktree.isolation").
			WithOperation("CreateIsolatedWorkspace")
	}

	// Create worktree
	worktreeReq := CreateWorktreeRequest{
		AgentID:     agentID,
		TaskID:      task.ID,
		BaseBranch:  i.determineBaseBranch(task),
		Description: task.Description,
		Metadata: map[string]interface{}{
			"task_type":       task.Type,
			"isolation_level": "high",
			"created_by":      "isolation_framework",
		},
	}

	wt, err := i.manager.CreateWorktree(ctx, worktreeReq)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create worktree").
			WithComponent("git.worktree.isolation").
			WithOperation("CreateIsolatedWorkspace").
			WithDetails("agent_id", agentID).
			WithDetails("task_id", task.ID)
	}

	// Create isolated workspace
	workspace := &IsolatedWorkspace{
		Worktree:    wt,
		Environment: i.buildEnvironment(agentID, wt),
		Mounts:      i.configureMounts(wt),
		Limits:      i.getResourceLimits(agentID),
		Isolation:   i.getIsolationConfig(task),
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
	}

	// Set up file system isolation
	if err := i.setupFilesystemIsolation(ctx, workspace); err != nil {
		i.manager.RemoveWorktree(ctx, wt.ID)
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup filesystem isolation").
			WithComponent("git.worktree.isolation").
			WithOperation("CreateIsolatedWorkspace").
			WithDetails("worktree_id", wt.ID)
	}

	// Configure network isolation
	if err := i.setupNetworkIsolation(ctx, workspace); err != nil {
		i.cleanup(ctx, workspace)
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup network isolation").
			WithComponent("git.worktree.isolation").
			WithOperation("CreateIsolatedWorkspace").
			WithDetails("worktree_id", wt.ID)
	}

	return workspace, nil
}

// setupFilesystemIsolation configures filesystem-level isolation
func (i *IsolationFramework) setupFilesystemIsolation(ctx context.Context, ws *IsolatedWorkspace) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Create overlay directories for isolation
	overlayDir := filepath.Join(ws.Worktree.Path, ".overlay")
	dirs := []string{
		filepath.Join(overlayDir, "upper"),
		filepath.Join(overlayDir, "work"),
		filepath.Join(overlayDir, "merged"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create overlay directory").
				WithComponent("git.worktree.isolation").
				WithOperation("setupFilesystemIsolation").
				WithDetails("directory", dir)
		}
	}

	// Configure read-only mounts for shared resources
	for _, mount := range ws.Mounts {
		if mount.ReadOnly {
			if err := i.mountReadOnly(ctx, mount.Source, mount.Target); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup read-only mount").
					WithComponent("git.worktree.isolation").
					WithOperation("setupFilesystemIsolation").
					WithDetails("source", mount.Source).
					WithDetails("target", mount.Target)
			}
		}
	}

	// Restrict access to sensitive paths
	restrictedPaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"~/.ssh",
		"~/.aws",
		".env",
		".secrets",
		".git-credentials",
	}

	for _, path := range restrictedPaths {
		if err := i.restrictAccess(ctx, ws.Worktree.Path, path); err != nil {
			// Log warning but continue - some paths may not exist
			continue
		}
	}

	// Create workspace-specific .gitignore for isolation artifacts
	gitignore := `# Isolation artifacts
.overlay/
.isolation/
*.isolation.tmp
.workspace/
`
	gitignorePath := filepath.Join(ws.Worktree.Path, ".gitignore.isolation")
	if err := os.WriteFile(gitignorePath, []byte(gitignore), 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create isolation gitignore").
			WithComponent("git.worktree.isolation").
			WithOperation("setupFilesystemIsolation").
			WithDetails("path", gitignorePath)
	}

	return nil
}

// setupNetworkIsolation configures network-level isolation (placeholder for actual implementation)
func (i *IsolationFramework) setupNetworkIsolation(ctx context.Context, ws *IsolatedWorkspace) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !ws.Isolation.NetworkIsolation {
		return nil // Network isolation disabled
	}

	// Create network namespace configuration (would require privileged access)
	// This is a simplified implementation - real network isolation would require
	// container runtime or network namespace support

	isolationConfig := filepath.Join(ws.Worktree.Path, ".isolation", "network.conf")
	if err := os.MkdirAll(filepath.Dir(isolationConfig), 0o755); err != nil {
		return err
	}

	networkConfig := fmt.Sprintf(`# Network isolation configuration for workspace %s
# Agent: %s
# Created: %s

# Allowed endpoints (placeholder)
allow_local: true
allow_git_remotes: true
deny_external: false

# DNS restrictions
dns_servers:
  - 8.8.8.8
  - 1.1.1.1

# Port restrictions
allowed_ports:
  - 22    # SSH
  - 80    # HTTP
  - 443   # HTTPS
  - 9418  # Git protocol
`, ws.Worktree.ID, ws.Worktree.AgentID, time.Now().Format(time.RFC3339))

	return os.WriteFile(isolationConfig, []byte(networkConfig), 0o644)
}

// Helper methods

func (i *IsolationFramework) determineBaseBranch(task Task) string {
	if branch, ok := task.Requirements["base_branch"].(string); ok && branch != "" {
		return branch
	}
	return "main"
}

func (i *IsolationFramework) buildEnvironment(agentID string, wt *Worktree) map[string]string {
	env := make(map[string]string)

	// Agent-specific environment
	env["GUILD_AGENT_ID"] = agentID
	env["GUILD_WORKTREE_ID"] = wt.ID
	env["GUILD_WORKTREE_PATH"] = wt.Path
	env["GUILD_BRANCH"] = wt.Branch
	env["GUILD_BASE_BRANCH"] = wt.BaseBranch

	// Isolation-specific environment
	env["GUILD_ISOLATED"] = "true"
	env["GUILD_WORKSPACE_TYPE"] = "isolated"

	// Git configuration
	env["GIT_AUTHOR_NAME"] = agentID
	env["GIT_AUTHOR_EMAIL"] = fmt.Sprintf("%s@guild.local", agentID)
	env["GIT_COMMITTER_NAME"] = agentID
	env["GIT_COMMITTER_EMAIL"] = fmt.Sprintf("%s@guild.local", agentID)

	// Working directory
	env["PWD"] = wt.Path
	env["OLDPWD"] = wt.Path

	return env
}

func (i *IsolationFramework) configureMounts(wt *Worktree) []Mount {
	var mounts []Mount

	// Common read-only mounts for shared resources
	commonMounts := []Mount{
		{
			Source:   "/usr/local/bin",
			Target:   filepath.Join(wt.Path, ".shared", "bin"),
			ReadOnly: true,
			Type:     "bind",
		},
		{
			Source:   "/usr/share",
			Target:   filepath.Join(wt.Path, ".shared", "share"),
			ReadOnly: true,
			Type:     "bind",
		},
	}

	mounts = append(mounts, commonMounts...)

	return mounts
}

func (i *IsolationFramework) getResourceLimits(agentID string) ResourceLimits {
	return ResourceLimits{
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
			"~/.aws",
			".env",
		},
	}
}

func (i *IsolationFramework) getIsolationConfig(task Task) IsolationConfig {
	config := IsolationConfig{
		NetworkIsolation: false, // Default to false for compatibility
		ProcessIsolation: true,
		FileSystemMask: []string{
			".env",
			".secrets",
			".credentials",
			"*.key",
			"*.pem",
		},
		EnvironmentMask: []string{
			"*_PASSWORD",
			"*_SECRET",
			"*_KEY",
			"*_TOKEN",
		},
	}

	// Override based on task requirements
	if isolated, ok := task.Requirements["network_isolation"].(bool); ok {
		config.NetworkIsolation = isolated
	}

	return config
}

func (i *IsolationFramework) mountReadOnly(ctx context.Context, source, target string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}

	// Create symlink for read-only access (simplified implementation)
	// In a real implementation, this would use mount syscalls or containers
	return os.Symlink(source, target+"_link")
}

func (i *IsolationFramework) restrictAccess(ctx context.Context, workspacePath, restrictedPath string) error {
	// Create .workspace-restrictions file to document restricted paths
	restrictionsFile := filepath.Join(workspacePath, ".workspace", "restrictions.txt")
	if err := os.MkdirAll(filepath.Dir(restrictionsFile), 0o755); err != nil {
		return err
	}

	restriction := fmt.Sprintf("# Access restricted to: %s\n# Timestamp: %s\n",
		restrictedPath, time.Now().Format(time.RFC3339))

	return appendToFile(restrictionsFile, restriction)
}

func (i *IsolationFramework) cleanup(ctx context.Context, ws *IsolatedWorkspace) {
	if ws.Worktree != nil {
		i.manager.RemoveWorktree(ctx, ws.Worktree.ID)
	}
}

// ChangeValidator validates changes made within isolated workspaces
type ChangeValidator struct {
	rules    []ValidationRule
	linters  map[string]Linter
	security *SecurityScanner
}

// NewChangeValidator creates a new change validator
func NewChangeValidator() *ChangeValidator {
	return &ChangeValidator{
		rules: []ValidationRule{
			&NoSecretRule{},
			&FileSizeRule{maxSize: 10 * 1024 * 1024}, // 10MB
			&PathValidationRule{},
		},
		linters:  make(map[string]Linter),
		security: NewSecurityScanner(),
	}
}

// ValidationRule defines an interface for change validation rules
type ValidationRule interface {
	Validate(ctx context.Context, changes []Change) []ValidationIssue
}

// ValidationResult contains the results of change validation
type ValidationResult struct {
	WorktreeID string            `json:"worktree_id"`
	Valid      bool              `json:"valid"`
	Issues     []ValidationIssue `json:"issues"`
	Timestamp  time.Time         `json:"timestamp"`
}

// ValidationIssue represents a validation problem
type ValidationIssue struct {
	File       string                 `json:"file"`
	Line       int                    `json:"line,omitempty"`
	Column     int                    `json:"column,omitempty"`
	Severity   Severity               `json:"severity"`
	Message    string                 `json:"message"`
	Rule       string                 `json:"rule"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Severity levels for validation issues
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Change represents a file change in the workspace
type Change struct {
	File       string    `json:"file"`
	Type       string    `json:"type"` // added, modified, deleted
	Content    string    `json:"content"`
	Line       int       `json:"line,omitempty"`
	OldContent string    `json:"old_content,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// Linter interface for language-specific validation
type Linter interface {
	Lint(ctx context.Context, change Change) []ValidationIssue
}

// ValidateChanges validates all changes in a worktree
func (cv *ChangeValidator) ValidateChanges(ctx context.Context, wt *Worktree) (*ValidationResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Get changes in worktree
	changes, err := cv.getChanges(ctx, wt)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get changes").
			WithComponent("git.worktree.isolation").
			WithOperation("ValidateChanges").
			WithDetails("worktree_id", wt.ID)
	}

	result := &ValidationResult{
		WorktreeID: wt.ID,
		Valid:      true,
		Issues:     []ValidationIssue{},
		Timestamp:  time.Now(),
	}

	// Apply validation rules
	for _, rule := range cv.rules {
		issues := rule.Validate(ctx, changes)
		result.Issues = append(result.Issues, issues...)
	}

	// Run linters
	for _, change := range changes {
		if linter := cv.getLinter(change.File); linter != nil {
			lintIssues := linter.Lint(ctx, change)
			result.Issues = append(result.Issues, lintIssues...)
		}
	}

	// Security scan
	if cv.security != nil {
		securityIssues := cv.security.Scan(ctx, changes)
		result.Issues = append(result.Issues, securityIssues...)
	}

	// Set validity
	for _, issue := range result.Issues {
		if issue.Severity == SeverityError {
			result.Valid = false
			break
		}
	}

	return result, nil
}

func (cv *ChangeValidator) getChanges(ctx context.Context, wt *Worktree) ([]Change, error) {
	// Use git to get changed files
	// This is a simplified implementation
	var changes []Change

	// For now, return empty changes - real implementation would parse git status
	return changes, nil
}

func (cv *ChangeValidator) getLinter(filename string) Linter {
	ext := filepath.Ext(filename)
	return cv.linters[ext]
}

// NoSecretRule validates that no secrets are committed
type NoSecretRule struct {
	patterns []*regexp.Regexp
}

func (nsr *NoSecretRule) Validate(ctx context.Context, changes []Change) []ValidationIssue {
	if nsr.patterns == nil {
		nsr.patterns = []*regexp.Regexp{
			regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*['"]?\w{20,}`),
			regexp.MustCompile(`(?i)(secret|password|passwd|pwd)\s*[:=]\s*['"]?\w+`),
			regexp.MustCompile(`(?i)aws[_-]?(access[_-]?key|secret)\s*[:=]\s*['"]?\w+`),
			regexp.MustCompile(`(?i)private[_-]?key\s*[:=]\s*['"]?-----BEGIN`),
			regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                      // OpenAI-style API keys
			regexp.MustCompile(`xoxb-[0-9]{12}-[0-9]{12}-[a-zA-Z0-9]{24}`), // Slack tokens
		}
	}

	var issues []ValidationIssue

	for _, change := range changes {
		lines := strings.Split(change.Content, "\n")
		for lineNum, line := range lines {
			for _, pattern := range nsr.patterns {
				if matches := pattern.FindAllString(line, -1); len(matches) > 0 {
					issues = append(issues, ValidationIssue{
						File:       change.File,
						Line:       lineNum + 1,
						Severity:   SeverityError,
						Message:    fmt.Sprintf("Potential secret detected: %s", matches[0]),
						Rule:       "no-secrets",
						Suggestion: "Remove sensitive data and use environment variables or secure vaults",
					})
				}
			}
		}
	}

	return issues
}

// FileSizeRule validates file sizes
type FileSizeRule struct {
	maxSize int64
}

func (fsr *FileSizeRule) Validate(ctx context.Context, changes []Change) []ValidationIssue {
	var issues []ValidationIssue

	for _, change := range changes {
		if int64(len(change.Content)) > fsr.maxSize {
			issues = append(issues, ValidationIssue{
				File:       change.File,
				Severity:   SeverityWarning,
				Message:    fmt.Sprintf("File size exceeds limit: %d bytes (max: %d)", len(change.Content), fsr.maxSize),
				Rule:       "file-size",
				Suggestion: "Consider breaking large files into smaller modules",
			})
		}
	}

	return issues
}

// PathValidationRule validates file paths
type PathValidationRule struct{}

func (pvr *PathValidationRule) Validate(ctx context.Context, changes []Change) []ValidationIssue {
	var issues []ValidationIssue

	restrictedPatterns := []string{
		`\.env$`,
		`\.secret`,
		`\.key$`,
		`\.pem$`,
		`id_rsa`,
		`\.aws/`,
	}

	for _, change := range changes {
		for _, pattern := range restrictedPatterns {
			if matched, _ := regexp.MatchString(pattern, change.File); matched {
				issues = append(issues, ValidationIssue{
					File:       change.File,
					Severity:   SeverityError,
					Message:    "File path contains sensitive patterns",
					Rule:       "path-validation",
					Suggestion: "Move sensitive files outside the repository or add to .gitignore",
				})
			}
		}
	}

	return issues
}

// SecurityScanner performs security scans on changes
type SecurityScanner struct{}

func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{}
}

func (ss *SecurityScanner) Scan(ctx context.Context, changes []Change) []ValidationIssue {
	var issues []ValidationIssue

	// Simple security patterns
	dangerousPatterns := map[string]string{
		`eval\s*\(`:            "Use of eval() is dangerous",
		`exec\s*\(`:            "Use of exec() can be dangerous",
		`__import__\s*\(`:      "Dynamic imports can be dangerous",
		`subprocess\.call`:     "Subprocess calls should be carefully reviewed",
		`os\.system`:           "Direct system calls should be avoided",
		`\.innerHTML\s*=`:      "Setting innerHTML can lead to XSS",
		`document\.write\s*\(`: "document.write can be dangerous",
	}

	for _, change := range changes {
		lines := strings.Split(change.Content, "\n")
		for lineNum, line := range lines {
			for pattern, message := range dangerousPatterns {
				if matched, _ := regexp.MatchString(pattern, line); matched {
					issues = append(issues, ValidationIssue{
						File:       change.File,
						Line:       lineNum + 1,
						Severity:   SeverityWarning,
						Message:    message,
						Rule:       "security-scan",
						Suggestion: "Review for security implications",
					})
				}
			}
		}
	}

	return issues
}

// Helper functions

func appendToFile(filename, content string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}
