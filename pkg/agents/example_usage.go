// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agents

import (
	"context"
	"fmt"
	"log"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
)

// ExampleCreateEnhancedGuild demonstrates creating a guild with Elena and rich agents
func ExampleCreateEnhancedGuild() {
	ctx := context.Background()

	// Create a mock prompt registry for this example
	promptRegistry := &mockPromptRegistry{}

	// Create agent initializer
	initializer := NewAgentInitializer(promptRegistry)

	// Create a new guild with Elena as the manager
	guildConfig, err := initializer.CreateGuildConfigWithElena(ctx, "example-guild")
	if err != nil {
		log.Fatalf("Failed to create guild: %v", err)
	}

	fmt.Printf("Created guild '%s' with %d agents\n", guildConfig.Name, len(guildConfig.Agents))
	fmt.Printf("Default manager: %s\n", guildConfig.Manager.Default)

	// Show agent details
	for _, agent := range guildConfig.Agents {
		fmt.Printf("\n--- Agent: %s ---\n", agent.Name)
		fmt.Printf("Type: %s\n", agent.Type)
		fmt.Printf("Provider: %s\n", agent.Provider)

		if agent.Backstory != nil {
			fmt.Printf("Guild Rank: %s\n", agent.Backstory.GuildRank)
			fmt.Printf("Experience: %s\n", agent.Backstory.Experience)
		}

		if agent.Personality != nil {
			fmt.Printf("Communication Style: %s\n", agent.Personality.Formality)
			fmt.Printf("Empathy Level: %d/10\n", agent.Personality.Empathy)
		}

		fmt.Printf("Capabilities: %v\n", agent.Capabilities)
	}
}

// ExampleUpgradeExistingGuild shows how to upgrade an existing simple guild
func ExampleUpgradeExistingGuild() {
	ctx := context.Background()
	projectPath := "/tmp/example-project"

	// Start with a simple guild config (like current defaults)
	simpleGuild := &config.GuildConfig{
		Name:        "simple-guild",
		Description: "Basic guild without personalities",
		Agents: []config.AgentConfig{
			{
				ID:           "manager",
				Name:         "manager",
				Type:         "manager",
				Description:  "Basic manager",
				Provider:     "anthropic",
				Model:        "claude-3-sonnet-20240229",
				Capabilities: []string{"task_breakdown", "agent_assignment"},
			},
			{
				ID:           "developer",
				Name:         "developer",
				Type:         "worker",
				Description:  "Basic developer",
				Provider:     "anthropic",
				Model:        "claude-3-sonnet-20240229",
				Capabilities: []string{"code_generation", "debugging"},
			},
		},
	}

	fmt.Printf("Before upgrade: %d agents, manager is '%s'\n",
		len(simpleGuild.Agents), simpleGuild.Manager.Default)

	// Create initializer
	promptRegistry := &mockPromptRegistry{}
	initializer := NewAgentInitializer(promptRegistry)

	// Upgrade the guild
	if err := initializer.UpgradeExistingGuild(ctx, simpleGuild, projectPath); err != nil {
		log.Fatalf("Failed to upgrade guild: %v", err)
	}

	fmt.Printf("After upgrade: %d agents, manager is '%s'\n",
		len(simpleGuild.Agents), simpleGuild.Manager.Default)

	// Find Elena
	for _, agent := range simpleGuild.Agents {
		if agent.ID == "elena-guild-master" {
			fmt.Printf("\nElena added successfully!\n")
			fmt.Printf("Name: %s\n", agent.Name)
			if agent.Backstory != nil {
				fmt.Printf("Philosophy: %s\n", agent.Backstory.Philosophy)
			}
			break
		}
	}
}

