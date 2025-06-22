// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agents

import (
	"context"
	"testing"
	"time"
)

func TestCraftElennaGuildMaster_Creation(t *testing.T) {
	creator := NewDefaultAgentCreator()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	elena, err := creator.CreateElenaGuildMaster(ctx)
	if err != nil {
		t.Fatalf("Failed to create Elena: %v", err)
	}

	// Verify Elena's core properties
	if elena.ID != "elena-guild-master" {
		t.Errorf("Expected ID 'elena-guild-master', got '%s'", elena.ID)
	}

	if elena.Name != "Elena the Guild Master" {
		t.Errorf("Expected name 'Elena the Guild Master', got '%s'", elena.Name)
	}

	if elena.Type != "manager" {
		t.Errorf("Expected type 'manager', got '%s'", elena.Type)
	}

	if elena.Provider != "claude_code" {
		t.Errorf("Expected provider 'claude_code', got '%s'", elena.Provider)
	}

	// Verify backstory is present and rich
	if elena.Backstory == nil {
		t.Fatal("Elena should have a backstory")
	}

	if elena.Backstory.Experience == "" {
		t.Error("Elena should have experience defined")
	}

	if len(elena.Backstory.PreviousRoles) == 0 {
		t.Error("Elena should have previous roles defined")
	}

	if elena.Backstory.Philosophy == "" {
		t.Error("Elena should have a philosophy defined")
	}

	if elena.Backstory.GuildRank != "Guild Master" {
		t.Errorf("Expected guild rank 'Guild Master', got '%s'", elena.Backstory.GuildRank)
	}

	// Verify personality is present
	if elena.Personality == nil {
		t.Fatal("Elena should have personality defined")
	}

	if elena.Personality.Empathy != 10 {
		t.Errorf("Expected empathy 10, got %d", elena.Personality.Empathy)
	}

	if len(elena.Personality.Traits) == 0 {
		t.Error("Elena should have personality traits defined")
	}

	// Verify specialization
	if elena.Specialization == nil {
		t.Fatal("Elena should have specialization defined")
	}

	if elena.Specialization.Domain != "project coordination and team leadership" {
		t.Errorf("Unexpected specialization domain: %s", elena.Specialization.Domain)
	}

	// Verify capabilities
	expectedCapabilities := []string{
		"project_management",
		"team_coordination",
		"strategic_planning",
		"stakeholder_management",
	}

	for _, expectedCap := range expectedCapabilities {
		found := false
		for _, cap := range elena.Capabilities {
			if cap == expectedCap {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Elena should have capability '%s'", expectedCap)
		}
	}
}

func TestCraftMarcusDeveloper_Creation(t *testing.T) {
	creator := NewDefaultAgentCreator()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	marcus, err := creator.CreateDefaultDeveloper(ctx)
	if err != nil {
		t.Fatalf("Failed to create Marcus: %v", err)
	}

	// Verify Marcus's core properties
	if marcus.ID != "marcus-developer" {
		t.Errorf("Expected ID 'marcus-developer', got '%s'", marcus.ID)
	}

	if marcus.Name != "Marcus the Code Artisan" {
		t.Errorf("Expected name 'Marcus the Code Artisan', got '%s'", marcus.Name)
	}

	if marcus.Type != "worker" {
		t.Errorf("Expected type 'worker', got '%s'", marcus.Type)
	}

	if marcus.Provider != "claude_code" {
		t.Errorf("Expected provider 'claude_code', got '%s'", marcus.Provider)
	}

	// Verify backstory and personality
	if marcus.Backstory == nil {
		t.Fatal("Marcus should have a backstory")
	}

	if marcus.Personality == nil {
		t.Fatal("Marcus should have personality defined")
	}

	if marcus.Personality.Craftsmanship != 10 {
		t.Errorf("Expected craftsmanship 10, got %d", marcus.Personality.Craftsmanship)
	}
}

func TestCraftVeraTester_Creation(t *testing.T) {
	creator := NewDefaultAgentCreator()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vera, err := creator.CreateDefaultTester(ctx)
	if err != nil {
		t.Fatalf("Failed to create Vera: %v", err)
	}

	// Verify Vera's core properties
	if vera.ID != "vera-tester" {
		t.Errorf("Expected ID 'vera-tester', got '%s'", vera.ID)
	}

	if vera.Name != "Vera the Quality Guardian" {
		t.Errorf("Expected name 'Vera the Quality Guardian', got '%s'", vera.Name)
	}

	if vera.Type != "specialist" {
		t.Errorf("Expected type 'specialist', got '%s'", vera.Type)
	}

	if vera.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", vera.Provider)
	}

	// Verify backstory and specialization
	if vera.Backstory == nil {
		t.Fatal("Vera should have a backstory")
	}

	if vera.Backstory.GuildRank != "Master Guardian" {
		t.Errorf("Expected guild rank 'Master Guardian', got '%s'", vera.Backstory.GuildRank)
	}

	if vera.Specialization == nil {
		t.Fatal("Vera should have specialization defined")
	}

	if vera.Specialization.Domain != "quality assurance and testing" {
		t.Errorf("Unexpected specialization domain: %s", vera.Specialization.Domain)
	}
}

