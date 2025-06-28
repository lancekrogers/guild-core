// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAgentRegistry for testing
type MockAgentRegistry struct {
	mock.Mock
}

func (m *MockAgentRegistry) RegisterAgentType(name string, factory registry.AgentFactory) error {
	args := m.Called(name, factory)
	return args.Error(0)
}

func (m *MockAgentRegistry) GetAgent(agentType string) (registry.Agent, error) {
	args := m.Called(agentType)
	if agent := args.Get(0); agent != nil {
		return agent.(registry.Agent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAgentRegistry) ListAgentTypes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockAgentRegistry) HasAgentType(agentType string) bool {
	args := m.Called(agentType)
	return args.Bool(0)
}

func (m *MockAgentRegistry) GetAgentsByCost(maxCost int) []registry.AgentInfo {
	args := m.Called(maxCost)
	return args.Get(0).([]registry.AgentInfo)
}

func (m *MockAgentRegistry) GetCheapestAgentByCapability(capability string) (*registry.AgentInfo, error) {
	args := m.Called(capability)
	if info := args.Get(0); info != nil {
		return info.(*registry.AgentInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAgentRegistry) GetAgentsByCapability(capability string) []registry.AgentInfo {
	args := m.Called(capability)
	return args.Get(0).([]registry.AgentInfo)
}

func (m *MockAgentRegistry) RegisterGuildAgent(config registry.GuildAgentConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockAgentRegistry) GetRegisteredAgents() []registry.GuildAgentConfig {
	args := m.Called()
	return args.Get(0).([]registry.GuildAgentConfig)
}

// MockComponentRegistry for testing
type MockComponentRegistry struct {
	mock.Mock
	agentRegistry   *MockAgentRegistry
	storageRegistry *MockStorageRegistry
}

func (m *MockComponentRegistry) Agents() registry.AgentRegistry {
	return m.agentRegistry
}

func (m *MockComponentRegistry) Storage() registry.StorageRegistry {
	return m.storageRegistry
}

func (m *MockComponentRegistry) Tools() registry.ToolRegistry {
	return nil
}

func (m *MockComponentRegistry) Providers() registry.ProviderRegistry {
	return nil
}

func (m *MockComponentRegistry) Memory() registry.MemoryRegistry {
	return nil
}

func (m *MockComponentRegistry) Prompts() *registry.PromptRegistry {
	return nil
}

func (m *MockComponentRegistry) Project() registry.ProjectRegistry {
	return nil
}

func (m *MockComponentRegistry) GetAgentsByCost(maxCost int) []registry.AgentInfo {
	args := m.Called(maxCost)
	return args.Get(0).([]registry.AgentInfo)
}

func (m *MockComponentRegistry) GetToolsByCost(maxCost int) []registry.ToolInfo {
	args := m.Called(maxCost)
	return args.Get(0).([]registry.ToolInfo)
}

func (m *MockComponentRegistry) GetPromptManager() (registry.LayeredPromptManager, error) {
	return nil, nil
}

func (m *MockComponentRegistry) Orchestrator() interface{} {
	return nil
}

func (m *MockComponentRegistry) Initialize(ctx context.Context, config registry.Config) error {
	return nil
}

func (m *MockComponentRegistry) Shutdown(ctx context.Context) error {
	return nil
}

func (m *MockComponentRegistry) GetCheapestAgentByCapability(capability string) (*registry.AgentInfo, error) {
	return nil, nil
}

func (m *MockComponentRegistry) GetCheapestToolByCapability(capability string) (*registry.ToolInfo, error) {
	return nil, nil
}

func (m *MockComponentRegistry) GetAgentsByCapability(capability string) []registry.AgentInfo {
	return nil
}

// MockStorageRegistry for testing
type MockStorageRegistry struct {
	mock.Mock
}

// Additional methods to satisfy StorageRegistry interface
func (m *MockStorageRegistry) RegisterTaskRepository(repo registry.TaskRepository) error {
	return nil
}

func (m *MockStorageRegistry) RegisterCampaignRepository(repo registry.CampaignRepository) error {
	return nil
}

func (m *MockStorageRegistry) RegisterCommissionRepository(repo registry.CommissionRepository) error {
	return nil
}

func (m *MockStorageRegistry) RegisterAgentRepository(repo registry.AgentRepository) error {
	return nil
}

func (m *MockStorageRegistry) RegisterPromptChainRepository(repo registry.PromptChainRepository) error {
	return nil
}

func (m *MockStorageRegistry) GetBoardRepository() registry.KanbanBoardRepository {
	return nil
}

func (m *MockStorageRegistry) GetAgentRepository() registry.AgentRepository {
	args := m.Called()
	if repo := args.Get(0); repo != nil {
		return repo.(registry.AgentRepository)
	}
	return nil
}

func (m *MockStorageRegistry) GetCampaignRepository() registry.CampaignRepository {
	return nil
}

func (m *MockStorageRegistry) GetCommissionRepository() registry.CommissionRepository {
	return nil
}

func (m *MockStorageRegistry) GetTaskRepository() registry.TaskRepository {
	return nil
}

func (m *MockStorageRegistry) GetPromptChainRepository() registry.PromptChainRepository {
	return nil
}

func (m *MockStorageRegistry) GetMemoryStore() registry.MemoryStore {
	return nil
}

func (m *MockStorageRegistry) GetKanbanTaskRepository() registry.KanbanTaskRepository {
	return nil
}

func (m *MockStorageRegistry) GetKanbanCampaignRepository() registry.KanbanCampaignRepository {
	return nil
}

func (m *MockStorageRegistry) GetKanbanCommissionRepository() registry.KanbanCommissionRepository {
	return nil
}

func (m *MockStorageRegistry) RegisterSessionRepository(repo registry.SessionRepository) error {
	return nil
}

func (m *MockStorageRegistry) GetSessionRepository() registry.SessionRepository {
	return nil
}

// MockAgentRepository for testing
type MockAgentRepository struct {
	mock.Mock
}

func (m *MockAgentRepository) CreateAgent(ctx context.Context, agent *registry.StorageAgent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockAgentRepository) GetAgent(ctx context.Context, id string) (*registry.StorageAgent, error) {
	args := m.Called(ctx, id)
	if result := args.Get(0); result != nil {
		return result.(*registry.StorageAgent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAgentRepository) ListAgents(ctx context.Context) ([]*registry.StorageAgent, error) {
	args := m.Called(ctx)
	if result := args.Get(0); result != nil {
		return result.([]*registry.StorageAgent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAgentRepository) UpdateAgent(ctx context.Context, agent *registry.StorageAgent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentRepository) ListAgentsByType(ctx context.Context, agentType string) ([]*registry.StorageAgent, error) {
	args := m.Called(ctx, agentType)
	if result := args.Get(0); result != nil {
		return result.([]*registry.StorageAgent), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestAgentListCommand(t *testing.T) {
	// Test that the command structure and flags are correct
	t.Run("command structure", func(t *testing.T) {
		assert.NotNil(t, agentListCmd)
		assert.Equal(t, "list", agentListCmd.Use)
		assert.Contains(t, agentListCmd.Short, "List all available agents")
		assert.NotNil(t, agentListCmd.RunE)

		// Check flags exist
		assert.NotNil(t, agentListCmd.Flags().Lookup("verbose"))
		assert.NotNil(t, agentListCmd.Flags().Lookup("type"))
		assert.NotNil(t, agentListCmd.Flags().Lookup("max-cost"))
	})

	// The actual display logic is tested in TestDisplayCompactAgentList and TestDisplayVerboseAgentList
	// The command integration requires a real registry, so we test the display functions directly
}

func TestDisplayCompactAgentList(t *testing.T) {
	agents := []registry.AgentInfo{
		{
			ID:            "agent1",
			Name:          "Code Writer",
			Type:          "developer",
			CostMagnitude: 2,
			Capabilities:  []string{"golang", "testing", "documentation"},
		},
		{
			ID:            "agent2",
			Name:          "Security Auditor",
			Type:          "security",
			CostMagnitude: 3,
			Capabilities:  []string{"vulnerability-scanning", "penetration-testing", "security-best-practices", "compliance-checks"},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	displayCompactAgentList(agents)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	assert.Contains(t, output, "Guild Agents")
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "COST")
	assert.Contains(t, output, "agent1")
	assert.Contains(t, output, "Code Writer")
	assert.Contains(t, output, "developer")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "agent2")
	assert.Contains(t, output, "Security Auditor")
	assert.Contains(t, output, "security")
	assert.Contains(t, output, "3")
	assert.Contains(t, output, "Total: 2 agents")

	// Check that long capabilities are truncated
	assert.Contains(t, output, "vulnerability-scanning, penetration-t...")
}

func TestDisplayVerboseAgentList(t *testing.T) {
	agents := []registry.AgentInfo{
		{
			ID:            "agent1",
			Name:          "Code Writer",
			Type:          "developer",
			CostMagnitude: 2,
			Capabilities:  []string{"golang", "testing"},
		},
	}

	agentTypes := []string{"developer", "reviewer", "security"}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	displayVerboseAgentList(agents, agentTypes)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	assert.Contains(t, output, "Guild Agents (Detailed View)")
	assert.Contains(t, output, "Code Writer (ID: agent1)")
	assert.Contains(t, output, "Type: developer")
	assert.Contains(t, output, "Cost: 2")
	assert.Contains(t, output, "Capabilities:")
	assert.Contains(t, output, "• golang")
	assert.Contains(t, output, "• testing")
	assert.Contains(t, output, "Total: 1 agents")
	assert.Contains(t, output, "Available Types: developer, reviewer, security")
}

func TestGetAgentCostIcon(t *testing.T) {
	tests := []struct {
		cost     int
		expected string
	}{
		{0, "💰"},
		{1, "💰"},
		{2, "💰💰"},
		{3, "💰💰"},
		{4, "💰💰💰"},
		{5, "💰💰💰"},
		{6, "💰💰💰💰"},
		{10, "💰💰💰💰"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("cost_%d", tt.cost), func(t *testing.T) {
			result := getAgentCostIcon(tt.cost)
			assert.Equal(t, tt.expected, result)
		})
	}
}