// ExamplePersonalityPromptGeneration shows how to use the backstory system
func ExamplePersonalityPromptGeneration() {
	ctx := context.Background()

	// Create initializer
	promptRegistry := &mockPromptRegistry{}
	initializer := NewAgentInitializer(promptRegistry)

	// Create Elena
	creator := NewDefaultAgentCreator()
	elena, err := creator.CreateElenaGuildMaster(ctx)
	if err != nil {
		log.Fatalf("Failed to create Elena: %v", err)
	}

	// Register Elena with backstory manager
	backstoryManager := initializer.GetBackstoryManager()
	if err := backstoryManager.RegisterAgent(elena); err != nil {
		log.Fatalf("Failed to register Elena: %v", err)
	}

	// Create a turn context
	turnContext := &layered.TurnContext{
		UserMessage: "I need help organizing this complex project with multiple teams",
		TaskID:      "project-organization",
		Urgency:     "high",
	}

	basePrompt := "Please help the user with their request."

	// Generate personality-enhanced prompt
	enhancedPrompt, err := initializer.GeneratePersonalityPrompt(
		ctx, elena.ID, basePrompt, turnContext)
	if err != nil {
		log.Fatalf("Failed to generate personality prompt: %v", err)
	}

	fmt.Printf("Base prompt: %s\n\n", basePrompt)
	fmt.Printf("Enhanced prompt:\n%s\n", enhancedPrompt)
}

// ExampleSpecialistEnhancement shows how to enhance agents with specialist templates
func ExampleSpecialistEnhancement() {
	ctx := context.Background()
	projectPath := "/tmp/example-project"

	// Start with basic guild
	guildConfig := &config.GuildConfig{
		Name: "specialist-guild",
		Agents: []config.AgentConfig{
			{
				ID:           "basic-agent",
				Name:         "Basic Agent",
				Type:         "specialist",
				Provider:     "anthropic",
				Model:        "claude-3-sonnet-20240229",
				Capabilities: []string{"general_tasks"},
			},
		},
	}

	// Create initializer
	promptRegistry := &mockPromptRegistry{}
	initializer := NewAgentInitializer(promptRegistry)

	// List available specialists
	specialists := initializer.GetAvailableSpecialists()
	fmt.Printf("Available specialist templates: %v\n", specialists)

	// Enhance the basic agent with security specialist template
	if len(specialists) > 0 {
		specialistID := specialists[0] // Use first available specialist

		fmt.Printf("\nEnhancing agent with specialist: %s\n", specialistID)

		err := initializer.EnhanceExistingAgent(ctx, "basic-agent", specialistID, guildConfig, projectPath)
		if err != nil {
			log.Fatalf("Failed to enhance agent: %v", err)
		}

		// Show enhanced agent
		enhancedAgent := &guildConfig.Agents[0]
		fmt.Printf("Enhanced agent name: %s\n", enhancedAgent.Name)

		if enhancedAgent.Backstory != nil {
			fmt.Printf("Guild rank: %s\n", enhancedAgent.Backstory.GuildRank)
			fmt.Printf("Specialties: %v\n", enhancedAgent.Backstory.Specialties)
		}

		if enhancedAgent.Specialization != nil {
			fmt.Printf("Domain: %s\n", enhancedAgent.Specialization.Domain)
			fmt.Printf("Expertise level: %s\n", enhancedAgent.Specialization.ExpertiseLevel)
		}
	}
}

// mockPromptRegistry provides a simple mock for examples
type mockPromptRegistry struct{}

// Registry interface methods
func (m *mockPromptRegistry) RegisterPrompt(role, domain, prompt string) error {
	return nil
}

func (m *mockPromptRegistry) RegisterTemplate(name, template string) error {
	return nil
}

func (m *mockPromptRegistry) GetPrompt(role, domain string) (string, error) {
	return "", nil
}

func (m *mockPromptRegistry) GetTemplate(name string) (string, error) {
	return "", nil
}

// LayeredRegistry interface methods
func (m *mockPromptRegistry) RegisterLayeredPrompt(layer layered.PromptLayer, identifier string, prompt layered.SystemPrompt) error {
	return nil
}

func (m *mockPromptRegistry) GetLayeredPrompt(layer layered.PromptLayer, identifier string) (*layered.SystemPrompt, error) {
	return &layered.SystemPrompt{}, nil
}

func (m *mockPromptRegistry) ListLayeredPrompts(layer layered.PromptLayer) ([]layered.SystemPrompt, error) {
	return nil, nil
}

func (m *mockPromptRegistry) DeleteLayeredPrompt(layer layered.PromptLayer, identifier string) error {
	return nil
}

func (m *mockPromptRegistry) GetDefaultPrompts(layer layered.PromptLayer) ([]layered.SystemPrompt, error) {
	return nil, nil
}
