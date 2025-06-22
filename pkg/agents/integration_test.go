// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
)

// Integration test showing how enhanced agents work with Guild's backstory system
func TestGuildBackstoryIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a mock prompt registry
	promptRegistry := &mockLayeredRegistry{}
	
	// Create agent initializer with backstory integration
	initializer := NewAgentInitializer(promptRegistry)

	// Create Elena
	creator := NewDefaultAgentCreator()
	elena, err := creator.CreateElenaGuildMaster(ctx)
	if err != nil {
		t.Fatalf("Failed to create Elena: %v", err)
	}

	// Register Elena with backstory manager
	backstoryManager := initializer.GetBackstoryManager()
	if err := backstoryManager.RegisterAgent(elena); err != nil {
		t.Fatalf("Failed to register Elena with backstory manager: %v", err)
	}

	// Create turn context for personality prompt generation
	turnContext := &layered.TurnContext{
		UserMessage: "Help me coordinate this complex multi-team project",
		TaskID:      "project-coordination",
		Urgency:     "high",
	}

	basePrompt := "Please help the user with their project coordination request."

	// Generate personality-enhanced prompt
	enhancedPrompt, err := backstoryManager.BuildPersonalityPrompt(
		ctx, elena.ID, basePrompt, turnContext)
	if err != nil {
		t.Fatalf("Failed to generate personality prompt: %v", err)
	}

	// Verify the enhanced prompt includes personality elements
	if enhancedPrompt == basePrompt {
		t.Error("Enhanced prompt should be different from base prompt")
	}

	if len(enhancedPrompt) <= len(basePrompt) {
		t.Error("Enhanced prompt should be longer than base prompt")
	}

	// The enhanced prompt should contain Elena's identity
	if !contains(enhancedPrompt, "Elena") {
		t.Error("Enhanced prompt should contain Elena's name")
	}

	// Should contain her role
	if !contains(enhancedPrompt, "Guild Master") {
		t.Error("Enhanced prompt should contain her guild rank")
	}
}

// Test complete agent set creation and backstory registration
func TestGuildCompleteAgentSetIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	promptRegistry := &mockLayeredRegistry{}
	initializer := NewAgentInitializer(promptRegistry)

	// Create complete agent set
	creator := NewDefaultAgentCreator()
	agents, err := creator.CreateDefaultAgentSet(ctx)
	if err != nil {
		t.Fatalf("Failed to create agent set: %v", err)
	}

	// Register all agents with backstory manager
	backstoryManager := initializer.GetBackstoryManager()
	for _, agent := range agents {
		if err := backstoryManager.RegisterAgent(agent); err != nil {
			t.Fatalf("Failed to register agent %s: %v", agent.ID, err)
		}
	}

	// Test personality prompt generation for each agent
	turnContext := &layered.TurnContext{
		UserMessage: "I need assistance with this task",
		TaskID:      "general-task",
	}

	basePrompt := "Please help the user."

	for _, agent := range agents {
		enhancedPrompt, err := backstoryManager.BuildPersonalityPrompt(
			ctx, agent.ID, basePrompt, turnContext)
		if err != nil {
			t.Errorf("Failed to generate prompt for %s: %v", agent.ID, err)
			continue
		}

		// Enhanced prompt should include agent's personality
		if !contains(enhancedPrompt, agent.Name) {
			t.Errorf("Enhanced prompt for %s should contain agent name", agent.ID)
		}
	}
}

// Test guild configuration creation with Elena
func TestGuildConfigCreationWithElena(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	promptRegistry := &mockLayeredRegistry{}
	initializer := NewAgentInitializer(promptRegistry)

	// Create guild config with Elena as manager
	guildConfig, err := initializer.CreateGuildConfigWithElena(ctx, "test-guild")
	if err != nil {
		t.Fatalf("Failed to create guild config: %v", err)
	}

	// Verify guild structure
	if guildConfig.Name != "test-guild" {
		t.Errorf("Expected guild name 'test-guild', got '%s'", guildConfig.Name)
	}

	if guildConfig.Manager.Default != "elena-guild-master" {
		t.Errorf("Expected default manager 'elena-guild-master', got '%s'", guildConfig.Manager.Default)
	}

	if len(guildConfig.Agents) != 3 {
		t.Errorf("Expected 3 agents, got %d", len(guildConfig.Agents))
	}

	// Verify Elena is in the agents list
	elenaFound := false
	for _, agent := range guildConfig.Agents {
		if agent.ID == "elena-guild-master" {
			elenaFound = true
			// Verify Elena has backstory
			if agent.Backstory == nil {
				t.Error("Elena should have backstory in guild config")
			}
			if agent.Personality == nil {
				t.Error("Elena should have personality in guild config")
			}
			break
		}
	}

	if !elenaFound {
		t.Error("Elena should be in the guild agents list")
	}
}