func TestCraftDefaultAgentSet_Complete(t *testing.T) {
	creator := NewDefaultAgentCreator()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agents, err := creator.CreateDefaultAgentSet(ctx)
	if err != nil {
		t.Fatalf("Failed to create default agent set: %v", err)
	}

	// Should have 3 agents: Elena, Marcus, Vera
	if len(agents) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(agents))
	}

	// Verify all agents have unique IDs
	idSet := make(map[string]bool)
	for _, agent := range agents {
		if idSet[agent.ID] {
			t.Errorf("Duplicate agent ID found: %s", agent.ID)
		}
		idSet[agent.ID] = true
	}

	// Verify we have the expected agents
	expectedIDs := []string{"elena-guild-master", "marcus-developer", "vera-tester"}
	for _, expectedID := range expectedIDs {
		if !idSet[expectedID] {
			t.Errorf("Expected agent '%s' not found in set", expectedID)
		}
	}

	// Verify all agents have backstories and personalities
	for _, agent := range agents {
		if agent.Backstory == nil {
			t.Errorf("Agent '%s' should have backstory", agent.ID)
		}
		if agent.Personality == nil {
			t.Errorf("Agent '%s' should have personality", agent.ID)
		}
		if len(agent.Capabilities) == 0 {
			t.Errorf("Agent '%s' should have capabilities", agent.ID)
		}
	}
}

func TestGuildOptimalProvider_Mapping(t *testing.T) {
	creator := NewDefaultAgentCreator()

	testCases := []struct {
		agentType string
		agentID   string
		expected  string
	}{
		{"manager", "elena-guild-master", "claude_code"},
		{"worker", "marcus-developer", "claude_code"},
		{"worker", "some-other-worker", "anthropic"},
		{"specialist", "vera-tester", "anthropic"},
		{"unknown", "random-agent", "anthropic"},
	}

	for _, tc := range testCases {
		result := creator.GetOptimalProvider(tc.agentType, tc.agentID)
		if result != tc.expected {
			t.Errorf("For %s/%s, expected provider '%s', got '%s'", 
				tc.agentType, tc.agentID, tc.expected, result)
		}
	}
}

func TestGuildSpecialistTemplates_Access(t *testing.T) {
	creator := NewDefaultAgentCreator()

	// Test listing available specialists
	specialists := creator.ListAvailableSpecialists()
	if len(specialists) == 0 {
		t.Error("Should have available specialists")
	}

	// Test getting a specific template
	if len(specialists) > 0 {
		template, err := creator.GetSpecialistTemplate(specialists[0])
		if err != nil {
			t.Errorf("Failed to get specialist template: %v", err)
		}

		if template == nil {
			t.Error("Template should not be nil")
		}
	}

	// Test getting non-existent template
	_, err := creator.GetSpecialistTemplate("non-existent-specialist")
	if err == nil {
		t.Error("Should return error for non-existent specialist")
	}
}

func TestCraftAgentContext_Cancellation(t *testing.T) {
	creator := NewDefaultAgentCreator()
	
	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := creator.CreateElenaGuildMaster(ctx)
	if err == nil {
		t.Error("Should return error for cancelled context")
	}

	_, err = creator.CreateDefaultDeveloper(ctx)
	if err == nil {
		t.Error("Should return error for cancelled context")
	}

	_, err = creator.CreateDefaultTester(ctx)
	if err == nil {
		t.Error("Should return error for cancelled context")
	}

	_, err = creator.CreateDefaultAgentSet(ctx)
	if err == nil {
		t.Error("Should return error for cancelled context")
	}
}