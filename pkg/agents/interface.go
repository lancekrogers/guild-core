// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agents

import (
	"context"

	"github.com/lancekrogers/guild/pkg/agents/backstory"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
)

// AgentCreator defines the interface for creating enhanced agents
type AgentCreator interface {
	// CreateElenaGuildMaster creates Elena the Guild Master
	CreateElenaGuildMaster(ctx context.Context) (*config.AgentConfig, error)

	// CreateDefaultDeveloper creates an enhanced developer agent
	CreateDefaultDeveloper(ctx context.Context) (*config.AgentConfig, error)

	// CreateDefaultTester creates an enhanced tester agent
	CreateDefaultTester(ctx context.Context) (*config.AgentConfig, error)

	// CreateDefaultAgentSet creates a complete set of default agents
	CreateDefaultAgentSet(ctx context.Context) ([]*config.AgentConfig, error)

	// GetOptimalProvider determines the best provider for an agent
	GetOptimalProvider(agentType, agentID string) string

	// GetSpecialistTemplate returns a specialist template by ID
	GetSpecialistTemplate(specialistID string) (*config.AgentConfig, error)

	// ListAvailableSpecialists returns all available specialist templates
	ListAvailableSpecialists() []string

	// EnhanceAgentWithBackstory enhances an agent with specialist backstory
	EnhanceAgentWithBackstory(ctx context.Context, agent *config.AgentConfig, backstoryID string) error
}

// Initializer defines the interface for agent initialization and management
type Initializer interface {
	// InitializeDefaultAgents creates and saves default agents to a project
	InitializeDefaultAgents(ctx context.Context, projectPath string) error

	// LoadAndEnhanceAgents loads agents from config and enhances them
	LoadAndEnhanceAgents(ctx context.Context, guildConfig *config.GuildConfig) error

	// CreateElenaIfMissing checks if Elena exists and creates her if missing
	CreateElenaIfMissing(ctx context.Context, guildConfig *config.GuildConfig, projectPath string) error

	// EnhanceExistingAgent enhances an existing agent with a specialist template
	EnhanceExistingAgent(ctx context.Context, agentID, specialistTemplate string, guildConfig *config.GuildConfig, projectPath string) error

	// GetBackstoryManager returns the backstory manager for external use
	GetBackstoryManager() *backstory.BackstoryManager

	// GetAvailableSpecialists returns list of available specialist templates
	GetAvailableSpecialists() []string

	// GeneratePersonalityPrompt generates an enhanced prompt using the backstory system
	GeneratePersonalityPrompt(ctx context.Context, agentID, basePrompt string, turnContext *layered.TurnContext) (string, error)

	// CreateGuildConfigWithElena creates a complete guild config with Elena as manager
	CreateGuildConfigWithElena(ctx context.Context, guildName string) (*config.GuildConfig, error)

	// UpgradeExistingGuild upgrades an existing guild with enhanced agents
	UpgradeExistingGuild(ctx context.Context, guildConfig *config.GuildConfig, projectPath string) error
}

// EnhancedAgentManager combines creation and initialization capabilities
type EnhancedAgentManager interface {
	AgentCreator
	Initializer
}
