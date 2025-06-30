// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package permissions

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// SafeCommandCondition ensures shell commands are safe to execute
type SafeCommandCondition struct {
	DangerousCommands []string `json:"dangerous_commands"`
	AllowedCommands   []string `json:"allowed_commands,omitempty"` // If specified, only these commands are allowed
}

// NewSafeCommandCondition creates a condition with default dangerous commands
func NewSafeCommandCondition() *SafeCommandCondition {
	return &SafeCommandCondition{
		DangerousCommands: []string{
			"rm -rf",
			"format",
			"dd",
			"chmod 777",
			"curl | sh",
			"wget | sh",
			"sudo",
			"su",
			"passwd",
			"useradd",
			"userdel",
			"shutdown",
			"reboot",
			"halt",
			"init",
			"poweroff",
			"eval",
			"exec",
			"nc -l",
			"python -c",
			"perl -e",
			"ruby -e",
			"node -e",
			"bash -c",
			"sh -c",
			"> /dev/",
			"mkfs",
			"fdisk",
			"mount",
			"umount",
		},
	}
}

// Evaluate checks if the shell command is safe to execute
func (scc *SafeCommandCondition) Evaluate(ctx EvaluationContext) bool {
	if !strings.HasPrefix(ctx.Resource, "shell:") {
		return true // Not a shell command, condition doesn't apply
	}

	command := strings.TrimPrefix(ctx.Resource, "shell:")
	command = strings.ToLower(strings.TrimSpace(command))

	// If allowlist is specified, only allow commands in the list
	if len(scc.AllowedCommands) > 0 {
		for _, allowed := range scc.AllowedCommands {
			if strings.HasPrefix(command, strings.ToLower(allowed)) {
				return true
			}
		}
		return false
	}

	// Check against dangerous commands
	for _, dangerous := range scc.DangerousCommands {
		if strings.Contains(command, strings.ToLower(dangerous)) {
			return false
		}
	}

	return true
}

// Name returns the condition name
func (scc *SafeCommandCondition) Name() string {
	return "safe_command"
}

// Description returns the condition description
func (scc *SafeCommandCondition) Description() string {
	return "Ensures shell commands do not contain dangerous operations"
}

// TimeWindowCondition restricts access to specific time periods
type TimeWindowCondition struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewTimeWindowCondition creates a time-based condition
func NewTimeWindowCondition(start, end time.Time) *TimeWindowCondition {
	return &TimeWindowCondition{
		Start: start,
		End:   end,
	}
}

// Evaluate checks if the current time is within the allowed window
func (twc *TimeWindowCondition) Evaluate(ctx EvaluationContext) bool {
	return ctx.Time.After(twc.Start) && ctx.Time.Before(twc.End)
}

// Name returns the condition name
func (twc *TimeWindowCondition) Name() string {
	return "time_window"
}

// Description returns the condition description
func (twc *TimeWindowCondition) Description() string {
	return fmt.Sprintf("Access allowed between %s and %s",
		twc.Start.Format(time.RFC3339), twc.End.Format(time.RFC3339))
}

// IPRangeCondition restricts access based on client IP address
type IPRangeCondition struct {
	AllowedRanges []string `json:"allowed_ranges"` // CIDR notation
	DeniedRanges  []string `json:"denied_ranges"`  // CIDR notation
}

// NewIPRangeCondition creates an IP-based condition
func NewIPRangeCondition(allowedRanges, deniedRanges []string) *IPRangeCondition {
	return &IPRangeCondition{
		AllowedRanges: allowedRanges,
		DeniedRanges:  deniedRanges,
	}
}

// Evaluate checks if the client IP is within allowed ranges
func (irc *IPRangeCondition) Evaluate(ctx EvaluationContext) bool {
	if ctx.IPAddress == "" {
		return true // No IP info available, allow
	}

	clientIP := net.ParseIP(ctx.IPAddress)
	if clientIP == nil {
		return false // Invalid IP format
	}

	// Check denied ranges first (explicit deny takes precedence)
	for _, deniedRange := range irc.DeniedRanges {
		if irc.ipInRange(clientIP, deniedRange) {
			return false
		}
	}

	// If no allowed ranges specified, allow all (except denied)
	if len(irc.AllowedRanges) == 0 {
		return true
	}

	// Check if IP is in any allowed range
	for _, allowedRange := range irc.AllowedRanges {
		if irc.ipInRange(clientIP, allowedRange) {
			return true
		}
	}

	return false
}

// ipInRange checks if an IP is within a CIDR range
func (irc *IPRangeCondition) ipInRange(ip net.IP, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}

// Name returns the condition name
func (irc *IPRangeCondition) Name() string {
	return "ip_range"
}

// Description returns the condition description
func (irc *IPRangeCondition) Description() string {
	return fmt.Sprintf("Access restricted to IP ranges: allowed=%v, denied=%v",
		irc.AllowedRanges, irc.DeniedRanges)
}

