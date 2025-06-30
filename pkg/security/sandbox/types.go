// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package sandbox

import (
	"context"
	"os/exec"
	"time"
)

// Command represents a command to be executed
type Command struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
	Dir  string   `json:"dir"`
	Env  []string `json:"env"`
}

// String returns a string representation of the command
func (c Command) String() string {
	if len(c.Args) == 0 {
		return c.Name
	}
	return c.Name + " " + c.Args[0]
}

// IsolatedCommand represents a command prepared for secure execution
type IsolatedCommand struct {
	Original       Command           `json:"original"`
	Namespace      string            `json:"namespace"`
	Chroot         string            `json:"chroot"`
	Capabilities   []string          `json:"capabilities"`
	Environment    map[string]string `json:"environment"`
	WorkingDir     string            `json:"working_dir"`
	ResourceLimits LimitConfig       `json:"resource_limits"`
}

// LimitConfig defines resource limits for command execution
type LimitConfig struct {
	MaxCPU       time.Duration `json:"max_cpu"`        // CPU time limit
	MaxMemory    int64         `json:"max_memory"`     // Memory limit in bytes
	MaxDisk      int64         `json:"max_disk"`       // Disk usage limit in bytes
	MaxProcesses int           `json:"max_processes"`  // Process count limit
	MaxOpenFiles int           `json:"max_open_files"` // File descriptor limit
	Timeout      time.Duration `json:"timeout"`        // Execution timeout
	MaxNetworkIO int64         `json:"max_network_io"` // Network I/O limit in bytes
}

// DefaultLimitConfig returns sensible default resource limits
func DefaultLimitConfig() LimitConfig {
	return LimitConfig{
		MaxCPU:       30 * time.Second,       // 30 seconds of CPU time
		MaxMemory:    1024 * 1024 * 1024,     // 1GB memory
		MaxDisk:      5 * 1024 * 1024 * 1024, // 5GB disk
		MaxProcesses: 100,                    // 100 processes
		MaxOpenFiles: 1000,                   // 1000 file descriptors
		Timeout:      10 * time.Minute,       // 10 minute total timeout
		MaxNetworkIO: 100 * 1024 * 1024,      // 100MB network I/O
	}
}

// SandboxConfig defines the configuration for a sandbox environment
type SandboxConfig struct {
	ProjectRoot       string            `json:"project_root"`
	AllowedReadPaths  []string          `json:"allowed_read_paths"`
	AllowedWritePaths []string          `json:"allowed_write_paths"`
	ForbiddenPaths    []string          `json:"forbidden_paths"`
	AllowedHosts      []string          `json:"allowed_hosts"`
	TempDirPattern    string            `json:"temp_dir_pattern"`
	ResourceLimits    LimitConfig       `json:"resource_limits"`
	Environment       map[string]string `json:"environment"`
	EnableNetworking  bool              `json:"enable_networking"`
	EnableFilesystem  bool              `json:"enable_filesystem"`
}

// DefaultSandboxConfig returns a default sandbox configuration
func DefaultSandboxConfig(projectRoot string) SandboxConfig {
	return SandboxConfig{
		ProjectRoot: projectRoot,
		AllowedReadPaths: []string{
			"/usr/lib",
			"/usr/share",
			"/etc/ssl/certs",
			"/etc/ca-certificates",
		},
		AllowedWritePaths: []string{
			projectRoot + "/**",
		},
		ForbiddenPaths: []string{
			"/home/*/.ssh",
			"/home/*/.aws",
			"/root/**",
			"/etc/passwd",
			"/etc/shadow",
			"/etc/sudoers",
		},
		AllowedHosts: []string{
			"api.github.com",
			"registry.npmjs.org",
			"pypi.org",
			"proxy.golang.org",
			"localhost",
			"127.0.0.1",
		},
		TempDirPattern:   "/tmp/guild-{agent_id}",
		ResourceLimits:   DefaultLimitConfig(),
		Environment:      make(map[string]string),
		EnableNetworking: true,
		EnableFilesystem: true,
	}
}

// SecurityEvent represents a security-related event during execution
type SecurityEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	Command   Command                `json:"command"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Blocked   bool                   `json:"blocked"`
	Reason    string                 `json:"reason"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID           string        `json:"id"`
	Rule         string        `json:"rule"`
	Severity     AlertSeverity `json:"severity"`
	Message      string        `json:"message"`
	Event        SecurityEvent `json:"event"`
	Timestamp    time.Time     `json:"timestamp"`
	Acknowledged bool          `json:"acknowledged"`
}

// AlertSeverity defines the severity levels for security alerts
type AlertSeverity int

