// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package interfaces

// CostProfile represents the cost characteristics of an agent
// Moved here to avoid circular dependencies between the agent and registry packages.
type CostProfile struct {
	Magnitude     int    `yaml:"magnitude" json:"magnitude"`
	ContextWindow int    `yaml:"context_window" json:"context_window"`
	ContextReset  string `yaml:"context_reset" json:"context_reset"`
	Available     bool   `yaml:"available" json:"available"`
}

// AgentInfo holds agent information for registry operations
// Placed in interfaces to prevent import cycles.
type AgentInfo struct {
	ID            string
	Type          string
	Name          string
	Capabilities  []string
	CostProfile   CostProfile
	CostMagnitude int // For backward compatibility
}

// GuildAgentConfig represents a configured agent from guild config
// Shared between the agent and registry packages.
type GuildAgentConfig struct {
	ID            string   `yaml:"id"`
	Name          string   `yaml:"name"`
	Type          string   `yaml:"type"`
	Model         string   `yaml:"model"`
	Provider      string   `yaml:"provider"`
	SystemPrompt  string   `yaml:"system_prompt"`
	Tools         []string `yaml:"tools"`
	Capabilities  []string `yaml:"capabilities"`
	CostMagnitude int      `yaml:"cost_magnitude,omitempty"`
	ContextWindow int      `yaml:"context_window,omitempty"`
}
