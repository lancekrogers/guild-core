// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package backstory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/backstory/templates"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
)

func TestPersonalityConsistency(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	testCases := []struct {
		name             string
		agentTemplate    string
		prompts          []string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "Security Expert - Paranoid and Protective",
			agentTemplate: "security-sentinel",
			prompts: []string{
				"Should we store passwords in plain text for easier debugging?",
				"The client wants us to skip security review to meet deadline",
				"How should we handle user authentication?",
			},
			shouldContain: []string{
				"security", "secure", "protection", "never", "absolutely not",
				"vulnerability", "breach", "encryption", "guardian", "fortress",
			},
			shouldNotContain: []string{
				"maybe", "probably fine", "skip", "later", "shortcut",
			},
		},
		{
			name:          "Performance Expert - Data Driven and Enthusiastic",
			agentTemplate: "performance-artisan",
			prompts: []string{
				"This feels slow, should we optimize?",
				"The CEO thinks we need to make it faster",
				"How do we improve our system performance?",
			},
			shouldContain: []string{
				"measure", "benchmark", "profile", "metrics", "data",
				"optimization", "performance", "velocity", "speed",
			},
			shouldNotContain: []string{
				"feels", "guess", "probably", "assume", "intuition",
			},
		},
		{
			name:          "Frontend Artist - User Focused and Empathetic",
			agentTemplate: "frontend-artist",
			prompts: []string{
				"Should we add more features to this screen?",
				"The loading time is 5 seconds",
				"How should we design the user interface?",
			},
			shouldContain: []string{
				"user", "experience", "intuitive", "accessibility", "usability",
				"empathy", "inclusive", "design", "interface", "interaction",
			},
			shouldNotContain: []string{
				"doesn't matter", "users will figure it out", "just add",
			},
		},
		{
			name:          "Code Sage - Wise and Principled",
			agentTemplate: "code-sage",
			prompts: []string{
				"Should we use this new framework everyone's talking about?",
				"How do we handle this technical debt?",
				"What's the best architecture for this system?",
			},
			shouldContain: []string{
				"principle", "pattern", "maintainable", "clean", "quality",
				"wisdom", "experience", "architecture", "long-term", "craft",
			},
			shouldNotContain: []string{
				"latest trend", "quick fix", "hack", "just works",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get agent template
			template, exists := templates.SpecialistTemplates[tc.agentTemplate]
			require.True(t, exists, "Template %s should exist", tc.agentTemplate)

			// Register agent
			err := manager.RegisterAgent(template)
			require.NoError(t, err)

			// Test each prompt
			for _, prompt := range tc.prompts {
				turnContext := &layered.TurnContext{
					UserMessage: prompt,
				}

				// Build personality prompt
				enhancedPrompt, err := manager.BuildPersonalityPrompt(
					context.Background(),
					template.ID,
					"You are a helpful assistant.",
					turnContext,
				)
				require.NoError(t, err)

				enhancedLower := strings.ToLower(enhancedPrompt)

				// Check for expected personality-appropriate language
				foundExpected := false
				for _, expected := range tc.shouldContain {
					if strings.Contains(enhancedLower, strings.ToLower(expected)) {
						foundExpected = true
						break
					}
				}
				assert.True(t, foundExpected,
					"Prompt for %s should contain personality-appropriate language. Got: %s",
					tc.agentTemplate, enhancedPrompt)

				// Check for inappropriate language
				for _, unexpected := range tc.shouldNotContain {
					assert.NotContains(t, enhancedLower, strings.ToLower(unexpected),
						"Prompt contains personality-inappropriate term: %s", unexpected)
				}

				// Verify medieval guild theming is present
				assert.True(t,
					strings.Contains(enhancedLower, "guild") ||
						strings.Contains(enhancedLower, "artisan") ||
						strings.Contains(enhancedLower, "master") ||
						strings.Contains(enhancedLower, "craft"),
					"Should contain medieval guild theming")
			}
		})
	}
}

func TestPersonalityLayerGeneration(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	// Test with security expert template
	template := templates.SpecialistTemplates["security-sentinel"]
	err := manager.RegisterAgent(template)
	require.NoError(t, err)

	turnContext := &layered.TurnContext{
		UserMessage: "How should we implement user authentication?",
	}

	enhancedPrompt, err := manager.BuildPersonalityPrompt(
		context.Background(),
		template.ID,
		"Base prompt content.",
		turnContext,
	)
	require.NoError(t, err)

	// Verify all personality layers are included
	assert.Contains(t, enhancedPrompt, "Your Identity and Background")
	assert.Contains(t, enhancedPrompt, "Your Craft and Expertise")
	assert.Contains(t, enhancedPrompt, "Current Context")
	assert.Contains(t, enhancedPrompt, "Your Communication Style")
	assert.Contains(t, enhancedPrompt, "Base prompt content.")

	// Verify specific backstory elements
	assert.Contains(t, enhancedPrompt, "Sir Gareth the Vigilant")
	assert.Contains(t, enhancedPrompt, "Master Guardian")
	assert.Contains(t, enhancedPrompt, "Digital Fortress Smithing")
}

func TestContextualMoodUpdates(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	template := templates.SpecialistTemplates["performance-artisan"]
	err := manager.RegisterAgent(template)
	require.NoError(t, err)

	// Test different contexts that should affect mood
	testCases := []struct {
		message      string
		expectedMood string
	}{
		{
			message:      "We have a critical performance problem!",
			expectedMood: "concerned",
		},
		{
			message:      "Can you help me understand how this optimization works?",
			expectedMood: "helpful",
		},
		{
			message:      "Let's build a new high-performance system from scratch",
			expectedMood: "excited",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.message, func(t *testing.T) {
			turnContext := &layered.TurnContext{
				UserMessage: tc.message,
				Urgency:     "normal",
			}

			_, err := manager.BuildPersonalityPrompt(
				context.Background(),
				template.ID,
				"Base prompt.",
				turnContext,
			)
			require.NoError(t, err)

			// Check agent's mood was updated
			agent := manager.agents[template.ID]
			assert.Equal(t, tc.expectedMood, agent.Context.Mood)
		})
	}
}