const (
	SeverityLow AlertSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s AlertSeverity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Interfaces

// Isolator defines the interface for command isolation
type Isolator interface {
	// Isolate prepares a command for secure execution
	Isolate(ctx context.Context, cmd Command) (IsolatedCommand, error)

	// ValidatePath checks if a path is allowed for the given operation
	ValidatePath(path string, operation PathOperation) error

	// GetTempDir returns the temporary directory for an agent
	GetTempDir(agentID string) string

	// Close releases resources used by the isolator
	Close() error
}

// PathOperation defines the type of path operation
type PathOperation int

const (
	PathOperationRead PathOperation = iota
	PathOperationWrite
	PathOperationExecute
	PathOperationDelete
)

func (po PathOperation) String() string {
	switch po {
	case PathOperationRead:
		return "read"
	case PathOperationWrite:
		return "write"
	case PathOperationExecute:
		return "execute"
	case PathOperationDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// ResourceLimiter defines the interface for applying resource limits
type ResourceLimiter interface {
	// ApplyLimits applies resource limits to a command
	ApplyLimits(ctx context.Context, cmd *exec.Cmd, limits LimitConfig) error

	// MonitorExecution monitors resource usage during execution
	MonitorExecution(ctx context.Context, cmd *exec.Cmd) (*ResourceUsage, error)

	// GetUsage returns current resource usage
	GetUsage(ctx context.Context) (*ResourceUsage, error)
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPUTime        time.Duration `json:"cpu_time"`
	MemoryBytes    int64         `json:"memory_bytes"`
	DiskBytes      int64         `json:"disk_bytes"`
	ProcessCount   int           `json:"process_count"`
	OpenFiles      int           `json:"open_files"`
	NetworkIOBytes int64         `json:"network_io_bytes"`
	Timestamp      time.Time     `json:"timestamp"`
}

// SecurityMonitor defines the interface for security monitoring
type SecurityMonitor interface {
	// StartMonitoring begins monitoring for security events
	StartMonitoring(ctx context.Context) error

	// StopMonitoring stops security monitoring
	StopMonitoring() error

	// AddRule adds a security monitoring rule
	AddRule(rule SecurityRule) error

	// RemoveRule removes a security monitoring rule
	RemoveRule(ruleID string) error

	// GetAlerts retrieves recent security alerts
	GetAlerts(ctx context.Context, filter AlertFilter) ([]SecurityAlert, error)

	// MonitorCommand monitors a specific command execution
	MonitorCommand(ctx context.Context, cmd Command) error
}

// SecurityRule defines a rule for security monitoring
type SecurityRule interface {
	// ID returns the rule identifier
	ID() string

	// Name returns the rule name
	Name() string

	// Evaluate checks if the rule triggers on an event
	Evaluate(ctx context.Context, event SecurityEvent) (*SecurityAlert, error)

	// IsEnabled returns whether the rule is active
	IsEnabled() bool

	// SetEnabled enables or disables the rule
	SetEnabled(enabled bool)
}

// AlertFilter defines criteria for querying security alerts
type AlertFilter struct {
	StartTime *time.Time    `json:"start_time,omitempty"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Severity  AlertSeverity `json:"severity,omitempty"`
	Rule      string        `json:"rule,omitempty"`
	AgentID   string        `json:"agent_id,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Offset    int           `json:"offset,omitempty"`
}

// NetworkFilter defines network access restrictions
type NetworkFilter interface {
	// IsHostAllowed checks if a host is allowed for network access
	IsHostAllowed(host string) bool

	// FilterRequest filters an outgoing network request
	FilterRequest(ctx context.Context, req NetworkRequest) error

	// GetAllowedHosts returns the list of allowed hosts
	GetAllowedHosts() []string
}

// NetworkRequest represents an outgoing network request
type NetworkRequest struct {
	Host    string            `json:"host"`
	Port    int               `json:"port"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// Sandbox defines the main interface for the sandboxing system
type Sandbox interface {
	// Execute runs a command in the sandboxed environment
	Execute(ctx context.Context, cmd Command) (*ExecutionResult, error)

	// ValidateCommand checks if a command is safe to execute
	ValidateCommand(ctx context.Context, cmd Command) error

	// GetConfig returns the current sandbox configuration
	GetConfig() SandboxConfig

	// UpdateConfig updates the sandbox configuration
	UpdateConfig(config SandboxConfig) error

	// GetStats returns sandbox usage statistics
	GetStats(ctx context.Context) (*SandboxStats, error)

	// Close releases sandbox resources
	Close() error
}

// ExecutionResult represents the result of command execution
type ExecutionResult struct {
	ExitCode       int             `json:"exit_code"`
	Stdout         string          `json:"stdout"`
	Stderr         string          `json:"stderr"`
	Duration       time.Duration   `json:"duration"`
	ResourceUsage  *ResourceUsage  `json:"resource_usage"`
	SecurityEvents []SecurityEvent `json:"security_events"`
	Error          error           `json:"error,omitempty"`
}

// SandboxStats provides statistics about sandbox usage
type SandboxStats struct {
	TotalExecutions    int64         `json:"total_executions"`
	BlockedCommands    int64         `json:"blocked_commands"`
	SecurityViolations int64         `json:"security_violations"`
	AverageExecution   time.Duration `json:"average_execution"`
	ResourceUsage      ResourceUsage `json:"resource_usage"`
	LastActivity       time.Time     `json:"last_activity"`
}
