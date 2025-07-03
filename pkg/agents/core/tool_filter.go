// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"slices"
	"strings"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/tools"
)

// ToolFilter provides tool access control for agents based on their configuration
type ToolFilter struct {
	config   config.ToolAccessConfig
	registry tools.Registry
	agentID  string
}

// NewToolFilter creates a new tool filter with the given configuration
func NewToolFilter(config config.ToolAccessConfig, registry tools.Registry, agentID string) *ToolFilter {
	return &ToolFilter{
		config:   config,
		registry: registry,
		agentID:  agentID,
	}
}

// CanUseTool checks if the agent can use the specified tool
func (tf *ToolFilter) CanUseTool(ctx context.Context, toolName string) bool {
	if err := ctx.Err(); err != nil {
		return false
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ToolFilter")
	ctx = observability.WithOperation(ctx, "CanUseTool")

	logger.DebugContext(ctx, "Checking tool access",
		"agent_id", tf.agentID,
		"tool_name", toolName,
		"allow_all", tf.config.AllowAll)

	// First check if the tool is explicitly blocked
	for _, blocked := range tf.config.Blocked {
		if strings.EqualFold(blocked, toolName) || blocked == "*" {
			logger.DebugContext(ctx, "Tool access denied - explicitly blocked",
				"agent_id", tf.agentID,
				"tool_name", toolName)
			return false
		}
	}

	// If allow_all is true, permit unless blocked (already checked above)
	if tf.config.AllowAll {
		logger.DebugContext(ctx, "Tool access granted - allow_all enabled",
			"agent_id", tf.agentID,
			"tool_name", toolName)
		return true
	}

	// Check allowed list
	for _, allowed := range tf.config.Allowed {
		if strings.EqualFold(allowed, toolName) || allowed == "*" {
			logger.DebugContext(ctx, "Tool access granted - explicitly allowed",
				"agent_id", tf.agentID,
				"tool_name", toolName)
			return true
		}
	}

	logger.DebugContext(ctx, "Tool access denied - not in allowed list",
		"agent_id", tf.agentID,
		"tool_name", toolName)
	return false
}

// FilterTools filters a list of available tools based on access control
func (tf *ToolFilter) FilterTools(ctx context.Context, available []string) []string {
	if err := ctx.Err(); err != nil {
		return []string{}
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ToolFilter")
	ctx = observability.WithOperation(ctx, "FilterTools")

	logger.DebugContext(ctx, "Filtering tools",
		"agent_id", tf.agentID,
		"available_count", len(available))

	filtered := make([]string, 0, len(available))
	blocked_count := 0

	for _, tool := range available {
		// Check context in loop
		if err := ctx.Err(); err != nil {
			break
		}

		if tf.CanUseTool(ctx, tool) {
			filtered = append(filtered, tool)
		} else {
			blocked_count++
		}
	}

	logger.InfoContext(ctx, "Tool filtering completed",
		"agent_id", tf.agentID,
		"available_count", len(available),
		"allowed_count", len(filtered),
		"blocked_count", blocked_count)

	return filtered
}

// GetAllowedTools returns all tools that the agent is allowed to use from the registry
func (tf *ToolFilter) GetAllowedTools(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ToolFilter").
			WithOperation("GetAllowedTools")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ToolFilter")
	ctx = observability.WithOperation(ctx, "GetAllowedTools")

	logger.DebugContext(ctx, "Getting allowed tools from registry", "agent_id", tf.agentID)

	// Get all available tools from registry
	if tf.registry == nil {
		return []string{}, gerror.New(gerror.ErrCodeInvalidInput, "tool registry is nil", nil).
			WithComponent("ToolFilter").
			WithOperation("GetAllowedTools").
			WithDetails("agent_id", tf.agentID)
	}

	// Note: The tools.Registry interface might not have a GetAllTools method
	// For now, we'll work with the tools specified in the config
	var availableTools []string

	if tf.config.AllowAll {
		// If allow_all is true, we need to get tools from registry
		// Since the registry interface doesn't expose all tools, we'll use the allowed list as a fallback
		availableTools = tf.config.Allowed
		if len(availableTools) == 0 {
			// Common tools that are typically available
			availableTools = []string{
				"file_tool", "grep_tool", "glob_tool", "shell_tool",
				"git_tool", "http_tool", "web_search", "web_fetch",
			}
		}
	} else {
		availableTools = tf.config.Allowed
	}

	filtered := tf.FilterTools(ctx, availableTools)

	logger.InfoContext(ctx, "Retrieved allowed tools",
		"agent_id", tf.agentID,
		"allowed_count", len(filtered))

	return filtered, nil
}

// ValidateToolAccess validates that a tool access request is allowed
func (tf *ToolFilter) ValidateToolAccess(ctx context.Context, toolName string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ToolFilter").
			WithOperation("ValidateToolAccess")
	}

	if !tf.CanUseTool(ctx, toolName) {
		return gerror.Newf(gerror.ErrCodeValidation, "agent '%s' is not allowed to use tool '%s'", tf.agentID, toolName).
			WithComponent("ToolFilter").
			WithOperation("ValidateToolAccess").
			WithDetails("agent_id", tf.agentID).
			WithDetails("tool_name", toolName)
	}

	return nil
}

// GetBlockedTools returns a list of explicitly blocked tools
func (tf *ToolFilter) GetBlockedTools() []string {
	// Return a copy to prevent modification
	blocked := make([]string, len(tf.config.Blocked))
	copy(blocked, tf.config.Blocked)
	return blocked
}

// IsToolBlocked checks if a specific tool is explicitly blocked
func (tf *ToolFilter) IsToolBlocked(toolName string) bool {
	return slices.Contains(tf.config.Blocked, toolName) || slices.Contains(tf.config.Blocked, "*")
}

// IsAllowAllEnabled returns true if the agent has access to all tools (unless explicitly blocked)
func (tf *ToolFilter) IsAllowAllEnabled() bool {
	return tf.config.AllowAll
}

// GetToolAccessSummary returns a summary of the tool access configuration
func (tf *ToolFilter) GetToolAccessSummary(ctx context.Context) map[string]interface{} {
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ToolFilter")
	ctx = observability.WithOperation(ctx, "GetToolAccessSummary")

	logger.DebugContext(ctx, "Generating tool access summary", "agent_id", tf.agentID)

	summary := map[string]interface{}{
		"agent_id":     tf.agentID,
		"allow_all":    tf.config.AllowAll,
		"allowed":      tf.config.Allowed,
		"blocked":      tf.config.Blocked,
		"has_registry": tf.registry != nil,
	}

	// Add counts
	summary["allowed_count"] = len(tf.config.Allowed)
	summary["blocked_count"] = len(tf.config.Blocked)

	return summary
}

// UpdateToolAccess updates the tool access configuration
func (tf *ToolFilter) UpdateToolAccess(ctx context.Context, newConfig config.ToolAccessConfig) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ToolFilter").
			WithOperation("UpdateToolAccess")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ToolFilter")
	ctx = observability.WithOperation(ctx, "UpdateToolAccess")

	logger.InfoContext(ctx, "Updating tool access configuration", "agent_id", tf.agentID)

	// Validate the new configuration
	if err := newConfig.Validate(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid tool access configuration").
			WithComponent("ToolFilter").
			WithOperation("UpdateToolAccess").
			WithDetails("agent_id", tf.agentID)
	}

	// Update the configuration
	oldConfig := tf.config
	tf.config = newConfig

	logger.InfoContext(ctx, "Tool access configuration updated",
		"agent_id", tf.agentID,
		"old_allow_all", oldConfig.AllowAll,
		"new_allow_all", newConfig.AllowAll,
		"old_allowed_count", len(oldConfig.Allowed),
		"new_allowed_count", len(newConfig.Allowed),
		"old_blocked_count", len(oldConfig.Blocked),
		"new_blocked_count", len(newConfig.Blocked))

	return nil
}

// ToolFilterFactory creates tool filters from agent configurations
type ToolFilterFactory struct {
	registry tools.Registry
}

// NewToolFilterFactory creates a new tool filter factory
func NewToolFilterFactory(registry tools.Registry) *ToolFilterFactory {
	return &ToolFilterFactory{
		registry: registry,
	}
}

// CreateToolFilter creates a tool filter for an agent configuration
func (tff *ToolFilterFactory) CreateToolFilter(config *config.EnhancedAgentConfig) *ToolFilter {
	return NewToolFilter(config.Tools, tff.registry, config.ID)
}

// CreateToolFilterFromBase creates a tool filter from a base agent configuration
func (tff *ToolFilterFactory) CreateToolFilterFromBase(baseConfig config.AgentConfig) *ToolFilter {
	// Convert base config to enhanced config first
	enhanced := config.FromBaseAgentConfig(baseConfig)
	return tff.CreateToolFilter(enhanced)
}