// Test upgrading existing simple guild
func TestGuildUpgradeExisting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create simple guild config (like current defaults)
	simpleGuild := &config.GuildConfig{
		Name:        "simple-guild",
		Description: "Basic guild without personalities",
		Agents: []config.AgentConfig{
			{
				ID:           "basic-manager",
				Name:         "Basic Manager",
				Type:         "manager",
				Provider:     "anthropic",
				Model:        "claude-3-sonnet-20240229",
				Capabilities: []string{"task_breakdown"},
			},
		},
	}

	originalAgentCount := len(simpleGuild.Agents)
	originalManager := simpleGuild.Manager.Default

	// Upgrade the guild
	promptRegistry := &mockLayeredRegistry{}
	initializer := NewAgentInitializer(promptRegistry)

	err := initializer.UpgradeExistingGuild(ctx, simpleGuild, tempDir)
	if err != nil {
		t.Fatalf("Failed to upgrade guild: %v", err)
	}

	// Verify upgrade results
	if len(simpleGuild.Agents) <= originalAgentCount {
		t.Error("Upgraded guild should have more agents")
	}

	if simpleGuild.Manager.Default == originalManager {
		t.Error("Manager should be updated after upgrade")
	}

	if simpleGuild.Manager.Default != "elena-guild-master" {
		t.Errorf("Expected manager 'elena-guild-master', got '%s'", simpleGuild.Manager.Default)
	}

	// Verify Elena was added
	elenaFound := false
	for _, agent := range simpleGuild.Agents {
		if agent.ID == "elena-guild-master" {
			elenaFound = true
			break
		}
	}

	if !elenaFound {
		t.Error("Elena should be added during upgrade")
	}
}

// Test file operations and persistence
func TestGuildAgentPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create temporary directory
	tempDir := t.TempDir()

	promptRegistry := &mockLayeredRegistry{}
	initializer := NewAgentInitializer(promptRegistry)

	// Initialize default agents in temp directory
	err := initializer.InitializeDefaultAgents(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize default agents: %v", err)
	}

	// Verify agent files were created
	agentsDir := filepath.Join(tempDir, ".campaign", "agents")
	
	// Check if agents directory exists
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		t.Fatal("Agents directory should be created")
	}

	// Check for Elena's file
	elenaFile := filepath.Join(agentsDir, "elena-guild-master.yaml")
	if _, err := os.Stat(elenaFile); os.IsNotExist(err) {
		t.Error("Elena's config file should be created")
	}

	// Check for Marcus's file
	marcusFile := filepath.Join(agentsDir, "marcus-developer.yaml")
	if _, err := os.Stat(marcusFile); os.IsNotExist(err) {
		t.Error("Marcus's config file should be created")
	}

	// Check for Vera's file
	veraFile := filepath.Join(agentsDir, "vera-tester.yaml")
	if _, err := os.Stat(veraFile); os.IsNotExist(err) {
		t.Error("Vera's config file should be created")
	}
}

// Mock implementation of layered registry for testing
type mockLayeredRegistry struct{}

// Registry interface methods
func (m *mockLayeredRegistry) RegisterPrompt(role, domain, prompt string) error {
	return nil
}

func (m *mockLayeredRegistry) RegisterTemplate(name, template string) error {
	return nil
}

func (m *mockLayeredRegistry) GetPrompt(role, domain string) (string, error) {
	return "", nil
}

func (m *mockLayeredRegistry) GetTemplate(name string) (string, error) {
	return "", nil
}

// LayeredRegistry interface methods
func (m *mockLayeredRegistry) RegisterLayeredPrompt(layer layered.PromptLayer, identifier string, prompt layered.SystemPrompt) error {
	return nil
}

func (m *mockLayeredRegistry) GetLayeredPrompt(layer layered.PromptLayer, identifier string) (*layered.SystemPrompt, error) {
	return &layered.SystemPrompt{}, nil
}

func (m *mockLayeredRegistry) ListLayeredPrompts(layer layered.PromptLayer) ([]layered.SystemPrompt, error) {
	return nil, nil
}

func (m *mockLayeredRegistry) DeleteLayeredPrompt(layer layered.PromptLayer, identifier string) error {
	return nil
}

func (m *mockLayeredRegistry) GetDefaultPrompts(layer layered.PromptLayer) ([]layered.SystemPrompt, error) {
	return nil, nil
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}