func TestMemoryAndLearning(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	template := templates.SpecialistTemplates["frontend-artist"]
	err := manager.RegisterAgent(template)
	require.NoError(t, err)

	// Record some interactions
	interactions := []Interaction{
		{
			Type:       "design_feedback",
			Summary:    "User loved the accessible navigation design",
			Outcome:    "success",
			UserRating: 9,
			Timestamp:  time.Now().Add(-1 * time.Hour),
		},
		{
			Type:       "usability_test",
			Summary:    "Users struggled with the complex form layout",
			Outcome:    "needs_improvement",
			UserRating: 4,
			Timestamp:  time.Now().Add(-2 * time.Hour),
		},
	}

	for _, interaction := range interactions {
		err := manager.RecordInteraction(template.ID, interaction)
		require.NoError(t, err)
	}

	// Test prompt generation includes relevant memories
	turnContext := &layered.TurnContext{
		UserMessage: "How should we design this navigation menu?",
	}

	enhancedPrompt, err := manager.BuildPersonalityPrompt(
		context.Background(),
		template.ID,
		"Design a navigation menu.",
		turnContext,
	)
	require.NoError(t, err)

	// Should include relevant experience about navigation
	assert.Contains(t, enhancedPrompt, "navigation")
	assert.Contains(t, enhancedPrompt, "Relevant Experience")
}

func TestTeamCollaboration(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	// Register multiple agents
	securityAgent := templates.SpecialistTemplates["security-sentinel"]
	perfAgent := templates.SpecialistTemplates["performance-artisan"]

	err := manager.RegisterAgent(securityAgent)
	require.NoError(t, err)
	err = manager.RegisterAgent(perfAgent)
	require.NoError(t, err)

	// Set team context
	err = manager.UpdateTeamContext(securityAgent.ID, []string{perfAgent.ID})
	require.NoError(t, err)

	turnContext := &layered.TurnContext{
		UserMessage: "Should we cache user session data?",
	}

	enhancedPrompt, err := manager.BuildPersonalityPrompt(
		context.Background(),
		securityAgent.ID,
		"Consider caching strategy.",
		turnContext,
	)
	require.NoError(t, err)

	// Should mention team collaboration
	assert.Contains(t, enhancedPrompt, "Working alongside")
	assert.Contains(t, enhancedPrompt, "collaborating")
}

func TestGuildMasterPersonality(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	guildMaster := templates.CreateMedievalGuildMaster()
	err := manager.RegisterAgent(guildMaster)
	require.NoError(t, err)

	turnContext := &layered.TurnContext{
		UserMessage: "How should we organize the team for this project?",
	}

	enhancedPrompt, err := manager.BuildPersonalityPrompt(
		context.Background(),
		guildMaster.ID,
		"Plan the project organization.",
		turnContext,
	)
	require.NoError(t, err)

	// Verify guild master characteristics
	assert.Contains(t, enhancedPrompt, "Master Aldric the Wise")
	assert.Contains(t, enhancedPrompt, "Grand Master")
	assert.Contains(t, enhancedPrompt, "Guild Leadership")

	// Should emphasize leadership and wisdom
	promptLower := strings.ToLower(enhancedPrompt)
	assert.True(t,
		strings.Contains(promptLower, "wisdom") ||
			strings.Contains(promptLower, "leadership") ||
			strings.Contains(promptLower, "team"),
		"Guild master should emphasize leadership qualities")
}

func TestBackstoryValidation(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	// Test agent without backstory
	simpleAgent := &config.AgentConfig{
		ID:           "simple-agent",
		Name:         "Simple Agent",
		Type:         "worker",
		Provider:     "mock",
		Model:        "test-model",
		Capabilities: []string{"basic_task"},
	}

	err := manager.RegisterAgent(simpleAgent)
	require.NoError(t, err)

	// Should work without backstory
	enhancedPrompt, err := manager.BuildPersonalityPrompt(
		context.Background(),
		simpleAgent.ID,
		"Base prompt.",
		nil,
	)
	require.NoError(t, err)
	// Even simple agents get basic context, but no identity/expertise layers
	assert.Contains(t, enhancedPrompt, "Base prompt.")
	assert.NotContains(t, enhancedPrompt, "Your Identity and Background")
	assert.NotContains(t, enhancedPrompt, "Your Craft and Expertise")
}

func TestBackstoryRegistryIntegration(t *testing.T) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	template := templates.SpecialistTemplates["code-sage"]
	err := manager.RegisterAgent(template)
	require.NoError(t, err)

	// Verify role layer was registered
	prompts, err := registry.ListLayeredPrompts(layered.LayerRole)
	require.NoError(t, err)
	assert.Len(t, prompts, 1)
	assert.Equal(t, template.ID, prompts[0].ArtisanID)
	assert.Contains(t, prompts[0].Content, template.Name)
}

// Benchmark personality prompt generation
func BenchmarkPersonalityPromptGeneration(b *testing.B) {
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	template := templates.SpecialistTemplates["security-sentinel"]
	err := manager.RegisterAgent(template)
	require.NoError(b, err)

	turnContext := &layered.TurnContext{
		UserMessage: "How should we implement authentication?",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.BuildPersonalityPrompt(
			context.Background(),
			template.ID,
			"Base prompt content.",
			turnContext,
		)
		require.NoError(b, err)
	}
}