// FilePathCondition restricts access to specific file paths
type FilePathCondition struct {
	AllowedPaths []string `json:"allowed_paths"` // Glob patterns
	DeniedPaths  []string `json:"denied_paths"`  // Glob patterns
}

// NewFilePathCondition creates a file path-based condition
func NewFilePathCondition(allowedPaths, deniedPaths []string) *FilePathCondition {
	return &FilePathCondition{
		AllowedPaths: allowedPaths,
		DeniedPaths:  deniedPaths,
	}
}

// Evaluate checks if the file path is allowed
func (fpc *FilePathCondition) Evaluate(ctx EvaluationContext) bool {
	if !strings.HasPrefix(ctx.Resource, "file:") {
		return true // Not a file operation, condition doesn't apply
	}

	filePath := strings.TrimPrefix(ctx.Resource, "file:")

	// Check denied paths first
	for _, deniedPattern := range fpc.DeniedPaths {
		if fpc.matchesPattern(filePath, deniedPattern) {
			return false
		}
	}

	// If no allowed paths specified, allow all (except denied)
	if len(fpc.AllowedPaths) == 0 {
		return true
	}

	// Check if path matches any allowed pattern
	for _, allowedPattern := range fpc.AllowedPaths {
		if fpc.matchesPattern(filePath, allowedPattern) {
			return true
		}
	}

	return false
}

// matchesPattern checks if a path matches a glob pattern
func (fpc *FilePathCondition) matchesPattern(path, pattern string) bool {
	// Simple glob matching - could be enhanced with filepath.Match for full glob support
	if pattern == "*" {
		return true
	}

	if strings.Contains(pattern, "*") {
		// Convert glob pattern to regex
		regexPattern := strings.ReplaceAll(pattern, "*", ".*")
		regexPattern = "^" + regexPattern + "$"

		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return false
		}

		return regex.MatchString(path)
	}

	return path == pattern
}

// Name returns the condition name
func (fpc *FilePathCondition) Name() string {
	return "file_path"
}

// Description returns the condition description
func (fpc *FilePathCondition) Description() string {
	return fmt.Sprintf("File access restricted to paths: allowed=%v, denied=%v",
		fpc.AllowedPaths, fpc.DeniedPaths)
}

// AgentRoleCondition restricts access based on agent's current roles
type AgentRoleCondition struct {
	RequiredRoles  []string `json:"required_roles"`  // Must have ALL of these roles
	ForbiddenRoles []string `json:"forbidden_roles"` // Must NOT have ANY of these roles
}

// NewAgentRoleCondition creates a role-based condition
func NewAgentRoleCondition(requiredRoles, forbiddenRoles []string) *AgentRoleCondition {
	return &AgentRoleCondition{
		RequiredRoles:  requiredRoles,
		ForbiddenRoles: forbiddenRoles,
	}
}

// Evaluate checks if the agent has the required roles (this would need role info in context)
func (arc *AgentRoleCondition) Evaluate(ctx EvaluationContext) bool {
	// This is a placeholder - in a real implementation, we'd need to look up
	// the agent's current roles from the permission model
	// For now, we'll use metadata to pass role information

	agentRoles, ok := ctx.Metadata["agent_roles"].([]string)
	if !ok {
		return len(arc.RequiredRoles) == 0 // If no role info, only allow if no roles required
	}

	// Check forbidden roles first
	for _, forbiddenRole := range arc.ForbiddenRoles {
		for _, agentRole := range agentRoles {
			if agentRole == forbiddenRole {
				return false
			}
		}
	}

	// Check required roles
	for _, requiredRole := range arc.RequiredRoles {
		found := false
		for _, agentRole := range agentRoles {
			if agentRole == requiredRole {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Name returns the condition name
func (arc *AgentRoleCondition) Name() string {
	return "agent_role"
}

// Description returns the condition description
func (arc *AgentRoleCondition) Description() string {
	return fmt.Sprintf("Agent must have required roles: %v, and not have forbidden roles: %v",
		arc.RequiredRoles, arc.ForbiddenRoles)
}

// ConditionFromJSON creates a condition from JSON data
func ConditionFromJSON(conditionType string, data []byte) (Condition, error) {
	switch conditionType {
	case "safe_command":
		var condition SafeCommandCondition
		if err := json.Unmarshal(data, &condition); err != nil {
			return nil, err
		}
		return &condition, nil

	case "time_window":
		var condition TimeWindowCondition
		if err := json.Unmarshal(data, &condition); err != nil {
			return nil, err
		}
		return &condition, nil

	case "ip_range":
		var condition IPRangeCondition
		if err := json.Unmarshal(data, &condition); err != nil {
			return nil, err
		}
		return &condition, nil

	case "file_path":
		var condition FilePathCondition
		if err := json.Unmarshal(data, &condition); err != nil {
			return nil, err
		}
		return &condition, nil

	case "agent_role":
		var condition AgentRoleCondition
		if err := json.Unmarshal(data, &condition); err != nil {
			return nil, err
		}
		return &condition, nil

	default:
		return nil, fmt.Errorf("unknown condition type: %s", conditionType)
	}
}
