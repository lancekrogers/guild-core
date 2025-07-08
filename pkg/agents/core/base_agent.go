// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

// BaseAgentConfig holds basic agent configuration
type BaseAgentConfig struct {
	ID           string
	Name         string
	SystemPrompt string
	Model        string
	MaxTokens    int
	Temperature  float64
}

// BaseAgent provides a base implementation with common fields
type BaseAgent struct {
	config BaseAgentConfig
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(config BaseAgentConfig) *BaseAgent {
	return &BaseAgent{
		config: config,
	}
}

// GetID returns the agent's ID
func (a *BaseAgent) GetID() string {
	return a.config.ID
}

// GetName returns the agent's name
func (a *BaseAgent) GetName() string {
	return a.config.Name
}

// GetConfig returns the agent's configuration
func (a *BaseAgent) GetConfig() BaseAgentConfig {
	return a.config
}
